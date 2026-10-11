package modules

import (
	"context"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// RouterosFactsModule reads identity, resources, routerboard and interfaces
type RouterosFactsModule struct{ *BaseModule }

func NewRouterosFactsModule() *RouterosFactsModule {
	return &RouterosFactsModule{BaseModule: NewBaseModule("routeros_facts")}
}

func (m *RouterosFactsModule) GetDescription() string { return "Facts of a RouterOS device over REST" }

func (m *RouterosFactsModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	start := time.Now()
	result := types.TaskResult{TaskName: taskName(args), Host: host.Name, Module: m.name, Timestamp: start, Success: true, Output: map[string]interface{}{}}
	r, err := routerosClient(host)
	if err != nil {
		result.Success, result.Error = false, err.Error()
		return result, nil
	}
	facts := map[string]interface{}{}
	for key, path := range map[string]string{"routeros_identity": "system/identity", "routeros_resource": "system/resource", "routeros_routerboard": "system/routerboard"} {
		items, err := r.items(ctx, path, nil)
		if err == nil && len(items) == 1 {
			facts[key] = items[0]
		}
	}
	if items, err := r.items(ctx, "interface", nil); err == nil {
		facts["routeros_interfaces"] = items
	} else {
		result.Success, result.Error = false, err.Error()
		return result, nil
	}
	if id, ok := facts["routeros_identity"].(map[string]string); ok {
		facts["routeros_hostname"] = id["name"]
	}
	if res, ok := facts["routeros_resource"].(map[string]string); ok {
		facts["routeros_version"] = res["version"]
		facts["routeros_board"] = res["board-name"]
	}
	result.Output["ansible_facts"] = facts
	result.Output["msg"] = "facts gathered"
	result.Duration = time.Since(start)
	return result, nil
}
