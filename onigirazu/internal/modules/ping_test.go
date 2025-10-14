package modules

import (
	"context"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestPingModule_Execute_Local(t *testing.T) {
	module := NewPingModule()

	host := types.Host{
		Name:    "localhost",
		Address: "127.0.0.1",
	}

	args := map[string]interface{}{
		"name": "test ping",
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	if result.Changed {
		t.Errorf("Expected changed=false, got changed=true")
	}

	// Check output
	if ping, ok := result.Output["ping"].(string); !ok || ping != "pong" {
		t.Errorf("Expected ping='pong', got %v", result.Output["ping"])
	}

	if conn, ok := result.Output["connection"].(string); !ok || conn != "local" {
		t.Errorf("Expected connection='local', got %v", result.Output["connection"])
	}
}

func TestPingModule_Execute_CustomData(t *testing.T) {
	module := NewPingModule()

	host := types.Host{
		Name:    "localhost",
		Address: "127.0.0.1",
	}

	args := map[string]interface{}{
		"name": "test ping",
		"data": "custom response",
	}

	ctx := context.Background()
	result, err := module.Execute(ctx, host, args)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success, got failure: %s", result.Error)
	}

	// Check custom data
	if ping, ok := result.Output["ping"].(string); !ok || ping != "custom response" {
		t.Errorf("Expected ping='custom response', got %v", result.Output["ping"])
	}
}

func TestPingModule_Validate(t *testing.T) {
	module := NewPingModule()

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid - no args",
			args: map[string]interface{}{
				"name": "test",
			},
			wantErr: false,
		},
		{
			name: "valid - with data",
			args: map[string]interface{}{
				"name": "test",
				"data": "custom",
			},
			wantErr: false,
		},
		{
			name:    "invalid - missing name",
			args:    map[string]interface{}{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := module.Validate(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPingModule_GetDescription(t *testing.T) {
	module := NewPingModule()
	desc := module.GetDescription()

	if desc == "" {
		t.Error("Expected non-empty description")
	}
}
