package ssh

import (
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
