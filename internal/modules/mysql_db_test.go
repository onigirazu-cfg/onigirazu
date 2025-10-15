package modules

import (
	"context"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestMySQLDBModule_Execute(t *testing.T) {
	module := NewMySQLDBModule()
	ctx := context.Background()
	host := types.Host{
		Name:    "localhost",
		Address: "127.0.0.1",
	}

	tests := []struct {
		name        string
		args        map[string]interface{}
		expectError bool
	}{
		{
			name: "missing database name",
			args: map[string]interface{}{
				"state": "present",
			},
			expectError: true,
		},
		{
			name: "create database",
			args: map[string]interface{}{
				"name":           "testdb",
				"state":          "present",
				"login_user":     "root",
				"login_password": "password",
			},
			expectError: false,
		},
		{
			name: "create database with charset",
			args: map[string]interface{}{
				"name":           "testdb_utf8",
				"state":          "present",
				"charset":        "utf8mb4",
				"collation":      "utf8mb4_general_ci",
				"login_user":     "root",
				"login_password": "password",
			},
			expectError: false,
		},
		{
			name: "drop database",
			args: map[string]interface{}{
				"name":           "testdb",
				"state":          "absent",
				"login_user":     "root",
				"login_password": "password",
			},
			expectError: false,
		},
		{
			name: "dump database",
			args: map[string]interface{}{
				"name":           "testdb",
				"state":          "dump",
				"target":         "/tmp/testdb.sql",
				"login_user":     "root",
				"login_password": "password",
			},
			expectError: false,
		},
		{
			name: "import database",
			args: map[string]interface{}{
				"name":           "testdb",
				"state":          "import",
				"target":         "/tmp/testdb.sql",
				"login_user":     "root",
				"login_password": "password",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := module.Execute(ctx, host, tt.args)

			if tt.expectError {
				assert.Error(t, err)
				assert.False(t, result.Success)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, "mysql_db", result.Module)
			}
		})
	}
}

func TestMySQLDBModule_GetName(t *testing.T) {
	module := NewMySQLDBModule()
	assert.Equal(t, "mysql_db", module.GetName())
}

func TestMySQLDBModule_GetDescription(t *testing.T) {
	module := NewMySQLDBModule()
	assert.Equal(t, "Manage MySQL databases", module.GetDescription())
}
