package modules

import (
	"context"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestPostgreSQLUserModule_Execute(t *testing.T) {
	module := NewPostgreSQLUserModule()
	ctx := context.Background()
	host := types.Host{Name: "localhost", Address: "127.0.0.1"}

	tests := []struct {
		name        string
		args        map[string]interface{}
		expectError bool
	}{
		{name: "missing user name", args: map[string]interface{}{"state": "present"}, expectError: true},
		{name: "create user", args: map[string]interface{}{"name": "testuser", "password": "pass123", "state": "present", "login_user": "postgres"}, expectError: false},
		{name: "create superuser", args: map[string]interface{}{"name": "admin", "password": "admin123", "superuser": true, "state": "present", "login_user": "postgres"}, expectError: false},
		{name: "grant privileges", args: map[string]interface{}{"name": "testuser", "db": "testdb", "priv": "ALL PRIVILEGES", "state": "present", "login_user": "postgres"}, expectError: false},
		{name: "drop user", args: map[string]interface{}{"name": "testuser", "state": "absent", "login_user": "postgres"}, expectError: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := module.Execute(ctx, host, tt.args)
			if tt.expectError {
				assert.Error(t, err)
				assert.False(t, result.Success)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, "postgresql_user", result.Module)
			}
		})
	}
}

func TestPostgreSQLUserModule_GetName(t *testing.T) {
	assert.Equal(t, "postgresql_user", NewPostgreSQLUserModule().GetName())
}
