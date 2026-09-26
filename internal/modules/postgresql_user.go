package modules

import (
	"context"
	"fmt"
	"strings"
	"time"

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
		Success: true, Output: make(map[string]interface{}), Timestamp: startTime,
	}
	fail := func(err error) (types.TaskResult, error) {
		result.Success = false
		result.Error = err.Error()
		result.Duration = time.Since(startTime)
		return result, err
	}

	userName, _ := args["name"].(string)
	if userName == "" {
		return fail(fmt.Errorf("user name is required"))
	}
	state, _ := args["state"].(string)
	if state == "" {
		state = "present"
	}
	priv, _ := args["priv"].(string)
	db, _ := args["db"].(string)
	if priv != "" {
		if db == "" {
			return fail(fmt.Errorf("priv requires db"))
		}
		for _, p := range strings.Split(priv, ",") {
			for _, r := range strings.TrimSpace(p) {
				if !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r == ' ') {
					return fail(fmt.Errorf("invalid privilege %q", p))
				}
			}
		}
	}

	exec, err := m.CreateExecutor(host)
	if err != nil {
		return fail(fmt.Errorf("failed to create executor: %w", err))
	}
	defer exec.Close()

	conn := dbConnFromArgs(args)
	out, err := exec.Execute(conn.psql("", "SELECT count(*) FROM pg_roles WHERE rolname = "+pgString(userName)))
	if err != nil {
		return fail(fmt.Errorf("failed to check user: %w", err))
	}
	exists := strings.TrimSpace(out) != "0"

	switch state {
	case "present":
		if !exists {
			sql := "CREATE ROLE " + pgIdent(userName) + " LOGIN"
			if password, _ := args["password"].(string); password != "" {
				sql += " PASSWORD " + pgString(password)
			}
			if superuser := getBoolArg(args, "superuser", false); superuser {
				sql += " SUPERUSER"
			}
			if createdb := getBoolArg(args, "createdb", false); createdb {
				sql += " CREATEDB"
			}
			if _, err := exec.Execute(conn.psql("", sql)); err != nil {
				return fail(fmt.Errorf("failed to create user: %w", err))
			}
			result.Changed = true
			result.Output["action"] = "created"
		}
		if priv != "" {
			acl := "SELECT datacl FROM pg_database WHERE datname = " + pgString(db)
			before, _ := exec.Execute(conn.psql("", acl))
			sql := fmt.Sprintf("GRANT %s ON DATABASE %s TO %s", priv, pgIdent(db), pgIdent(userName))
			if _, err := exec.Execute(conn.psql("", sql)); err != nil {
				return fail(fmt.Errorf("failed to grant privileges: %w", err))
			}
			if after, _ := exec.Execute(conn.psql("", acl)); after != before {
				result.Changed = true
				result.Output["privileges"] = "granted"
			}
		}

	case "absent":
		if exists {
			if _, err := exec.Execute(conn.psql("", "DROP ROLE "+pgIdent(userName))); err != nil {
				return fail(fmt.Errorf("failed to drop user: %w", err))
			}
			result.Changed = true
			result.Output["action"] = "dropped"
		}

	default:
		return fail(fmt.Errorf("unsupported state %q", state))
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// Validate validates postgresql_user module arguments
func (m *PostgreSQLUserModule) Validate(args map[string]interface{}) error {
	return requireStringArg(args, "name")
}
