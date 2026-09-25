package modules

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

type MongoDBModule struct {
	*BaseExecutorModule
}

func NewMongoDBModule() *MongoDBModule {
	return &MongoDBModule{
		BaseExecutorModule: NewBaseExecutorModule("mongodb"),
	}
}

// GetDescription returns the module description
func (m *MongoDBModule) GetDescription() string {
	return "Manage MongoDB databases and users"
}

func (m *MongoDBModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()
	result := types.TaskResult{
		TaskName: "mongodb", Host: host.Name, Module: m.GetName(),
		Success: true, Output: make(map[string]interface{}), Timestamp: startTime,
	}
	fail := func(err error) (types.TaskResult, error) {
		result.Success = false
		result.Error = err.Error()
		result.Duration = time.Since(startTime)
		return result, err
	}

	operation, _ := args["operation"].(string)
	if operation == "" {
		operation = "database"
	}
	state, _ := args["state"].(string)
	if state == "" {
		state = "present"
	}
	if state != "present" && state != "absent" {
		return fail(fmt.Errorf("unsupported state %q", state))
	}
	name, _ := args["name"].(string)
	database, _ := args["database"].(string)
	switch {
	case operation != "database" && operation != "user":
		return fail(fmt.Errorf("unsupported operation %q", operation))
	case name == "" && operation == "database":
		return fail(fmt.Errorf("database name is required"))
	case name == "":
		return fail(fmt.Errorf("user name is required"))
	case operation == "user" && database == "":
		return fail(fmt.Errorf("database is required for user operation"))
	}

	exec, err := m.CreateExecutor(host)
	if err != nil {
		return fail(fmt.Errorf("failed to create executor: %w", err))
	}
	defer exec.Close()

	shell := newMongoShell(args)
	if operation == "database" {
		err = m.handleDatabase(exec, shell, name, state, &result)
	} else {
		err = m.handleUser(exec, shell, name, database, state, args, &result)
	}
	if err != nil {
		return fail(err)
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// mongoShell builds mongosh command lines (the legacy mongo shell when
// mongosh is missing). Values reach the scripts as JSON literals.
type mongoShell struct {
	host, port, user, password, authDB string
}

func newMongoShell(args map[string]interface{}) mongoShell {
	s := mongoShell{}
	s.host, _ = args["login_host"].(string)
	if p, ok := toInt(args["login_port"]); ok && p > 0 {
		s.port = fmt.Sprint(p)
	}
	s.user, _ = args["login_user"].(string)
	s.password, _ = args["login_password"].(string)
	s.authDB, _ = args["login_database"].(string)
	if s.authDB == "" {
		s.authDB = "admin"
	}
	return s
}

func (s mongoShell) command(db, script string) string {
	opts := " --quiet"
	if s.host != "" {
		opts += " --host " + shellQuote(s.host)
	}
	if s.port != "" {
		opts += " --port " + s.port
	}
	if s.user != "" {
		opts += " --username " + shellQuote(s.user) + " --password " + shellQuote(s.password) +
			" --authenticationDatabase " + shellQuote(s.authDB)
	}
	if db == "" {
		db = "admin"
	}
	opts += " " + shellQuote(db) + " --eval " + shellQuote(script)
	return "if command -v mongosh >/dev/null 2>&1; then mongosh" + opts + "; else mongo" + opts + "; fi"
}

// eval runs script and returns its last output line
func (s mongoShell) eval(exec *executor.CommandExecutor, db, script string) (string, error) {
	out, err := exec.Execute(s.command(db, script))
	if err != nil {
		return "", err
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	return strings.TrimSpace(lines[len(lines)-1]), nil
}

func jsLiteral(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "null"
	}
	return string(b)
}

func (m *MongoDBModule) handleDatabase(exec *executor.CommandExecutor, shell mongoShell, dbName, state string, result *types.TaskResult) error {
	out, err := shell.eval(exec, "admin",
		"db.adminCommand({listDatabases: 1, nameOnly: true}).databases.some(d => d.name === "+jsLiteral(dbName)+")")
	if err != nil {
		return fmt.Errorf("failed to check database: %w", err)
	}
	exists := out == "true"

	switch {
	case state == "present" && !exists:
		// a database exists once it holds a collection
		if _, err := shell.eval(exec, dbName, "db.createCollection('_init')"); err != nil {
			return fmt.Errorf("failed to create database: %w", err)
		}
		result.Changed = true
		result.Output["action"] = "created"
	case state == "absent" && exists:
		if _, err := shell.eval(exec, dbName, "db.dropDatabase()"); err != nil {
			return fmt.Errorf("failed to drop database: %w", err)
		}
		result.Changed = true
		result.Output["action"] = "dropped"
	}
	return nil
}

func (m *MongoDBModule) handleUser(exec *executor.CommandExecutor, shell mongoShell, userName, database, state string, args map[string]interface{}, result *types.TaskResult) error {
	out, err := shell.eval(exec, database, "db.getUser("+jsLiteral(userName)+") !== null")
	if err != nil {
		return fmt.Errorf("failed to check user: %w", err)
	}
	exists := out == "true"

	switch {
	case state == "present" && !exists:
		password, _ := args["password"].(string)
		roles, _ := args["roles"].([]interface{})
		if roles == nil {
			roles = []interface{}{}
		}
		user := map[string]interface{}{"user": userName, "pwd": password, "roles": roles}
		if _, err := shell.eval(exec, database, "db.createUser("+jsLiteral(user)+")"); err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}
		result.Changed = true
		result.Output["action"] = "created"
	case state == "absent" && exists:
		if _, err := shell.eval(exec, database, "db.dropUser("+jsLiteral(userName)+")"); err != nil {
			return fmt.Errorf("failed to drop user: %w", err)
		}
		result.Changed = true
		result.Output["action"] = "dropped"
	}
	return nil
}
