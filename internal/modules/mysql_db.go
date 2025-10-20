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
		Changed:   false,
		Output:    make(map[string]interface{}),
		Timestamp: startTime,
	}

	// Use CreateExecutor to get fresh executor for this host
	exec, err := m.CreateExecutor(host)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to create executor: %v", err)
		return result, err
	}
	defer exec.Close()

	dbName, ok := args["name"].(string)
	if !ok || dbName == "" {
		result.Success = false
		result.Error = "database name is required"
		return result, fmt.Errorf("database name is required")
	}

	state, _ := args["state"].(string)
	if state == "" {
		state = "present"
	}

	loginUser, _ := args["login_user"].(string)
	if loginUser == "" {
		loginUser = "root"
	}

	loginPassword, _ := args["login_password"].(string)
	loginHost, _ := args["login_host"].(string)
	if loginHost == "" {
		loginHost = "localhost"
	}

	loginPort, _ := args["login_port"].(string)
	if loginPort == "" {
		loginPort = "3306"
	}

	exists, _, err := m.databaseExists(ctx, exec, dbName, loginUser, loginPassword, loginHost, loginPort)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to check database: %v", err)
		return result, err
	}

	switch state {
	case "present":
		if !exists {
			charset, _ := args["charset"].(string)
			if charset == "" {
				charset = "utf8mb4"
			}

			collation, _ := args["collation"].(string)
			if collation == "" {
				collation = "utf8mb4_unicode_ci"
			}

			if err := m.createDatabase(ctx, exec, dbName, charset, collation, loginUser, loginPassword, loginHost, loginPort); err != nil {
				result.Success = false
				result.Error = fmt.Sprintf("failed to create database: %v", err)
				return result, err
			}
			result.Changed = true
			result.Output["action"] = "created"
		}

	case "absent":
		if exists {
			if err := m.dropDatabase(ctx, exec, dbName, loginUser, loginPassword, loginHost, loginPort); err != nil {
				result.Success = false
				result.Error = fmt.Sprintf("failed to drop database: %v", err)
				return result, err
			}
			result.Changed = true
			result.Output["action"] = "dropped"
		}

	case "dump":
		target, _ := args["target"].(string)
		if target == "" {
			result.Success = false
			result.Error = "target file is required for dump"
			return result, fmt.Errorf("target file is required for dump")
		}

		if err := m.dumpDatabase(ctx, exec, dbName, target, loginUser, loginPassword, loginHost, loginPort); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to dump database: %v", err)
			return result, err
		}
		result.Changed = true
		result.Output["action"] = "dumped"

	case "import":
		target, _ := args["target"].(string)
		if target == "" {
			result.Success = false
			result.Error = "target file is required for import"
			return result, fmt.Errorf("target file is required for import")
		}

		if err := m.importDatabase(ctx, exec, dbName, target, loginUser, loginPassword, loginHost, loginPort); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to import database: %v", err)
			return result, err
		}
		result.Changed = true
		result.Output["action"] = "imported"
	}

	if state != "absent" {
		_, dbInf, _ := m.databaseExists(ctx, exec, dbName, loginUser, loginPassword, loginHost, loginPort)
		result.Output["database"] = dbInf
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

func (m *MySQLDBModule) buildMySQLCmd(loginUser, loginPassword, loginHost, loginPort string) string {
	cmdParts := []string{"mysql"}

	if loginUser != "" {
		cmdParts = append(cmdParts, fmt.Sprintf("-u%s", loginUser))
	}

	if loginPassword != "" {
		cmdParts = append(cmdParts, fmt.Sprintf("-p'%s'", loginPassword))
	}

	if loginHost != "" {
		cmdParts = append(cmdParts, fmt.Sprintf("-h%s", loginHost))
	}

	if loginPort != "" {
		cmdParts = append(cmdParts, fmt.Sprintf("-P%s", loginPort))
	}

	return strings.Join(cmdParts, " ")
}

func (m *MySQLDBModule) databaseExists(ctx context.Context, exec *executor.CommandExecutor, dbName, loginUser, loginPassword, loginHost, loginPort string) (bool, *MySQLDBInfo, error) {
	baseCmd := m.buildMySQLCmd(loginUser, loginPassword, loginHost, loginPort)
	cmd := fmt.Sprintf("%s -e \"SELECT SCHEMA_NAME, DEFAULT_CHARACTER_SET_NAME, DEFAULT_COLLATION_NAME FROM information_schema.SCHEMATA WHERE SCHEMA_NAME='%s'\" 2>/dev/null", baseCmd, dbName)

	stdout, err := exec.Execute(cmd)
	if err != nil {
		return false, nil, nil
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) < 2 {
		return false, nil, nil
	}

	fields := strings.Fields(lines[1])
	if len(fields) < 3 {
		return false, nil, nil
	}

	info := &MySQLDBInfo{
		Name:      fields[0],
		Charset:   fields[1],
		Collation: fields[2],
		Exists:    true,
	}

	return true, info, nil
}

func (m *MySQLDBModule) createDatabase(ctx context.Context, exec *executor.CommandExecutor, dbName, charset, collation, loginUser, loginPassword, loginHost, loginPort string) error {
	baseCmd := m.buildMySQLCmd(loginUser, loginPassword, loginHost, loginPort)
	cmd := fmt.Sprintf("%s -e \"CREATE DATABASE %s CHARACTER SET %s COLLATE %s\"", baseCmd, dbName, charset, collation)

	_, err := exec.Execute(cmd)
	if err != nil {
		return fmt.Errorf("failed to create database: %s", err.Error())
	}

	return nil
}

func (m *MySQLDBModule) dropDatabase(ctx context.Context, exec *executor.CommandExecutor, dbName, loginUser, loginPassword, loginHost, loginPort string) error {
	baseCmd := m.buildMySQLCmd(loginUser, loginPassword, loginHost, loginPort)
	cmd := fmt.Sprintf("%s -e \"DROP DATABASE %s\"", baseCmd, dbName)

	_, err := exec.Execute(cmd)
	if err != nil {
		return fmt.Errorf("failed to drop database: %s", err.Error())
	}

	return nil
}

func (m *MySQLDBModule) dumpDatabase(ctx context.Context, exec *executor.CommandExecutor, dbName, target, loginUser, loginPassword, loginHost, loginPort string) error {
	cmdParts := []string{"mysqldump"}

	if loginUser != "" {
		cmdParts = append(cmdParts, fmt.Sprintf("-u%s", loginUser))
	}

	if loginPassword != "" {
		cmdParts = append(cmdParts, fmt.Sprintf("-p'%s'", loginPassword))
	}

	if loginHost != "" {
		cmdParts = append(cmdParts, fmt.Sprintf("-h%s", loginHost))
	}

	if loginPort != "" {
		cmdParts = append(cmdParts, fmt.Sprintf("-P%s", loginPort))
	}

	cmdParts = append(cmdParts, dbName, ">", target)
	cmd := strings.Join(cmdParts, " ")

	_, err := exec.Execute(cmd)
	if err != nil {
		return fmt.Errorf("failed to dump database: %s", err.Error())
	}

	return nil
}

func (m *MySQLDBModule) importDatabase(ctx context.Context, exec *executor.CommandExecutor, dbName, target, loginUser, loginPassword, loginHost, loginPort string) error {
	baseCmd := m.buildMySQLCmd(loginUser, loginPassword, loginHost, loginPort)
	cmd := fmt.Sprintf("%s %s < %s", baseCmd, dbName, target)

	_, err := exec.Execute(cmd)
	if err != nil {
		return fmt.Errorf("failed to import database: %s", err.Error())
	}

	return nil
}
