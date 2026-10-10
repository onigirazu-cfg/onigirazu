package modules

import (
	"context"
	"strings"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestDBConnUnixSocket(t *testing.T) {
	c := dbConnFromArgs(map[string]interface{}{"login_user": "root", "login_unix_socket": "/run/mysqld/mysqld.sock"})
	if got := c.mysqlClient("mysql"); got != "mysql -u 'root' --socket='/run/mysqld/mysqld.sock'" {
		t.Fatalf("mysql: %s", got)
	}
	if got := c.psqlArgs(); got != " -U 'root' -h '/run/mysqld/mysqld.sock'" {
		t.Fatalf("psql: %s", got)
	}
	// an explicit host wins over the socket for psql
	c = dbConnFromArgs(map[string]interface{}{"login_host": "db", "login_unix_socket": "/x"})
	if got := c.psqlArgs(); got != " -h 'db'" {
		t.Fatalf("psql host: %s", got)
	}
}

func TestValidateBeforeWriteNeedsPlaceholder(t *testing.T) {
	err := validateBeforeWrite(context.Background(), types.Host{Name: "h"}, map[string]interface{}{"validate": "sshd -t"}, "/etc/x", []byte("x"))
	if err == nil || !strings.Contains(err.Error(), "%s") {
		t.Fatalf("expected the %%s error, got %v", err)
	}
	// nothing to validate: no host contact
	if err := validateBeforeWrite(context.Background(), types.Host{Name: "h"}, map[string]interface{}{}, "/etc/x", nil); err != nil {
		t.Fatal(err)
	}
	if err := validateBeforeWrite(context.Background(), types.Host{Name: "h"}, map[string]interface{}{"validate": "true %s", "_check_mode": true}, "/etc/x", nil); err != nil {
		t.Fatal(err)
	}
}
