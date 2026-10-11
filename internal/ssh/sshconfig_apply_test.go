package ssh

import (
	"strings"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestApplySSHConfig(t *testing.T) {
	cfg := map[string]string{"HostName": "10.9.9.9", "Port": "2222", "User": "deploy", "ProxyJump": "bastion"}
	get := func(alias, key string) string {
		if alias == "web1" {
			return cfg[key]
		}
		return ""
	}
	h := types.Host{Name: "web1", Address: "web1"}
	applySSHConfig(&h, get)
	if h.Address != "10.9.9.9" || h.Port != 2222 || h.User != "deploy" || h.SSHArgs != "-J bastion" {
		t.Errorf("alias filled from the config: %+v", h)
	}
	// the inventory wins where it says something
	h = types.Host{Name: "web1", Address: "192.168.1.5", Port: 22, User: "root", SSHArgs: "-o ProxyJump=other", Vars: map[string]interface{}{"ansible_port": 22}}
	applySSHConfig(&h, get)
	if h.Address != "192.168.1.5" || h.Port != 22 || h.User != "root" || h.SSHArgs != "-o ProxyJump=other" {
		t.Errorf("the inventory's values stay: %+v", h)
	}
	// an unknown alias changes nothing
	h = types.Host{Name: "other", Address: "other"}
	applySSHConfig(&h, get)
	if h.Address != "other" || h.Port != 0 {
		t.Errorf("unknown alias: %+v", h)
	}
}

// ansible_host may be the ssh config alias rather than the inventory name
func TestApplySSHConfigAliasInAddress(t *testing.T) {
	cfg := map[string]string{"mjodnet-oci1/HostName": "129.151.153.153", "mjodnet-oci1/Port": "47931", "mjodnet-oci1/User": "usx", "mjodnet-oci1/ProxyJump": "cs"}
	get := func(alias, key string) string { return cfg[alias+"/"+key] }
	h := &types.Host{Name: "oci1", Address: "mjodnet-oci1", Port: 22, Vars: map[string]interface{}{}}
	applySSHConfig(h, get)
	if h.Address != "129.151.153.153" || h.Port != 47931 || h.User != "usx" || !strings.Contains(h.SSHArgs, "-J cs") {
		t.Errorf("alias in ansible_host: %+v", h)
	}
	ip := &types.Host{Name: "x", Address: "10.0.0.1", Port: 22, Vars: map[string]interface{}{}}
	applySSHConfig(ip, get)
	if ip.Address != "10.0.0.1" {
		t.Errorf("an IP is never an alias: %s", ip.Address)
	}
}
