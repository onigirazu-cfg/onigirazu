package modules

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/internal/bridge"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestRegistryBridgeSettingsIsolation(t *testing.T) {
	for _, name := range []string{"first", "second"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			registry := NewRegistry()
			binary := filepath.Join(t.TempDir(), name+"-ansible-playbook")
			registry.SetBridgeConfig(bridge.Config{Modules: []string{name + ".*"}, AnsiblePlaybook: binary})
			result, err := registry.ExecuteTask(context.Background(), &types.Task{Module: name + ".task"}, types.Host{Name: name}, nil)
			if err != nil {
				t.Fatal(err)
			}
			if !result.Failed || !strings.Contains(result.Error, binary) {
				t.Fatalf("wrong bridge executable: %+v", result)
			}
			_, err = registry.ExecuteTask(context.Background(), &types.Task{Module: "other.task"}, types.Host{Name: name}, nil)
			if err == nil {
				t.Fatal("registry allowed another run's module")
			}
		})
	}
}
