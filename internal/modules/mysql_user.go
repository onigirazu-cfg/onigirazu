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
		Output:    make(map[string]interface{}),
		Timestamp: startTime,
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
	userHost, _ := args["host"].(string)
	if userHost == "" {
		userHost = "localhost"
	}
	priv, _ := args["priv"].(string)
	grants, err := parseMySQLPriv(priv)
	if err != nil {
		return fail(err)
	}

	exec, err := m.CreateExecutor(host)
	if err != nil {
		return fail(fmt.Errorf("failed to create executor: %w", err))
	}
	defer exec.Close()

	conn := dbConnFromArgs(args)
	account := mysqlString(userName) + "@" + mysqlString(userHost)
	out, err := exec.Execute(conn.mysql(fmt.Sprintf("SELECT COUNT(*) FROM mysql.user WHERE User = %s AND Host = %s",
		mysqlString(userName), mysqlString(userHost))))
	if err != nil {
		return fail(fmt.Errorf("failed to check user: %w", err))
	}
	exists := strings.TrimSpace(out) != "0"

	switch state {
	case "present":
		if !exists {
			sql := "CREATE USER " + account
			if password, _ := args["password"].(string); password != "" {
				sql += " IDENTIFIED BY " + mysqlString(password)
			}
			if _, err := exec.Execute(conn.mysql(sql)); err != nil {
				return fail(fmt.Errorf("failed to create user: %w", err))
			}
			result.Changed = true
			result.Output["action"] = "created"
		}
		if len(grants) > 0 {
			before := m.showGrants(exec, conn, account)
			for _, g := range grants {
				if _, err := exec.Execute(conn.mysql(fmt.Sprintf("GRANT %s ON %s TO %s", g.privs, g.object, account))); err != nil {
					return fail(fmt.Errorf("failed to grant privileges: %w", err))
				}
			}
			if m.showGrants(exec, conn, account) != before {
				result.Changed = true
				result.Output["privileges"] = "granted"
			}
		}

	case "absent":
		if exists {
			if _, err := exec.Execute(conn.mysql("DROP USER " + account)); err != nil {
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

func (m *MySQLUserModule) showGrants(exec *executor.CommandExecutor, conn dbConn, account string) string {
	out, err := exec.Execute(conn.mysql("SHOW GRANTS FOR " + account))
	if err != nil {
		return "error: " + err.Error()
	}
	return out
}

type mysqlGrant struct{ object, privs string }

// parseMySQLPriv parses "db.*:ALL/db2.table:SELECT,INSERT" (the Ansible
// format) into GRANT objects with quoted identifiers
func parseMySQLPriv(priv string) ([]mysqlGrant, error) {
	var grants []mysqlGrant
	for _, part := range strings.Split(priv, "/") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		obj, privs, ok := strings.Cut(part, ":")
		db, table, dot := strings.Cut(obj, ".")
		if !ok || !dot || db == "" || table == "" || strings.TrimSpace(privs) == "" {
			return nil, fmt.Errorf("invalid priv %q, expected db.table:PRIV[,PRIV]", part)
		}
		for _, p := range strings.Split(privs, ",") {
			for _, r := range strings.TrimSpace(p) {
				if !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r == ' ' || r == '_') {
					return nil, fmt.Errorf("invalid privilege %q", p)
				}
			}
		}
		quote := func(s string) string {
			if s == "*" {
				return s
			}
			return mysqlIdent(s)
		}
		grants = append(grants, mysqlGrant{object: quote(db) + "." + quote(table), privs: privs})
	}
	return grants, nil
}

// Validate validates mysql_user module arguments
func (m *MySQLUserModule) Validate(args map[string]interface{}) error {
	return requireStringArg(args, "name")
}
