package modules

import (
	"context"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestMongoDBModule_Execute(t *testing.T) {
	module := NewMongoDBModule()
	ctx := context.Background()
	host := types.Host{Name: "localhost", Address: "127.0.0.1"}

	tests := []struct {
		name        string
		args        map[string]interface{}
		expectError bool
	}{
		{name: "create database", args: map[string]interface{}{"operation": "database", "name": "testdb", "state": "present"}, expectError: false},
		{name: "drop database", args: map[string]interface{}{"operation": "database", "name": "testdb", "state": "absent"}, expectError: false},
		{name: "create user", args: map[string]interface{}{"operation": "user", "name": "testuser", "database": "testdb", "password": "pass123", "roles": []interface{}{"readWrite"}, "state": "present"}, expectError: false},
		{name: "drop user", args: map[string]interface{}{"operation": "user", "name": "testuser", "database": "testdb", "state": "absent"}, expectError: false},
		{name: "missing database name", args: map[string]interface{}{"operation": "database", "state": "present"}, expectError: true},
		{name: "missing user name", args: map[string]interface{}{"operation": "user", "database": "testdb", "state": "present"}, expectError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := module.Execute(ctx, host, tt.args)
			if tt.expectError {
				assert.Error(t, err)
				assert.False(t, result.Success)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, "mongodb", result.Module)
			}
		})
	}
}

func TestMongoDBModule_GetName(t *testing.T) {
	assert.Equal(t, "mongodb", NewMongoDBModule().GetName())
}
