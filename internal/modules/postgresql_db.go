package modules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

type PostgreSQLDBModule struct {
	*BaseExecutorModule
}

func NewPostgreSQLDBModule() *PostgreSQLDBModule {
	return &PostgreSQLDBModule{
		BaseExecutorModule: NewBaseExecutorModule("postgresql_db"),
	}
}

func (m *PostgreSQLDBModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()
	result := types.TaskResult{
		TaskName: "postgresql_db", Host: host.Name, Module: m.GetName(),
		Success: true, Changed: false, Output: make(map[string]interface{}), Timestamp: startTime,
	}

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
		loginUser = "postgres"
	}

	loginHost, _ := args["login_host"].(string)
	if loginHost == "" {
		loginHost = "localhost"
	}

	loginPort, _ := args["login_port"].(string)
	if loginPort == "" {
		loginPort = "5432"
	}

	// Use WithExecutor to get fresh executor for this host
	err := m.WithExecutor(host, func(exec *executor.CommandExecutor) error {
		exists, err := m.databaseExists(ctx, exec, dbName, loginUser, loginHost, loginPort)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to check database: %v", err)
			return err
		}

		switch state {
		case "present":
			if !exists {
				owner, _ := args["owner"].(string)
				encoding, _ := args["encoding"].(string)
				if encoding == "" {
					encoding = "UTF8"
				}

				if err := m.createDatabase(ctx, exec, dbName, owner, encoding, loginUser, loginHost, loginPort); err != nil {
					result.Success = false
					result.Error = fmt.Sprintf("failed to create database: %v", err)
					return err
				}
				result.Changed = true
				result.Output["action"] = "created"
			}

		case "absent":
			if exists {
				if err := m.dropDatabase(ctx, exec, dbName, loginUser, loginHost, loginPort); err != nil {
					result.Success = false
					result.Error = fmt.Sprintf("failed to drop database: %v", err)
					return err
				}
				result.Changed = true
				result.Output["action"] = "dropped"
			}

		case "dump":
			target, _ := args["target"].(string)
			if target == "" {
				result.Success = false
				result.Error = "target file is required for dump"
				return fmt.Errorf("target file is required for dump")
			}

			if err := m.dumpDatabase(ctx, exec, dbName, target, loginUser, loginHost, loginPort); err != nil {
				result.Success = false
				result.Error = fmt.Sprintf("failed to dump database: %v", err)
				return err
			}
			result.Changed = true
			result.Output["action"] = "dumped"

		case "restore":
			target, _ := args["target"].(string)
			if target == "" {
				result.Success = false
				result.Error = "target file is required for restore"
				return fmt.Errorf("target file is required for restore")
			}

			if err := m.restoreDatabase(ctx, exec, dbName, target, loginUser, loginHost, loginPort); err != nil {
				result.Success = false
				result.Error = fmt.Sprintf("failed to restore database: %v", err)
				return err
			}
			result.Changed = true
			result.Output["action"] = "restored"
		}

		return nil
	})

	if err != nil && result.Success {
		// WithExecutor failed
		result.Success = false
		result.Error = fmt.Sprintf("executor error: %v", err)
		return result, err
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

func (m *PostgreSQLDBModule) buildPsqlCmd(loginUser, loginHost, loginPort string) string {
	cmdParts := []string{"psql"}
	if loginUser != "" {
		cmdParts = append(cmdParts, "-U", loginUser)
	}
	if loginHost != "" {
		cmdParts = append(cmdParts, "-h", loginHost)
	}
	if loginPort != "" {
		cmdParts = append(cmdParts, "-p", loginPort)
	}
	return strings.Join(cmdParts, " ")
}

func (m *PostgreSQLDBModule) databaseExists(ctx context.Context, exec *executor.CommandExecutor, dbName, loginUser, loginHost, loginPort string) (bool, error) {
	baseCmd := m.buildPsqlCmd(loginUser, loginHost, loginPort)
	cmd := fmt.Sprintf("%s -lqt | cut -d \\| -f 1 | grep -qw %s", baseCmd, dbName)

	_, err := exec.Execute(cmd)
	if err != nil {
		return false, nil
	}

	return err == nil, nil
}

func (m *PostgreSQLDBModule) createDatabase(ctx context.Context, exec *executor.CommandExecutor, dbName, owner, encoding, loginUser, loginHost, loginPort string) error {
	baseCmd := m.buildPsqlCmd(loginUser, loginHost, loginPort)
	createCmd := fmt.Sprintf("CREATE DATABASE %s", dbName)

	if encoding != "" {
		createCmd += fmt.Sprintf(" ENCODING '%s'", encoding)
	}
	if owner != "" {
		createCmd += fmt.Sprintf(" OWNER %s", owner)
	}

	cmd := fmt.Sprintf("%s -c \"%s\"", baseCmd, createCmd)
	_, err := exec.Execute(cmd)
	if err != nil {
		return fmt.Errorf("failed to create database: %s", err.Error())
	}

	return nil
}

func (m *PostgreSQLDBModule) dropDatabase(ctx context.Context, exec *executor.CommandExecutor, dbName, loginUser, loginHost, loginPort string) error {
	baseCmd := m.buildPsqlCmd(loginUser, loginHost, loginPort)
	cmd := fmt.Sprintf("%s -c \"DROP DATABASE %s\"", baseCmd, dbName)

	_, err := exec.Execute(cmd)
	if err != nil {
		return fmt.Errorf("failed to drop database: %s", err.Error())
	}

	return nil
}

func (m *PostgreSQLDBModule) dumpDatabase(ctx context.Context, exec *executor.CommandExecutor, dbName, target, loginUser, loginHost, loginPort string) error {
	cmdParts := []string{"pg_dump"}
	if loginUser != "" {
		cmdParts = append(cmdParts, "-U", loginUser)
	}
	if loginHost != "" {
		cmdParts = append(cmdParts, "-h", loginHost)
	}
	if loginPort != "" {
		cmdParts = append(cmdParts, "-p", loginPort)
	}
	cmdParts = append(cmdParts, "-f", target, dbName)

	cmd := strings.Join(cmdParts, " ")
	_, err := exec.Execute(cmd)
	if err != nil {
		return fmt.Errorf("failed to dump database: %s", err.Error())
	}

	return nil
}

func (m *PostgreSQLDBModule) restoreDatabase(ctx context.Context, exec *executor.CommandExecutor, dbName, target, loginUser, loginHost, loginPort string) error {
	baseCmd := m.buildPsqlCmd(loginUser, loginHost, loginPort)
	cmd := fmt.Sprintf("%s -d %s -f %s", baseCmd, dbName, target)

	_, err := exec.Execute(cmd)
	if err != nil {
		return fmt.Errorf("failed to restore database: %s", err.Error())
	}

	return nil
}
