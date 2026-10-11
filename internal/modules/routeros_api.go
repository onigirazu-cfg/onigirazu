package modules

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// RouterosAPIModule: items of a RouterOS menu path (community.routeros.api,
// api_modify): path, find (the item: keys and values that identify it),
// values (what it should have), state present/absent; without find, a
// single-object path (system/identity) is set
type RouterosAPIModule struct{ *BaseModule }

func NewRouterosAPIModule() *RouterosAPIModule {
	return &RouterosAPIModule{BaseModule: NewBaseModule("routeros_api")}
}

func (m *RouterosAPIModule) GetDescription() string {
	return "Manage items of a RouterOS menu path over REST"
}

func (m *RouterosAPIModule) Validate(args map[string]interface{}) error {
	if getStringArg(args, "path", "") == "" {
		return fmt.Errorf("routeros_api requires 'path'")
	}
	switch getStringArg(args, "state", "present") {
	case "present", "absent":
		return nil
	}
	return fmt.Errorf("routeros_api: state must be present or absent")
}

func (m *RouterosAPIModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	start := time.Now()
	result := types.TaskResult{TaskName: taskName(args), Host: host.Name, Module: m.name, Timestamp: start, Success: true, Output: map[string]interface{}{}}
	fail := func(msg string) (types.TaskResult, error) {
		result.Success, result.Error, result.Duration = false, msg, time.Since(start)
		return result, nil
	}
	if err := m.Validate(args); err != nil {
		return fail(err.Error())
	}
	r, err := routerosClient(host)
	if err != nil {
		return fail(err.Error())
	}
	path := strings.Trim(strings.ReplaceAll(getStringArg(args, "path", ""), " ", "/"), "/")
	find := stringMap(args["find"])
	values := stringMap(args["values"])
	state := getStringArg(args, "state", "present")
	// the candidates: the path filtered by the find keys
	query := url.Values{}
	for k, v := range find {
		query.Set(k, v)
	}
	items, err := r.items(ctx, path, query)
	if err != nil {
		return fail(err.Error())
	}
	var match map[string]string
	for _, it := range items {
		ok := true
		for k, v := range find {
			if it[k] != v {
				ok = false
			}
		}
		if ok {
			match = it
			break
		}
	}
	single := len(find) == 0 && len(items) == 1 && items[0][".id"] == ""
	if single {
		match = items[0]
	}
	diff := func(have map[string]string) []string {
		var changed []string
		for k, v := range values {
			if have[k] != v {
				changed = append(changed, k)
			}
		}
		sort.Strings(changed)
		return changed
	}
	result.Output["path"] = path
	switch {
	case state == "absent":
		if match == nil || single {
			result.Output["msg"] = "not there"
			break
		}
		result.Changed = true
		result.Output["msg"] = "removed " + match[".id"]
		if !inCheckMode(args) {
			if _, err := r.call(ctx, http.MethodDelete, path+"/"+match[".id"], nil, nil); err != nil {
				return fail(err.Error())
			}
		}
	case match == nil:
		body := map[string]string{}
		for k, v := range find {
			body[k] = v
		}
		for k, v := range values {
			body[k] = v
		}
		result.Changed = true
		result.Output["msg"] = "added"
		if !inCheckMode(args) {
			out, err := r.call(ctx, http.MethodPut, path, nil, body)
			if err != nil {
				return fail(err.Error())
			}
			var created map[string]interface{}
			if json.Unmarshal(out, &created) == nil {
				result.Output["item"] = created
			}
		}
	default:
		changed := diff(match)
		result.Output["item"] = match
		if len(changed) == 0 {
			result.Output["msg"] = "as wanted"
			break
		}
		result.Changed = true
		result.Output["msg"] = "set " + strings.Join(changed, ", ")
		if !inCheckMode(args) {
			body := map[string]string{}
			for _, k := range changed {
				body[k] = values[k]
			}
			target := path
			if id := match[".id"]; id != "" {
				target += "/" + id
			}
			if _, err := r.call(ctx, http.MethodPatch, target, nil, body); err != nil {
				return fail(err.Error())
			}
		}
	}
	result.Duration = time.Since(start)
	return result, nil
}
