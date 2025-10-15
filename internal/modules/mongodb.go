package modules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

type MongoDBModule struct {
	BaseModule
	executor *executor.CommandExecutor
}

func NewMongoDBModule() *MongoDBModule {
	return &MongoDBModule{
		BaseModule: BaseModule{name: "mongodb", description: "Manage MongoDB databases and users"},
	}
}

func (m *MongoDBModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()
	result := types.TaskResult{
		TaskName: "mongodb", Host: host.Name, Module: m.GetName(),
		Success: true, Changed: false, Output: make(map[string]interface{}), Timestamp: startTime,
	}

	if m.executor == nil {
		var err error
		m.executor, err = executor.NewCommandExecutor(host)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to create executor: %v", err)
			return result, err
		}
	}

	operation, _ := args["operation"].(string)
	if operation == "" {
		operation = "database"
	}

	loginHost, _ := args["login_host"].(string)
	if loginHost == "" {
		loginHost = "localhost"
	}

	loginPort, _ := args["login_port"].(string)
	if loginPort == "" {
		loginPort = "27017"
	}

	loginUser, _ := args["login_user"].(string)
	loginPassword, _ := args["login_password"].(string)
	loginDatabase, _ := args["login_database"].(string)
	if loginDatabase == "" {
		loginDatabase = "admin"
	}

	switch operation {
	case "database":
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

		if err := m.handleDatabase(ctx, dbName, state, loginUser, loginPassword, loginHost, loginPort, loginDatabase, &result); err != nil {
			result.Success = false
			result.Error = err.Error()
			return result, err
		}

	case "user":
		userName, ok := args["name"].(string)
		if !ok || userName == "" {
			result.Success = false
			result.Error = "user name is required"
			return result, fmt.Errorf("user name is required")
		}

		database, ok := args["database"].(string)
		if !ok || database == "" {
			result.Success = false
			result.Error = "database is required for user operation"
			return result, fmt.Errorf("database is required for user operation")
		}

		state, _ := args["state"].(string)
		if state == "" {
			state = "present"
		}

		if err := m.handleUser(ctx, userName, database, state, args, loginUser, loginPassword, loginHost, loginPort, loginDatabase, &result); err != nil {
			result.Success = false
			result.Error = err.Error()
			return result, err
		}
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

func (m *MongoDBModule) buildMongoCmd(loginUser, loginPassword, loginHost, loginPort, loginDatabase string) string {
	cmdParts := []string{"mongo"}

	if loginHost != "" && loginPort != "" {
		cmdParts = append(cmdParts, fmt.Sprintf("--host %s --port %s", loginHost, loginPort))
	}

	if loginUser != "" && loginPassword != "" {
		cmdParts = append(cmdParts, fmt.Sprintf("--username %s --password '%s' --authenticationDatabase %s", loginUser, loginPassword, loginDatabase))
	}

	return strings.Join(cmdParts, " ")
}

func (m *MongoDBModule) handleDatabase(ctx context.Context, dbName, state, loginUser, loginPassword, loginHost, loginPort, loginDatabase string, result *types.TaskResult) error {
	exists, err := m.databaseExists(ctx, dbName, loginUser, loginPassword, loginHost, loginPort, loginDatabase)
	if err != nil {
		return fmt.Errorf("failed to check database: %v", err)
	}

	switch state {
	case "present":
		if !exists {
			if err := m.createDatabase(ctx, dbName, loginUser, loginPassword, loginHost, loginPort, loginDatabase); err != nil {
				return fmt.Errorf("failed to create database: %v", err)
			}
			result.Changed = true
			result.Output["action"] = "created"
		}

	case "absent":
		if exists {
			if err := m.dropDatabase(ctx, dbName, loginUser, loginPassword, loginHost, loginPort, loginDatabase); err != nil {
				return fmt.Errorf("failed to drop database: %v", err)
			}
			result.Changed = true
			result.Output["action"] = "dropped"
		}
	}

	return nil
}

func (m *MongoDBModule) handleUser(ctx context.Context, userName, database, state string, args map[string]interface{}, loginUser, loginPassword, loginHost, loginPort, loginDatabase string, result *types.TaskResult) error {
	exists, err := m.userExists(ctx, userName, database, loginUser, loginPassword, loginHost, loginPort, loginDatabase)
	if err != nil {
		return fmt.Errorf("failed to check user: %v", err)
	}

	switch state {
	case "present":
		if !exists {
			password, _ := args["password"].(string)
			roles, _ := args["roles"].([]interface{})
			if err := m.createUser(ctx, userName, password, database, roles, loginUser, loginPassword, loginHost, loginPort, loginDatabase); err != nil {
				return fmt.Errorf("failed to create user: %v", err)
			}
			result.Changed = true
			result.Output["action"] = "created"
		}

	case "absent":
		if exists {
			if err := m.dropUser(ctx, userName, database, loginUser, loginPassword, loginHost, loginPort, loginDatabase); err != nil {
				return fmt.Errorf("failed to drop user: %v", err)
			}
			result.Changed = true
			result.Output["action"] = "dropped"
		}
	}

	return nil
}

func (m *MongoDBModule) databaseExists(ctx context.Context, dbName, loginUser, loginPassword, loginHost, loginPort, loginDatabase string) (bool, error) {
	baseCmd := m.buildMongoCmd(loginUser, loginPassword, loginHost, loginPort, loginDatabase)
	cmd := fmt.Sprintf("%s --quiet --eval \"db.adminCommand('listDatabases').databases.map(d => d.name).includes('%s')\"", baseCmd, dbName)

	stdout, err := m.executor.Execute(cmd)
	if err != nil {
		return false, nil
	}

	return strings.TrimSpace(stdout) == "true", nil
}

func (m *MongoDBModule) createDatabase(ctx context.Context, dbName, loginUser, loginPassword, loginHost, loginPort, loginDatabase string) error {
	baseCmd := m.buildMongoCmd(loginUser, loginPassword, loginHost, loginPort, loginDatabase)
	cmd := fmt.Sprintf("%s %s --eval \"db.createCollection('_init')\"", baseCmd, dbName)

	_, err := m.executor.Execute(cmd)
	if err != nil {
		return fmt.Errorf("failed to create database: %s", err.Error())
	}

	return nil
}

func (m *MongoDBModule) dropDatabase(ctx context.Context, dbName, loginUser, loginPassword, loginHost, loginPort, loginDatabase string) error {
	baseCmd := m.buildMongoCmd(loginUser, loginPassword, loginHost, loginPort, loginDatabase)
	cmd := fmt.Sprintf("%s %s --eval \"db.dropDatabase()\"", baseCmd, dbName)

	_, err := m.executor.Execute(cmd)
	if err != nil {
		return fmt.Errorf("failed to drop database: %s", err.Error())
	}

	return nil
}

func (m *MongoDBModule) userExists(ctx context.Context, userName, database, loginUser, loginPassword, loginHost, loginPort, loginDatabase string) (bool, error) {
	baseCmd := m.buildMongoCmd(loginUser, loginPassword, loginHost, loginPort, loginDatabase)
	cmd := fmt.Sprintf("%s %s --quiet --eval \"db.getUser('%s') != null\"", baseCmd, database, userName)

	stdout, err := m.executor.Execute(cmd)
	if err != nil {
		return false, nil
	}

	return strings.TrimSpace(stdout) == "true", nil
}

func (m *MongoDBModule) createUser(ctx context.Context, userName, password, database string, roles []interface{}, loginUser, loginPassword, loginHost, loginPort, loginDatabase string) error {
	baseCmd := m.buildMongoCmd(loginUser, loginPassword, loginHost, loginPort, loginDatabase)

	rolesStr := "[]"
	if len(roles) > 0 {
		rolesList := make([]string, len(roles))
		for i, role := range roles {
			rolesList[i] = fmt.Sprintf("'%v'", role)
		}
		rolesStr = fmt.Sprintf("[%s]", strings.Join(rolesList, ","))
	}

	cmd := fmt.Sprintf("%s %s --eval \"db.createUser({user: '%s', pwd: '%s', roles: %s})\"", baseCmd, database, userName, password, rolesStr)

	_, err := m.executor.Execute(cmd)
	if err != nil {
		return fmt.Errorf("failed to create user: %s", err.Error())
	}

	return nil
}

func (m *MongoDBModule) dropUser(ctx context.Context, userName, database, loginUser, loginPassword, loginHost, loginPort, loginDatabase string) error {
	baseCmd := m.buildMongoCmd(loginUser, loginPassword, loginHost, loginPort, loginDatabase)
	cmd := fmt.Sprintf("%s %s --eval \"db.dropUser('%s')\"", baseCmd, database, userName)

	_, err := m.executor.Execute(cmd)
	if err != nil {
		return fmt.Errorf("failed to drop user: %s", err.Error())
	}

	return nil
}
