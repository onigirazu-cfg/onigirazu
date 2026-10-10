package inventory

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/internal/tfstate"
)

const tfState = `{"version": 4, "terraform_version": "1.14.0", "outputs": {"vpc": {"value": "vpc-1", "type": "string"}},
 "resources": [
  {"mode": "managed", "type": "vsphere_virtual_machine", "name": "web", "provider": "p", "instances": [
    {"index_key": 0, "attributes": {"name": "web1", "default_ip_address": "10.0.0.11", "folder": "Prod/Web"}},
    {"index_key": 1, "attributes": {"name": "web2", "default_ip_address": "", "guest_ip_addresses": ["10.0.0.12"], "folder": "Prod/Web"}}]},
  {"mode": "managed", "type": "aws_instance", "name": "bastion", "module": "module.edge", "provider": "p", "instances": [
    {"attributes": {"id": "i-1", "tags": {"Name": "bastion"}, "public_ip": "203.0.113.5", "private_ip": "10.1.0.5"}}]},
  {"mode": "data", "type": "aws_ami", "name": "x", "instances": [{"attributes": {"id": "ami"}}]},
  {"mode": "managed", "type": "ansible_host", "name": "db", "provider": "ansible", "instances": [
    {"attributes": {"name": "db1", "groups": ["databases"], "variables": {"ansible_host": "10.0.0.20", "ansible_user": "pg"}}}]},
  {"mode": "managed", "type": "ansible_group", "name": "dbs", "provider": "ansible", "instances": [
    {"attributes": {"name": "databases", "variables": {"pg_version": "16"}}}]}
 ]}`

func loadTF(t *testing.T, cfg map[string]interface{}) map[string]interface{} {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "terraform.tfstate"), []byte(tfState), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg["_dir"] = dir
	out, err := loadTerraformInventory(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	var inv map[string]interface{}
	if err := json.Unmarshal(out, &inv); err != nil {
		t.Fatal(err)
	}
	return inv
}

func hostvars(inv map[string]interface{}, host string) map[string]interface{} {
	return inv["_meta"].(map[string]interface{})["hostvars"].(map[string]interface{})[host].(map[string]interface{})
}

func tfGroup(inv map[string]interface{}, group string) []interface{} {
	g, ok := inv[group].(map[string]interface{})
	if !ok {
		return nil
	}
	return g["hosts"].([]interface{})
}

func TestTerraformInventoryDefaults(t *testing.T) {
	inv := loadTF(t, map[string]interface{}{"plugin": "terraform"})
	if len(tfGroup(inv, "all")) != 4 {
		t.Fatalf("hosts: %v", tfGroup(inv, "all"))
	}
	w1 := hostvars(inv, "web1")
	if w1["ansible_host"] != "10.0.0.11" || w1["terraform_type"] != "vsphere_virtual_machine" || w1["terraform_address"] != "vsphere_virtual_machine.web[0]" {
		t.Errorf("web1: %v", w1)
	}
	if hostvars(inv, "web2")["ansible_host"] != "10.0.0.12" {
		t.Errorf("the fallback path guest_ip_addresses.0: %v", hostvars(inv, "web2"))
	}
	b := hostvars(inv, "bastion")
	if b["ansible_host"] != "203.0.113.5" || b["terraform_module"] != "module.edge" {
		t.Errorf("bastion: %v", b)
	}
	if len(tfGroup(inv, "module_edge")) != 1 || len(tfGroup(inv, "tf_vsphere_virtual_machine")) != 2 {
		t.Errorf("groups: %v", inv)
	}
	// the ansible provider's resources
	db := hostvars(inv, "db1")
	if db["ansible_host"] != "10.0.0.20" || db["ansible_user"] != "pg" || db["pg_version"] != "16" {
		t.Errorf("ansible_host resource: %v", db)
	}
	if len(tfGroup(inv, "databases")) != 1 {
		t.Errorf("ansible_group: %v", inv["databases"])
	}
	if _, data := inv["tf_aws_ami"]; data {
		t.Error("data sources are not hosts")
	}
}

func TestTerraformInventoryRules(t *testing.T) {
	inv := loadTF(t, map[string]interface{}{"plugin": "terraform", "group_by": []interface{}{"type"},
		"hosts": []interface{}{map[string]interface{}{"type": "vsphere_virtual_machine", "groups": []interface{}{"vsphere", "{{ folder }}"},
			"vars": map[string]interface{}{"ansible_user": "ubuntu", "vm_folder": "{{ folder }}"}}}})
	if len(tfGroup(inv, "all")) != 3 { // the two VMs and the ansible_host resource, always a host
		t.Fatalf("only the listed type: %v", tfGroup(inv, "all"))
	}
	if len(tfGroup(inv, "prod_web")) != 2 || len(tfGroup(inv, "vsphere")) != 2 {
		t.Errorf("groups from templates: %v", inv)
	}
	w := hostvars(inv, "web1")
	if w["ansible_user"] != "ubuntu" || w["vm_folder"] != "Prod/Web" || w["ansible_host"] != "10.0.0.11" {
		t.Errorf("vars and the default address: %v", w)
	}
}

func TestTerraformOutputsFromState(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "terraform.tfstate")
	if err := os.WriteFile(p, []byte(tfState), 0o600); err != nil {
		t.Fatal(err)
	}
	out, err := tfstate.Outputs(context.Background(), "", p, "")
	if err != nil || out["vpc"] != "vpc-1" {
		t.Errorf("outputs: %v %v", out, err)
	}
}

func TestTfAddress(t *testing.T) {
	for in, want := range map[string]string{"10.0.0.5/24": "10.0.0.5", "dhcp": "", " 192.168.1.9 ": "192.168.1.9", "": ""} {
		if got := tfAddress(in); got != want {
			t.Errorf("tfAddress(%q) = %q, want %q", in, got, want)
		}
	}
}
