package modules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

type PostgreSQLUserModule struct {
	*BaseExecutorModule
}

func NewPostgreSQLUserModule() *PostgreSQLUserModule {
	m := &PostgreSQLUserModule{
		BaseExecutorModule: NewBaseExecutorModule("postgresql_user"),
	}
	m.description = "Manage PostgreSQL users and roles"
	return m
}

func (m *PostgreSQLUserModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()
	result := types.TaskResult{
		TaskName: "postgresql_user", Host: host.Name, Module: m.GetName(),
		Success: true, Changed: false, Output: make(map[string]interface{}), Timestamp: startTime,
	}

	userName, ok := args["name"].(string)
	if !ok || userName == "" {
		result.Success = false
		result.Error = "user name is required"
		return result, fmt.Errorf("user name is required")
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
		exists, err := m.userExists(ctx, exec, userName, loginUser, loginHost, loginPort)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to check user: %v", err)
			return err
		}

		switch state {
		case "present":
			if !exists {
				password, _ := args["password"].(string)
				if err := m.createUser(ctx, exec, userName, password, args, loginUser, loginHost, loginPort); err != nil {
					result.Success = false
					result.Error = fmt.Sprintf("failed to create user: %v", err)
					return err
				}
				result.Changed = true
				result.Output["action"] = "created"
			}

			if priv, ok := args["priv"].(string); ok && priv != "" {
				db, _ := args["db"].(string)
				if err := m.grantPrivileges(ctx, exec, userName, db, priv, loginUser, loginHost, loginPort); err != nil {
					result.Success = false
					result.Error = fmt.Sprintf("failed to grant privileges: %v", err)
					return err
				}
				result.Changed = true
				result.Output["privileges"] = "granted"
			}

		case "absent":
			if exists {
				if err := m.dropUser(ctx, exec, userName, loginUser, loginHost, loginPort); err != nil {
					result.Success = false
					result.Error = fmt.Sprintf("failed to drop user: %v", err)
					return err
				}
				result.Changed = true
				result.Output["action"] = "dropped"
			}
		}

		return nil
	})

	if err != nil && result.Success {
		result.Success = false
		result.Error = fmt.Sprintf("executor error: %v", err)
		return result, err
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

func (m *PostgreSQLUserModule) buildPsqlCmd(loginUser, loginHost, loginPort string) string {
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

func (m *PostgreSQLUserModule) userExists(ctx context.Context, exec *executor.CommandExecutor, userName, loginUser, loginHost, loginPort string) (bool, error) {
	baseCmd := m.buildPsqlCmd(loginUser, loginHost, loginPort)
	cmd := fmt.Sprintf("%s -tAc \"SELECT 1 FROM pg_roles WHERE rolname='%s'\"", baseCmd, userName)

	stdout, err := exec.Execute(cmd)
	if err != nil {
		return false, nil
	}

	return strings.TrimSpace(stdout) == "1", nil
}

func (m *PostgreSQLUserModule) createUser(ctx context.Context, exec *executor.CommandExecutor, userName, password string, args map[string]interface{}, loginUser, loginHost, loginPort string) error {
	baseCmd := m.buildPsqlCmd(loginUser, loginHost, loginPort)
	createCmd := fmt.Sprintf("CREATE USER %s", userName)

	if password != "" {
		createCmd += fmt.Sprintf(" WITH PASSWORD '%s'", password)
	}

	if superuser, ok := args["superuser"].(bool); ok && superuser {
		createCmd += " SUPERUSER"
	}

	if createdb, ok := args["createdb"].(bool); ok && createdb {
		createCmd += " CREATEDB"
	}

	cmd := fmt.Sprintf("%s -c \"%s\"", baseCmd, createCmd)
	_, err := exec.Execute(cmd)
	if err != nil {
		return fmt.Errorf("failed to create user: %s", err.Error())
	}

	return nil
}

func (m *PostgreSQLUserModule) dropUser(ctx context.Context, exec *executor.CommandExecutor, userName, loginUser, loginHost, loginPort string) error {
	baseCmd := m.buildPsqlCmd(loginUser, loginHost, loginPort)
	cmd := fmt.Sprintf("%s -c \"DROP USER %s\"", baseCmd, userName)

	_, err := exec.Execute(cmd)
	if err != nil {
		return fmt.Errorf("failed to drop user: %s", err.Error())
	}

	return nil
}

func (m *PostgreSQLUserModule) grantPrivileges(ctx context.Context, exec *executor.CommandExecutor, userName, db, priv, loginUser, loginHost, loginPort string) error {
	baseCmd := m.buildPsqlCmd(loginUser, loginHost, loginPort)
	cmd := fmt.Sprintf("%s -c \"GRANT %s ON DATABASE %s TO %s\"", baseCmd, priv, db, userName)

	_, err := exec.Execute(cmd)
	if err != nil {
		return fmt.Errorf("failed to grant privileges: %s", err.Error())
	}

	return nil
}
