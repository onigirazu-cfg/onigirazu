package inventory

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func decodeInv(t *testing.T, out []byte) (map[string]interface{}, map[string]interface{}) {
	t.Helper()
	var inv map[string]interface{}
	if err := json.Unmarshal(out, &inv); err != nil {
		t.Fatal(err)
	}
	return inv, inv["_meta"].(map[string]interface{})["hostvars"].(map[string]interface{})
}

func groupHosts(inv map[string]interface{}, g string) string {
	m, _ := inv[g].(map[string]interface{})
	b, _ := json.Marshal(m["hosts"])
	return string(b)
}

func TestVsphereInventory(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "POST" && r.URL.Path == "/api/session":
			if u, p, ok := r.BasicAuth(); !ok || u != "ro@vsphere.local" || p != "pw" {
				w.WriteHeader(401)
				return
			}
			_, _ = w.Write([]byte(`"tok123"`))
		case r.Header.Get("vmware-api-session-id") != "tok123":
			w.WriteHeader(401)
		case r.URL.Path == "/api/vcenter/folder":
			_, _ = w.Write([]byte(`[{"folder":"group-v1","name":"Production","type":"VIRTUAL_MACHINE"},{"folder":"group-v2","name":"Lab","type":"VIRTUAL_MACHINE"}]`))
		case r.URL.Path == "/api/vcenter/vm" && r.URL.Query().Get("folders") == "group-v1":
			if r.URL.Query().Get("power_states") != "POWERED_ON" {
				t.Errorf("power filter missing: %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`[{"vm":"vm-1","name":"web1","power_state":"POWERED_ON","cpu_count":2,"memory_size_MiB":2048}]`))
		case r.URL.Path == "/api/vcenter/vm":
			_, _ = w.Write([]byte(`[]`))
		case r.URL.Path == "/api/vcenter/vm/vm-1/guest/identity":
			_, _ = w.Write([]byte(`{"host_name":"web1.example","ip_address":"10.1.1.5","family":"LINUX","full_name":{"default_message":"Ubuntu Linux (64-bit)"}}`))
		case r.URL.Path == "/api/vcenter/vm/vm-1/guest/networking/interfaces":
			_, _ = w.Write([]byte(`[{"mac_address":"00:50:56:aa:bb:cc","ip":{"ip_addresses":[{"ip_address":"fe80::1","state":"PREFERRED"},{"ip_address":"172.17.0.1","state":"PREFERRED"},{"ip_address":"10.1.1.5","state":"PREFERRED"}]}}]`))
		case r.Method == "DELETE" && r.URL.Path == "/api/session":
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	cfg := map[string]interface{}{"url": srv.URL, "user": "ro@vsphere.local", "password": "pw", "group_by": []interface{}{"folder", "power_state", "guest_family"}}
	out, err := loadVsphereInventory(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	inv, hv := decodeInv(t, out)
	web1 := hv["web1"].(map[string]interface{})
	if web1["ansible_host"] != "10.1.1.5" || web1["vsphere_folder"] != "Production" || web1["vsphere_guest_os"] != "Ubuntu Linux (64-bit)" || web1["vsphere_cpu_count"] != float64(2) {
		t.Errorf("web1: %v", web1)
	}
	for g, want := range map[string]string{"folder_production": `["web1"]`, "power_state_powered_on": `["web1"]`, "guest_family_linux": `["web1"]`} {
		if groupHosts(inv, g) != want {
			t.Errorf("group %s: %s", g, groupHosts(inv, g))
		}
	}
	cfg["password"] = "bad"
	if _, err := loadVsphereInventory(context.Background(), cfg); err == nil {
		t.Error("expected a session error")
	}
}

func TestProxmoxInventory(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "PVEAPIToken=inv@pve!oni=s3cret" {
			w.WriteHeader(401)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api2/json/cluster/resources":
			_, _ = w.Write([]byte(`{"data":[
			  {"vmid":100,"name":"db1","node":"pve1","type":"qemu","status":"running","tags":"prod;db","pool":"main"},
			  {"vmid":101,"name":"ct1","node":"pve2","type":"lxc","status":"running","tags":"","pool":""},
			  {"vmid":102,"name":"off","node":"pve1","type":"qemu","status":"stopped"},
			  {"vmid":900,"name":"tpl","node":"pve1","type":"qemu","status":"stopped","template":1}]}`))
		case "/api2/json/nodes/pve1/qemu/100/agent/network-get-interfaces":
			_, _ = w.Write([]byte(`{"data":{"result":[{"name":"lo","ip-addresses":[{"ip-address":"127.0.0.1","ip-address-type":"ipv4"}]},{"name":"ens18","ip-addresses":[{"ip-address":"10.2.0.100","ip-address-type":"ipv4"},{"ip-address":"fe80::1","ip-address-type":"ipv6"}]}]}}`))
		case "/api2/json/nodes/pve2/lxc/101/interfaces":
			_, _ = w.Write([]byte(`{"data":[{"name":"lo","inet":"127.0.0.1/8"},{"name":"eth0","inet":"10.2.0.101/24"}]}`))
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	t.Setenv("PROXMOX_TOKEN_SECRET", "s3cret")
	out, err := loadProxmoxInventory(context.Background(), map[string]interface{}{"url": srv.URL, "user": "inv@pve", "token_id": "oni", "group_by": []interface{}{"node", "type", "tags", "pool"}})
	if err != nil {
		t.Fatal(err)
	}
	inv, hv := decodeInv(t, out)
	if hv["db1"].(map[string]interface{})["ansible_host"] != "10.2.0.100" || hv["ct1"].(map[string]interface{})["ansible_host"] != "10.2.0.101" {
		t.Errorf("addresses: %v %v", hv["db1"], hv["ct1"])
	}
	if _, ok := hv["off"]; ok {
		t.Error("stopped VMs are left out by default")
	}
	if _, ok := hv["tpl"]; ok {
		t.Error("templates are left out")
	}
	for g, want := range map[string]string{"node_pve1": `["db1"]`, "type_lxc": `["ct1"]`, "tag_prod": `["db1"]`, "tag_db": `["db1"]`, "pool_main": `["db1"]`} {
		if groupHosts(inv, g) != want {
			t.Errorf("group %s: %s", g, groupHosts(inv, g))
		}
	}
}

func TestNetbirdInventory(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Token pat" || r.URL.Path != "/api/peers" {
			w.WriteHeader(401)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
		 {"id":"p1","name":"web1","ip":"100.64.0.5","hostname":"web1","dns_label":"web1","connected":true,"os":"Linux Ubuntu 24.04","version":"0.80.0","ssh_enabled":false,"last_seen":"2026-10-09T10:00:00Z","groups":[{"name":"servers"},{"name":"All"}]},
		 {"id":"p2","name":"laptop","ip":"100.64.0.9","hostname":"lap","dns_label":"laptop","connected":false,"os":"Darwin","groups":[{"name":"admins"}]},
		 {"id":"p3","name":"other","ip":"100.64.0.7","hostname":"other","dns_label":"other","connected":true,"os":"Linux","groups":[{"name":"guests"}]}]`))
	}))
	defer srv.Close()
	out, err := loadNetbirdInventory(context.Background(), map[string]interface{}{"url": srv.URL, "token": "pat", "groups": []interface{}{"servers"}, "group_by": []interface{}{"groups", "os"}})
	if err != nil {
		t.Fatal(err)
	}
	inv, hv := decodeInv(t, out)
	if len(hv) != 1 || hv["web1"].(map[string]interface{})["ansible_host"] != "100.64.0.5" {
		t.Errorf("hosts: %v", hv)
	}
	for g, want := range map[string]string{"netbird_servers": `["web1"]`, "netbird_all": `["web1"]`, "os_linux": `["web1"]`} {
		if groupHosts(inv, g) != want {
			t.Errorf("group %s: %s", g, groupHosts(inv, g))
		}
	}
	out, _ = loadNetbirdInventory(context.Background(), map[string]interface{}{"url": srv.URL, "token": "pat", "connected": false})
	_, hv = decodeInv(t, out)
	if len(hv) != 3 {
		t.Errorf("connected: false must list every peer, got %d", len(hv))
	}
}
