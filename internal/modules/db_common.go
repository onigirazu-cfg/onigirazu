package modules

import (
	"fmt"
	"strings"
)

// dbConn holds the login_* arguments of the database modules. Host and port
// are passed only when set: without them the clients use the local socket,
// where root (MySQL/MariaDB unix_socket) and postgres (peer) log in without a
// password.
type dbConn struct {
	user, password, host, port string
}

func dbConnFromArgs(args map[string]interface{}) dbConn {
	c := dbConn{}
	c.user, _ = args["login_user"].(string)
	c.password, _ = args["login_password"].(string)
	c.host, _ = args["login_host"].(string)
	if p, ok := toInt(args["login_port"]); ok && p > 0 {
		c.port = fmt.Sprint(p)
	}
	return c
}

// mysqlClient returns the command line of a MySQL client binary (mysql,
// mysqldump) with the connection options
func (c dbConn) mysqlClient(bin string) string {
	var b strings.Builder
	if c.password != "" {
		b.WriteString("MYSQL_PWD=" + shellQuote(c.password) + " ")
	}
	b.WriteString(bin)
	if c.user != "" {
		b.WriteString(" -u " + shellQuote(c.user))
	}
	if c.host != "" {
		b.WriteString(" -h " + shellQuote(c.host))
	}
	if c.port != "" {
		b.WriteString(" -P " + c.port)
	}
	return b.String()
}

// mysql returns a shell command that runs sql in batch mode without headers
func (c dbConn) mysql(sql string) string {
	return c.mysqlClient("mysql") + " -N -B -e " + shellQuote(sql)
}

// psqlArgs returns connection options shared by psql, pg_dump and friends
func (c dbConn) psqlArgs() string {
	var b strings.Builder
	if c.user != "" {
		b.WriteString(" -U " + shellQuote(c.user))
	}
	if c.host != "" {
		b.WriteString(" -h " + shellQuote(c.host))
	}
	if c.port != "" {
		b.WriteString(" -p " + c.port)
	}
	return b.String()
}

func (c dbConn) pgEnv() string {
	if c.password == "" {
		return ""
	}
	return "PGPASSWORD=" + shellQuote(c.password) + " "
}

// psql returns a shell command that runs sql in db (maintenance db if empty)
// and prints unaligned tuples only
func (c dbConn) psql(db, sql string) string {
	if db == "" {
		db = "postgres"
	}
	return c.pgEnv() + "psql -X -q -v ON_ERROR_STOP=1 -tA" + c.psqlArgs() + " -d " + shellQuote(db) + " -c " + shellQuote(sql)
}

// mysqlString quotes a string literal; backslashes are escaped for MySQL, whose
// default mode treats them as escapes
func mysqlString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func mysqlIdent(s string) string {
	return "`" + strings.ReplaceAll(s, "`", "``") + "`"
}

func pgString(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func pgIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}
