package expression

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLookups221(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "users.csv"), []byte("alice,admin,30\nbob,user,25\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app.ini"), []byte("[global]\nname=plain\n[db]\nhost = db1\nport: 5432\nhost_ro = db2\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app.properties"), []byte("# comment\nuser=svc\n"), 0o644))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("l1\nl2\n")) }))
	defer srv.Close()
	vars := map[string]interface{}{
		"playbook_dir": dir, "app_name": "web", "app_port": 80, "other": 1,
		"groups": map[string]interface{}{
			"all": []interface{}{"a", "b", "c", "d"}, "web": []interface{}{"a", "b"}, "db": []interface{}{"b", "c"},
		},
		"url": srv.URL,
	}
	cases := map[string]interface{}{
		`lookup('vars', 'app_name')`:                                      "web",
		`query('vars', 'app_name', 'app_port')`:                           []interface{}{"web", 80},
		`lookup('vars', 'missing', default='x')`:                          "x",
		`query('varnames', '^app_')`:                                      []interface{}{"app_name", "app_port"},
		`query('inventory_hostnames', 'web')`:                             []interface{}{"a", "b"},
		`query('inventory_hostnames', 'web:&db')`:                         []interface{}{"b"},
		`query('inventory_hostnames', 'all:!web')`:                        []interface{}{"c", "d"},
		`query('inventory_hostnames', 'web:db')`:                          []interface{}{"a", "b", "c"},
		`query('sequence', 'start=1 end=5 stride=2')`:                     []interface{}{"1", "3", "5"},
		`query('sequence', 'count=3 format=node%02d')`:                    []interface{}{"node01", "node02", "node03"},
		`query('sequence', '0-4/2')`:                                      []interface{}{"0", "2", "4"},
		`query('sequence', '3')`:                                          []interface{}{"0", "1", "2", "3"},
		`lookup('csvfile', 'bob', file='users.csv', col=2)`:               "25",
		`lookup('csvfile', 'alice', file='users.csv')`:                    "admin",
		`lookup('csvfile', 'zed', file='users.csv', default='none')`:      "none",
		`lookup('ini', 'host', file='app.ini', section='db')`:             "db1",
		`lookup('ini', 'port', file='app.ini', section='db')`:             "5432",
		`lookup('ini', 'name', file='app.ini')`:                           "plain",
		`query('ini', 'host.*', file='app.ini', section='db', re=True)`:   []interface{}{"db1", "db2"},
		`lookup('ini', 'user', file='app.properties', type='properties')`: "svc",
		`query('url', url)`:                                               []interface{}{"l1", "l2"},
		`lookup('url', url, split_lines=False)`:                           "l1\nl2\n",
		`lookup('config', 'DEFAULT_FORKS')`:                               10,
		`lookup('config', 'NOPE', on_missing='skip') | length`:            0,
	}
	for e, want := range cases {
		got, err := Eval(e, vars)
		require.NoError(t, err, e)
		assert.Equal(t, want, got, e)
	}
	_, err := Eval(`lookup('vars', 'missing')`, vars)
	assert.Error(t, err)

	pw1, err := Eval(`lookup('password', 'creds/db length=12 chars=ascii_letters,digits')`, vars)
	require.NoError(t, err)
	assert.Len(t, pw1, 12)
	pw2, err := Eval(`lookup('password', 'creds/db length=12')`, vars)
	require.NoError(t, err)
	assert.Equal(t, pw1, pw2, "the stored password is reused")
	stored, err := os.ReadFile(filepath.Join(dir, "creds", "db"))
	require.NoError(t, err)
	assert.Equal(t, pw1.(string)+"\n", string(stored))
	dn, err := Eval(`lookup('password', '/dev/null', length=8, chars='digits')`, vars)
	require.NoError(t, err)
	assert.Regexp(t, `^\d{8}$`, dn)
}
