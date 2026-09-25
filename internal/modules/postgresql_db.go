package modules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

type PostgreSQLDBModule struct {
	*BaseExecutorModule
}

func NewPostgreSQLDBModule() *PostgreSQLDBModule {
	m := &PostgreSQLDBModule{
		BaseExecutorModule: NewBaseExecutorModule("postgresql_db"),
	}
	m.description = "Manage PostgreSQL databases"
	return m
}

func (m *PostgreSQLDBModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()
	result := types.TaskResult{
		TaskName: "postgresql_db", Host: host.Name, Module: m.GetName(),
		Success: true, Output: make(map[string]interface{}), Timestamp: startTime,
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
	if (state == "dump" || state == "restore") && target == "" {
		return fail(fmt.Errorf("target file is required for %s", state))
	}

	exec, err := m.CreateExecutor(host)
	if err != nil {
		return fail(fmt.Errorf("failed to create executor: %w", err))
	}
	defer exec.Close()

	conn := dbConnFromArgs(args)
	out, err := exec.Execute(conn.psql("", "SELECT count(*) FROM pg_database WHERE datname = "+pgString(dbName)))
	if err != nil {
		return fail(fmt.Errorf("failed to check database: %w", err))
	}
	exists := strings.TrimSpace(out) != "0"

	switch state {
	case "present":
		if !exists {
			sql := "CREATE DATABASE " + pgIdent(dbName)
			if owner, _ := args["owner"].(string); owner != "" {
				sql += " OWNER " + pgIdent(owner)
			}
			// template1 may carry another encoding; template0 accepts any
			if encoding, _ := args["encoding"].(string); encoding != "" {
				sql += " ENCODING " + pgString(encoding) + " TEMPLATE template0"
			}
			if _, err := exec.Execute(conn.psql("", sql)); err != nil {
				return fail(fmt.Errorf("failed to create database: %w", err))
			}
			result.Changed = true
			result.Output["action"] = "created"
		}

	case "absent":
		if exists {
			if _, err := exec.Execute(conn.psql("", "DROP DATABASE "+pgIdent(dbName))); err != nil {
				return fail(fmt.Errorf("failed to drop database: %w", err))
			}
			result.Changed = true
			result.Output["action"] = "dropped"
		}

	case "dump":
		cmd := conn.pgEnv() + "pg_dump" + conn.psqlArgs() + " -f " + shellQuote(target) + " " + shellQuote(dbName)
		if _, err := exec.Execute(cmd); err != nil {
			return fail(fmt.Errorf("failed to dump database: %w", err))
		}
		result.Changed = true
		result.Output["action"] = "dumped"

	case "restore":
		cmd := conn.pgEnv() + "psql -X -q -v ON_ERROR_STOP=1" + conn.psqlArgs() + " -d " + shellQuote(dbName) + " -f " + shellQuote(target)
		if _, err := exec.Execute(cmd); err != nil {
			return fail(fmt.Errorf("failed to restore database: %w", err))
		}
		result.Changed = true
		result.Output["action"] = "restored"

	default:
		return fail(fmt.Errorf("unsupported state %q", state))
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// Validate validates postgresql_db module arguments
func (m *PostgreSQLDBModule) Validate(args map[string]interface{}) error {
	return requireStringArg(args, "name")
}
