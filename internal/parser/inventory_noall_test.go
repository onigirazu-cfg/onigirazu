package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAnsibleYAMLInventoryWithoutAll(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "hosts.yml")
	if err := os.WriteFile(p, []byte(`web:
  hosts:
    web1: {ansible_host: 10.0.0.1}
    web2:
  vars: {role: web}
db:
  children:
    pg: {hosts: {db1: {}}}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	inv, err := NewInventoryParser(&mockLogger{}).ParseInventoryFile(t.Context(), p)
	if err != nil {
		t.Fatal(err)
	}
	if len(inv.Hosts) != 3 {
		t.Fatalf("hosts: %v", inv.Hosts)
	}
	if g := inv.Groups["web"]; g == nil || len(g.Hosts) != 2 || g.Vars["role"] != "web" {
		t.Errorf("web group: %+v", g)
	}
	if g := inv.Groups["pg"]; g == nil || len(g.Hosts) != 1 {
		t.Errorf("nested group: %+v", g)
	}
}
