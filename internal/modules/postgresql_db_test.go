package modules

import (
	"context"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestPostgreSQLDBModule_Execute(t *testing.T) {
	module := NewPostgreSQLDBModule()
	ctx := context.Background()
	host := types.Host{Name: "localhost", Address: "127.0.0.1"}

	tests := []struct {
		name        string
		args        map[string]interface{}
		expectError bool
	}{
		{name: "missing database name", args: map[string]interface{}{"state": "present"}, expectError: true},
		{name: "create database", args: map[string]interface{}{"name": "testdb", "state": "present", "login_user": "postgres"}, expectError: false},
		{name: "create with owner", args: map[string]interface{}{"name": "testdb", "owner": "testuser", "state": "present", "login_user": "postgres"}, expectError: false},
		{name: "drop database", args: map[string]interface{}{"name": "testdb", "state": "absent", "login_user": "postgres"}, expectError: false},
		{name: "dump database", args: map[string]interface{}{"name": "testdb", "state": "dump", "target": "/tmp/testdb.sql", "login_user": "postgres"}, expectError: false},
		{name: "restore database", args: map[string]interface{}{"name": "testdb", "state": "restore", "target": "/tmp/testdb.sql", "login_user": "postgres"}, expectError: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := module.Execute(ctx, host, tt.args)
			if tt.expectError {
				assert.Error(t, err)
				assert.False(t, result.Success)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, "postgresql_db", result.Module)
			}
		})
	}
}

func TestPostgreSQLDBModule_GetName(t *testing.T) {
	assert.Equal(t, "postgresql_db", NewPostgreSQLDBModule().GetName())
}
