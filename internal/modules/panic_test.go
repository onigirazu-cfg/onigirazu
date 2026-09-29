package modules

import (
	"context"
	"strings"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

type panickingModule struct{ *BaseModule }

func (panickingModule) Execute(context.Context, types.Host, map[string]interface{}) (types.TaskResult, error) {
	panic("assignment to entry in nil map")
}
func (panickingModule) Validate(map[string]interface{}) error { return nil }
func (panickingModule) GetDescription() string                { return "panics" }

func TestModulePanicFailsOnlyTheTask(t *testing.T) {
	r := NewRegistry()
	r.RegisterModule(panickingModule{BaseModule: NewBaseModule("zz_panics")})
	host := types.Host{Name: "h1", Address: "localhost", Vars: map[string]interface{}{"ansible_connection": "local"}}
	res, err := r.ExecuteTask(context.Background(), &types.Task{Name: "crash", Module: "zz_panics", Args: map[string]interface{}{}}, host, nil)
	if err == nil || !res.Failed {
		t.Fatalf("a panic must fail the task: %+v, %v", res, err)
	}
	if !strings.Contains(res.Error, "crashed") || !strings.Contains(res.Error, "nil map") {
		t.Errorf("error = %q", res.Error)
	}
}
