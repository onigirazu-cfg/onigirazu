package inventory

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNetboxInventory(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Token secret" {
			w.WriteHeader(403)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		q := r.URL.Query()
		switch {
		case r.URL.Path == "/api/dcim/devices/" && q.Get("offset") == "":
			if q.Get("status") != "active" || q.Get("site") != "fra1" {
				t.Errorf("filters: %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"next": "` + srv.URL + `/api/dcim/devices/?offset=1", "results": [
			  {"id": 1, "name": "sw1", "site": {"slug": "fra1"}, "role": {"slug": "switch"}, "platform": {"slug": "nxos"},
			   "status": {"value": "active"}, "primary_ip4": {"address": "10.0.0.1/24"}, "device_type": {"model": "N9K"},
			   "serial": "ABC", "tags": [{"slug": "core"}, {"slug": "managed"}], "custom_fields": {"rack_unit": 3}}]}`))
		case r.URL.Path == "/api/dcim/devices/":
			_, _ = w.Write([]byte(`{"next": null, "results": [
			  {"id": 2, "name": "noip", "site": {"slug": "fra1"}, "role": {"slug": "server"}, "status": {"value": "active"}, "tags": []}]}`))
		case r.URL.Path == "/api/virtualization/virtual-machines/":
			_, _ = w.Write([]byte(`{"next": null, "results": [
			  {"id": 7, "name": "vm1", "site": {"slug": "fra1"}, "role": {"slug": "server"}, "cluster": {"name": "RED"},
			   "status": {"value": "active"}, "primary_ip4": {"address": "10.0.0.7/24"}, "primary_ip6": {"address": "fd00::7/64"}, "tags": [{"slug": "managed"}]}]}`))
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	t.Setenv("NETBOX_TOKEN", "secret")
	cfg := map[string]interface{}{"plugin": "netbox", "url": srv.URL + "/", "token": "${NETBOX_TOKEN}",
		"filters": map[string]interface{}{"site": "fra1"}, "group_by": []interface{}{"site", "role", "tags", "type"}}
	out, err := loadNetboxInventory(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	var inv map[string]interface{}
	if err := json.Unmarshal(out, &inv); err != nil {
		t.Fatal(err)
	}
	hostvars := inv["_meta"].(map[string]interface{})["hostvars"].(map[string]interface{})
	sw1 := hostvars["sw1"].(map[string]interface{})
	if sw1["ansible_host"] != "10.0.0.1" || sw1["netbox_device_type"] != "N9K" || sw1["netbox_serial"] != "ABC" {
		t.Errorf("sw1: %v", sw1)
	}
	vm1 := hostvars["vm1"].(map[string]interface{})
	if vm1["ansible_host"] != "10.0.0.7" || vm1["netbox_primary_ip6"] != "fd00::7" || vm1["netbox_cluster"] != "RED" || vm1["netbox_type"] != "virtual_machine" {
		t.Errorf("vm1: %v", vm1)
	}
	if _, ok := hostvars["noip"].(map[string]interface{})["ansible_host"]; ok {
		t.Error("a device without a primary IP keeps its name as the address")
	}
	hosts := func(g string) string {
		m, _ := inv[g].(map[string]interface{})
		b, _ := json.Marshal(m["hosts"])
		return string(b)
	}
	for g, want := range map[string]string{"all": `["noip","sw1","vm1"]`, "site_fra1": `["noip","sw1","vm1"]`, "role_switch": `["sw1"]`,
		"tag_managed": `["sw1","vm1"]`, "tag_core": `["sw1"]`, "device": `["noip","sw1"]`, "virtual_machine": `["vm1"]`} {
		if hosts(g) != want {
			t.Errorf("group %s: %s, want %s", g, hosts(g), want)
		}
	}
	// a bad token is an error, not an empty inventory
	t.Setenv("NETBOX_TOKEN", "wrong")
	if _, err := loadNetboxInventory(context.Background(), cfg); err == nil || !strings.Contains(err.Error(), "403") {
		t.Errorf("expected HTTP 403, got %v", err)
	}
}

func TestPluginConfigAndHelpers(t *testing.T) {
	name, cfg, err := pluginConfig([]byte("plugin: netbox\nurl: https://nb\n"))
	if err != nil || name != "netbox" || cfg["url"] != "https://nb" {
		t.Fatalf("%q %v %v", name, cfg, err)
	}
	if name, _, err := pluginConfig([]byte("all:\n  hosts:\n    h1:\n")); err != nil || name != "" {
		t.Fatalf("plain inventory: %q %v", name, err)
	}
	if _, _, err := pluginConfig([]byte("plugin: nope\n")); err == nil {
		t.Fatal("unknown plugin must be an error")
	}
	t.Setenv("T_SECRET", "s3")
	for in, want := range map[string]string{"${T_SECRET}": "s3", "env:T_SECRET": "s3", "plain": "plain", "": "s3"} {
		if got := secretArg(map[string]interface{}{"token": in}, "token", "T_SECRET"); got != want {
			t.Errorf("secretArg(%q) = %q", in, got)
		}
	}
	if groupName("Fra 1 / Rack-A") != "fra_1_rack_a" {
		t.Errorf("groupName: %q", groupName("Fra 1 / Rack-A"))
	}
}
