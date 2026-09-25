package modules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

type MySQLDBModule struct {
	*BaseExecutorModule
}

type MySQLDBInfo struct {
	Name      string `json:"name"`
	Charset   string `json:"charset"`
	Collation string `json:"collation"`
	Exists    bool   `json:"exists"`
}

func NewMySQLDBModule() *MySQLDBModule {
	return &MySQLDBModule{
		BaseExecutorModule: NewBaseExecutorModule("mysql_db"),
	}
}

// GetDescription returns the module description
func (m *MySQLDBModule) GetDescription() string {
	return "Manage MySQL databases"
}

func (m *MySQLDBModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()
	result := types.TaskResult{
		TaskName:  "mysql_db",
		Host:      host.Name,
		Module:    m.GetName(),
		Success:   true,
		Output:    make(map[string]interface{}),
		Timestamp: startTime,
	}
	fail := func(err error) (types.TaskResult, error) {
		result.Success = false
		result.Error = err.Error()
		result.Duration = time.Since(startTime)
		return result, err
	}

	dbName, _ := args["name"].(string)
	if dbName == "" {
		return fail(fmt.Errorf("database name is required"))
	}
	state, _ := args["state"].(string)
	if state == "" {
		state = "present"
	}
	target, _ := args["target"].(string)
	if (state == "dump" || state == "import") && target == "" {
		return fail(fmt.Errorf("target file is required for %s", state))
	}

	exec, err := m.CreateExecutor(host)
	if err != nil {
		return fail(fmt.Errorf("failed to create executor: %w", err))
	}
	defer exec.Close()

	conn := dbConnFromArgs(args)
	info, err := m.databaseInfo(exec, conn, dbName)
	if err != nil {
		return fail(fmt.Errorf("failed to check database: %w", err))
	}

	switch state {
	case "present":
		if info == nil {
			charset, _ := args["charset"].(string)
			if charset == "" {
				charset = "utf8mb4"
			}
			collation, _ := args["collation"].(string)
			if collation == "" {
				collation = "utf8mb4_unicode_ci"
			}
			sql := fmt.Sprintf("CREATE DATABASE %s CHARACTER SET %s COLLATE %s",
				mysqlIdent(dbName), mysqlString(charset), mysqlString(collation))
			if _, err := exec.Execute(conn.mysql(sql)); err != nil {
				return fail(fmt.Errorf("failed to create database: %w", err))
			}
			result.Changed = true
			result.Output["action"] = "created"
		}

	case "absent":
		if info != nil {
			if _, err := exec.Execute(conn.mysql("DROP DATABASE " + mysqlIdent(dbName))); err != nil {
				return fail(fmt.Errorf("failed to drop database: %w", err))
			}
			result.Changed = true
			result.Output["action"] = "dropped"
		}

	case "dump":
		cmd := conn.mysqlClient("mysqldump") + " " + shellQuote(dbName) + " > " + shellQuote(target)
		if _, err := exec.Execute(cmd); err != nil {
			return fail(fmt.Errorf("failed to dump database: %w", err))
		}
		result.Changed = true
		result.Output["action"] = "dumped"

	case "import":
		cmd := conn.mysqlClient("mysql") + " " + shellQuote(dbName) + " < " + shellQuote(target)
		if _, err := exec.Execute(cmd); err != nil {
			return fail(fmt.Errorf("failed to import database: %w", err))
		}
		result.Changed = true
		result.Output["action"] = "imported"

	default:
		return fail(fmt.Errorf("unsupported state %q", state))
	}

	if state != "absent" {
		if info, _ := m.databaseInfo(exec, conn, dbName); info != nil {
			result.Output["database"] = info
		}
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// databaseInfo returns nil when the database does not exist
func (m *MySQLDBModule) databaseInfo(exec *executor.CommandExecutor, conn dbConn, dbName string) (*MySQLDBInfo, error) {
	sql := "SELECT SCHEMA_NAME, DEFAULT_CHARACTER_SET_NAME, DEFAULT_COLLATION_NAME FROM information_schema.SCHEMATA WHERE SCHEMA_NAME = " + mysqlString(dbName)
	out, err := exec.Execute(conn.mysql(sql))
	if err != nil {
		return nil, err
	}
	fields := strings.Split(strings.TrimSpace(out), "\t")
	if len(fields) < 3 {
		return nil, nil
	}
	return &MySQLDBInfo{Name: fields[0], Charset: fields[1], Collation: fields[2], Exists: true}, nil
}

// Validate validates mysql_db module arguments
func (m *MySQLDBModule) Validate(args map[string]interface{}) error {
	return requireStringArg(args, "name")
}
