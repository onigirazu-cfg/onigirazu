package modules

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMySQLPriv(t *testing.T) {
	grants, err := parseMySQLPriv("app.*:ALL/logs.events:SELECT,INSERT")
	require.NoError(t, err)
	assert.Equal(t, []mysqlGrant{
		{object: "`app`.*", privs: "ALL"},
		{object: "`logs`.`events`", privs: "SELECT,INSERT"},
	}, grants)

	for _, bad := range []string{"app", "app.*", "app.*:ALL; DROP DATABASE x", "*:ALL"} {
		_, err := parseMySQLPriv(bad)
		assert.Error(t, err, bad)
	}
}

func TestDBConnCommands(t *testing.T) {
	// no host/port: the clients use the local socket
	assert.Equal(t, `mysql -N -B -e 'SELECT 1'`, dbConn{}.mysql("SELECT 1"))
	c := dbConnFromArgs(map[string]interface{}{"login_user": "u", "login_password": "p'w", "login_host": "db", "login_port": 3307})
	assert.Equal(t, `MYSQL_PWD='p'\''w' mysql -u 'u' -h 'db' -P 3307 -N -B -e 'SELECT 1'`, c.mysql("SELECT 1"))
	assert.Equal(t, `psql -X -q -v ON_ERROR_STOP=1 -tA -d 'postgres' -c 'SELECT 1'`, dbConn{}.psql("", "SELECT 1"))

	assert.Equal(t, `'it''s \\ ok'`, mysqlString(`it's \ ok`))
	assert.Equal(t, "`a``b`", mysqlIdent("a`b"))
	assert.Equal(t, `"a""b"`, pgIdent(`a"b`))
}
