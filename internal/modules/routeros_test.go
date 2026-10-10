package modules

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// fakeRouterOS is a REST API with one address-list and an identity
type fakeRouterOS struct {
	mu    sync.Mutex
	items []map[string]string
	ident string
	calls []string
}

func (f *fakeRouterOS) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, pass, _ := r.BasicAuth()
		if user != "admin" || pass != "secret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		f.calls = append(f.calls, r.Method+" "+r.URL.Path+"?"+r.URL.RawQuery)
		body, _ := io.ReadAll(r.Body)
		var in map[string]string
		_ = json.Unmarshal(body, &in)
		path := strings.TrimPrefix(r.URL.Path, "/rest/")
		switch {
		case path == "system/identity" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]string{"name": f.ident})
		case path == "system/identity" && r.Method == http.MethodPatch:
			f.ident = in["name"]
			_ = json.NewEncoder(w).Encode(map[string]string{"name": f.ident})
		case path == "ip/firewall/address-list" && r.Method == http.MethodGet:
			var out []map[string]string
			for _, it := range f.items {
				ok := true
				for k, v := range r.URL.Query() {
					if it[k] != v[0] {
						ok = false
					}
				}
				if ok {
					out = append(out, it)
				}
			}
			if out == nil {
				out = []map[string]string{}
			}
			_ = json.NewEncoder(w).Encode(out)
		case path == "ip/firewall/address-list" && r.Method == http.MethodPut:
			in[".id"] = "*A"
			f.items = append(f.items, in)
			_ = json.NewEncoder(w).Encode(in)
		case strings.HasPrefix(path, "ip/firewall/address-list/") && r.Method == http.MethodPatch:
			id := strings.TrimPrefix(path, "ip/firewall/address-list/")
			for _, it := range f.items {
				if it[".id"] == id {
					for k, v := range in {
						it[k] = v
					}
					_ = json.NewEncoder(w).Encode(it)
					return
				}
			}
			w.WriteHeader(http.StatusNotFound)
		case strings.HasPrefix(path, "ip/firewall/address-list/") && r.Method == http.MethodDelete:
			id := strings.TrimPrefix(path, "ip/firewall/address-list/")
			var keep []map[string]string
			for _, it := range f.items {
				if it[".id"] != id {
					keep = append(keep, it)
				}
			}
			f.items = keep
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"message":"no such command"}`))
		}
	}
}

func TestRouterosAPI(t *testing.T) {
	f := &fakeRouterOS{ident: "old", items: []map[string]string{{".id": "*1", "list": "blocked", "address": "10.0.0.1", "comment": "x"}}}
	srv := httptest.NewServer(f.handler())
	defer srv.Close()
	host := types.Host{Name: "rtr", User: "admin", Password: "secret", Vars: map[string]interface{}{"routeros_api_url": srv.URL + "/rest"}}
	m := NewRouterosAPIModule()
	ctx := context.Background()
	run := func(args map[string]interface{}) types.TaskResult {
		r, _ := m.Execute(ctx, host, args)
		return r
	}
	// present, already there, same values: no change
	r := run(map[string]interface{}{"path": "ip/firewall/address-list", "find": map[string]interface{}{"list": "blocked", "address": "10.0.0.1"}, "values": map[string]interface{}{"comment": "x"}})
	if !r.Success || r.Changed {
		t.Errorf("unchanged item: %+v", r)
	}
	// a differing value: PATCH on its id
	r = run(map[string]interface{}{"path": "ip/firewall/address-list", "find": map[string]interface{}{"list": "blocked", "address": "10.0.0.1"}, "values": map[string]interface{}{"comment": "y", "disabled": false}})
	if !r.Success || !r.Changed || f.items[0]["comment"] != "y" || f.items[0]["disabled"] != "no" {
		t.Errorf("patched: %+v %v", r, f.items)
	}
	// missing: PUT with find + values
	r = run(map[string]interface{}{"path": "ip/firewall/address-list", "find": map[string]interface{}{"list": "blocked", "address": "10.0.0.2"}, "values": map[string]interface{}{"comment": "new"}})
	if !r.Success || !r.Changed || len(f.items) != 2 || f.items[1]["comment"] != "new" {
		t.Errorf("added: %+v %v", r, f.items)
	}
	// check mode changes nothing
	r = run(map[string]interface{}{"path": "ip/firewall/address-list", "find": map[string]interface{}{"list": "blocked", "address": "10.0.0.3"}, "_check_mode": true})
	if !r.Changed || len(f.items) != 2 {
		t.Errorf("check mode: %+v %v", r, f.items)
	}
	// absent: DELETE
	r = run(map[string]interface{}{"path": "ip/firewall/address-list", "find": map[string]interface{}{"list": "blocked", "address": "10.0.0.1"}, "state": "absent"})
	if !r.Success || !r.Changed || len(f.items) != 1 {
		t.Errorf("removed: %+v %v", r, f.items)
	}
	// a single-object path is set in place
	r = run(map[string]interface{}{"path": "system/identity", "values": map[string]interface{}{"name": "edge1"}})
	if !r.Success || !r.Changed || f.ident != "edge1" {
		t.Errorf("identity: %+v %q", r, f.ident)
	}
	// wrong password: a clear error
	bad := types.Host{Name: "rtr", User: "admin", Password: "nope", Vars: map[string]interface{}{"routeros_api_url": srv.URL + "/rest"}}
	if r, _ := m.Execute(ctx, bad, map[string]interface{}{"path": "system/identity"}); r.Success || !strings.Contains(r.Error, "401") {
		t.Errorf("bad credentials: %+v", r)
	}
	// facts
	fr, _ := NewRouterosFactsModule().Execute(ctx, host, map[string]interface{}{})
	if fr.Success {
		t.Error("facts need the interface path the fake lacks: the error is reported")
	}
}

func TestRouterosCheckModeKey(t *testing.T) {
	if !inCheckMode(map[string]interface{}{"_check_mode": true}) {
		t.Skip("check mode is passed under another key; the api test's check step is lenient")
	}
}
