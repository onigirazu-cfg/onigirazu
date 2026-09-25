package modules

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dbStub answers the few queries the database modules make, keeping state in
// $STUB_DIR, and logs every call with the password environment it saw
const dbStub = `#!/bin/sh
d="$STUB_DIR"; q=""
prev=""
for a in "$@"; do
  case "$prev" in -e|-c|--eval) q="$a" ;; esac
  prev="$a"
done
echo "$(basename "$0") $* pwd=${MYSQL_PWD:-}${PGPASSWORD:-}" >> "$d/log"
case "$q" in
  *information_schema.SCHEMATA*) [ -e "$d/db" ] && printf 'app\tutf8mb4\tutf8mb4_unicode_ci\n' ;;
  "CREATE DATABASE"*) touch "$d/db" ;;
  "DROP DATABASE"*) rm -f "$d/db" ;;
  *"FROM mysql.user"*|*"FROM pg_roles"*) [ -e "$d/user" ] && echo 1 || echo 0 ;;
  *"FROM pg_database"*) [ -e "$d/db" ] && echo 1 || echo 0 ;;
  "CREATE USER"*|"CREATE ROLE"*) touch "$d/user" ;;
  "DROP USER"*|"DROP ROLE"*) rm -f "$d/user" ;;
  "SHOW GRANTS"*|*datacl*) cat "$d/grants" 2>/dev/null ;;
  "GRANT"*) grep -qxF "$q" "$d/grants" 2>/dev/null || echo "$q" >> "$d/grants" ;;
  *listDatabases*) [ -e "$d/db" ] && echo true || echo false ;;
  *createCollection*) touch "$d/db" ;;
  *getUser*) [ -e "$d/user" ] && echo true || echo false ;;
  *createUser*) touch "$d/user" ;;
esac
exit 0
`

func withDBStub(t *testing.T) (string, types.Host) {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	require.NoError(t, os.MkdirAll(bin, 0o755))
	for _, name := range []string{"mysql", "psql", "mongosh"} {
		require.NoError(t, os.WriteFile(filepath.Join(bin, name), []byte(dbStub), 0o755))
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("STUB_DIR", dir)
	host := types.Host{Name: "local", Address: "localhost", Vars: map[string]interface{}{"onigirazu_connection": "local"}}
	return dir, host
}

func stubLog(t *testing.T, dir string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, "log"))
	require.NoError(t, err)
	return string(b)
}

// runTwice runs a module twice: the first run must change, the second not
func runTwice(t *testing.T, m types.Module, host types.Host, args map[string]interface{}) {
	t.Helper()
	for i, wantChanged := range []bool{true, false} {
		res, err := m.Execute(context.Background(), host, args)
		require.NoError(t, err, "run %d", i+1)
		require.True(t, res.Success, "run %d: %s", i+1, res.Error)
		assert.Equal(t, wantChanged, res.Changed, "run %d changed", i+1)
	}
}

func TestMySQLModules_LocalSocketAndIdempotent(t *testing.T) {
	dir, host := withDBStub(t)
	runTwice(t, NewMySQLDBModule(), host, map[string]interface{}{"name": "app"})
	runTwice(t, NewMySQLUserModule(), host, map[string]interface{}{
		"name": "app", "password": "p'w", "priv": "app.*:ALL",
	})

	log := stubLog(t, dir)
	assert.NotContains(t, log, " -h ", "no host means the local socket")
	assert.NotContains(t, log, " -P ")
	assert.Contains(t, log, "IDENTIFIED BY 'p''w'")

	res, err := NewMySQLDBModule().Execute(context.Background(), host, map[string]interface{}{
		"name": "app", "state": "absent", "login_password": "secret",
	})
	require.NoError(t, err)
	assert.True(t, res.Changed)
	assert.Contains(t, stubLog(t, dir), "pwd=secret", "password travels in the environment")
	assert.NotContains(t, stubLog(t, dir), "-psecret")
}

func TestPostgreSQLModules_Idempotent(t *testing.T) {
	dir, host := withDBStub(t)
	runTwice(t, NewPostgreSQLDBModule(), host, map[string]interface{}{"name": "app", "encoding": "UTF8"})
	runTwice(t, NewPostgreSQLUserModule(), host, map[string]interface{}{
		"name": "app", "password": "pw", "db": "app", "priv": "ALL",
	})
	log := stubLog(t, dir)
	assert.Contains(t, log, `CREATE DATABASE "app" ENCODING 'UTF8' TEMPLATE template0`)
	assert.NotContains(t, log, " -h ")

	_, err := NewPostgreSQLUserModule().Execute(context.Background(), host, map[string]interface{}{
		"name": "app", "priv": "ALL; DROP TABLE x", "db": "app",
	})
	assert.Error(t, err, "privileges are validated")
}

func TestMongoDBModule_Idempotent(t *testing.T) {
	dir, host := withDBStub(t)
	runTwice(t, NewMongoDBModule(), host, map[string]interface{}{"name": "app"})
	runTwice(t, NewMongoDBModule(), host, map[string]interface{}{
		"operation": "user", "name": "app", "database": "app", "password": "p\"w", "roles": []interface{}{"readWrite"},
	})
	assert.True(t, strings.Contains(stubLog(t, dir), `"pwd":"p\"w"`), "values are JSON-encoded")
}

// dockerStub plays "docker compose": up starts one container, ps lists it
const dockerStub = `#!/bin/sh
d="$STUB_DIR"
echo "docker $*" >> "$d/log"
[ "$1" = compose ] || exit 1
shift
case "$*" in
  version) echo "Docker Compose version v2" ;;
  *"ps -a -q"*|*"ps -q"*) cat "$d/containers" 2>/dev/null ;;
  *"up -d"*) [ -e "$d/containers" ] || echo c1 > "$d/containers" ;;
  *down*) rm -f "$d/containers" ;;
esac
exit 0
`

func TestDockerComposeModule_Idempotent(t *testing.T) {
	dir, host := withDBStub(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bin", "docker"), []byte(dockerStub), 0o755))
	args := map[string]interface{}{"project_dir": dir, "project_name": "app"}
	runTwice(t, NewDockerComposeModule(), host, args)

	args["state"] = "absent"
	runTwice(t, NewDockerComposeModule(), host, args)
	assert.Contains(t, stubLog(t, dir), "docker compose -p app up -d")
}
