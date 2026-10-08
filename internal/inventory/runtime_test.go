package inventory

import (
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func runtimeManager(t *testing.T) *Manager {
	t.Helper()
	m := NewManager(nil, &mockLogger{}, newMockCache())
	inv := &types.Inventory{Groups: map[string]*types.Group{
		"web": {Name: "web", Hosts: map[string]*types.Host{"w1": {Name: "w1", Address: "10.0.0.1", Vars: map[string]interface{}{}}}},
	}}
	if err := m.SetInventory(inv); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestAddHost(t *testing.T) {
	m := runtimeManager(t)
	if err := m.AddHost("new1", map[string]interface{}{"ansible_host": "10.0.0.9", "ansible_port": "2222", "role": "x"}, []string{"added"}); err != nil {
		t.Fatal(err)
	}
	hosts, _ := m.GetHosts("added")
	if len(hosts) != 1 || hosts[0].Name != "new1" || hosts[0].Address != "10.0.0.9" || hosts[0].Port != 2222 || hosts[0].Vars["role"] != "x" {
		t.Fatalf("added: %+v", hosts)
	}
	all, _ := m.GetHosts("all")
	if len(all) != 2 {
		t.Errorf("all: %d hosts", len(all))
	}
	// a known host gets the variables and the group
	if err := m.AddHost("w1", map[string]interface{}{"extra": 1}, []string{"added"}); err != nil {
		t.Fatal(err)
	}
	hosts, _ = m.GetHosts("added")
	if len(hosts) != 2 {
		t.Errorf("added after w1: %d", len(hosts))
	}
	w1, _ := m.GetHosts("w1")
	if len(w1) != 1 || w1[0].Address != "10.0.0.1" || w1[0].Vars["extra"] != 1 {
		t.Errorf("w1: %+v", w1)
	}
}

func TestAddHostToGroup(t *testing.T) {
	m := runtimeManager(t)
	if err := m.AddHostToGroup("w1", "os_ubuntu", []string{"linux"}); err != nil {
		t.Fatal(err)
	}
	for _, g := range []string{"os_ubuntu", "linux"} {
		if hosts, _ := m.GetHosts(g); len(hosts) != 1 || hosts[0].Name != "w1" {
			t.Errorf("%s: %+v", g, hosts)
		}
	}
	w1, _ := m.GetHosts("w1")
	names, _ := w1[0].Vars["group_names"].([]string)
	if len(names) != 3 {
		t.Errorf("group_names %v", names)
	}
	if err := m.AddHostToGroup("nobody", "g", nil); err == nil {
		t.Error("an unknown host is an error")
	}
}
