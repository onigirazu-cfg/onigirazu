package modules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

type MySQLUserModule struct {
	*BaseExecutorModule
}

func NewMySQLUserModule() *MySQLUserModule {
	m := &MySQLUserModule{
		BaseExecutorModule: NewBaseExecutorModule("mysql_user"),
	}
	m.description = "Manage MySQL users and permissions"
	return m
}

func (m *MySQLUserModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()
	result := types.TaskResult{
		TaskName:  "mysql_user",
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

	userHost, _ := args["host"].(string)
	if userHost == "" {
		userHost = "localhost"
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

	exists, err := m.userExists(ctx, exec, userName, userHost, loginUser, loginPassword, loginHost, loginPort)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to check user: %v", err)
		return result, err
	}

	switch state {
	case "present":
		if !exists {
			password, _ := args["password"].(string)
			if err := m.createUser(ctx, exec, userName, userHost, password, loginUser, loginPassword, loginHost, loginPort); err != nil {
				result.Success = false
				result.Error = fmt.Sprintf("failed to create user: %v", err)
				return result, err
			}
			result.Changed = true
			result.Output["action"] = "created"
		}

		if priv, ok := args["priv"].(string); ok && priv != "" {
			if err := m.grantPrivileges(ctx, exec, userName, userHost, priv, loginUser, loginPassword, loginHost, loginPort); err != nil {
				result.Success = false
				result.Error = fmt.Sprintf("failed to grant privileges: %v", err)
				return result, err
			}
			result.Changed = true
			result.Output["privileges"] = "granted"
		}

	case "absent":
		if exists {
			if err := m.dropUser(ctx, exec, userName, userHost, loginUser, loginPassword, loginHost, loginPort); err != nil {
				result.Success = false
				result.Error = fmt.Sprintf("failed to drop user: %v", err)
				return result, err
			}
			result.Changed = true
			result.Output["action"] = "dropped"
		}
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

func (m *MySQLUserModule) buildMySQLCmd(loginUser, loginPassword, loginHost, loginPort string) string {
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

func (m *MySQLUserModule) userExists(ctx context.Context, exec *executor.CommandExecutor, userName, userHost, loginUser, loginPassword, loginHost, loginPort string) (bool, error) {
	baseCmd := m.buildMySQLCmd(loginUser, loginPassword, loginHost, loginPort)
	cmd := fmt.Sprintf("%s -e \"SELECT User FROM mysql.user WHERE User='%s' AND Host='%s'\" 2>/dev/null", baseCmd, userName, userHost)

	stdout, err := exec.Execute(cmd)
	if err != nil {
		return false, nil
	}

	return strings.Contains(stdout, userName), nil
}

func (m *MySQLUserModule) createUser(ctx context.Context, exec *executor.CommandExecutor, userName, userHost, password, loginUser, loginPassword, loginHost, loginPort string) error {
	baseCmd := m.buildMySQLCmd(loginUser, loginPassword, loginHost, loginPort)
	cmd := fmt.Sprintf("%s -e \"CREATE USER '%s'@'%s' IDENTIFIED BY '%s'\"", baseCmd, userName, userHost, password)

	_, err := exec.Execute(cmd)
	if err != nil {
		return fmt.Errorf("failed to create user: %s", err.Error())
	}

	return nil
}

func (m *MySQLUserModule) dropUser(ctx context.Context, exec *executor.CommandExecutor, userName, userHost, loginUser, loginPassword, loginHost, loginPort string) error {
	baseCmd := m.buildMySQLCmd(loginUser, loginPassword, loginHost, loginPort)
	cmd := fmt.Sprintf("%s -e \"DROP USER '%s'@'%s'\"", baseCmd, userName, userHost)

	_, err := exec.Execute(cmd)
	if err != nil {
		return fmt.Errorf("failed to drop user: %s", err.Error())
	}

	return nil
}

func (m *MySQLUserModule) grantPrivileges(ctx context.Context, exec *executor.CommandExecutor, userName, userHost, priv, loginUser, loginPassword, loginHost, loginPort string) error {
	baseCmd := m.buildMySQLCmd(loginUser, loginPassword, loginHost, loginPort)
	cmd := fmt.Sprintf("%s -e \"GRANT %s TO '%s'@'%s'; FLUSH PRIVILEGES\"", baseCmd, priv, userName, userHost)

	_, err := exec.Execute(cmd)
	if err != nil {
		return fmt.Errorf("failed to grant privileges: %s", err.Error())
	}

	return nil
}
