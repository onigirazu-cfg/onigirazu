package modules

import (
	"strings"
	"testing"
)

func TestYumRepoContentAndSections(t *testing.T) {
	c := yumRepoContent(map[string]interface{}{"name": "epel", "description": "EPEL", "baseurl": "https://x/epel", "enabled": true, "gpgcheck": false, "exclude": []interface{}{"a", "b"}})
	want := "[epel]\nname=EPEL\nbaseurl=https://x/epel\nenabled=1\ngpgcheck=0\nexclude=a b\n"
	if c != want {
		t.Errorf("repo:\n%s", c)
	}
	s := splitIniSections("[a]\nk=1\n\n[b]\nk=2\n")
	if len(s) != 2 || !strings.HasPrefix(s["b"], "[b]\n") {
		t.Errorf("sections: %v", s)
	}
	s["b"] = "[b]\nk=3\n"
	if j := joinIniSections(s); j != "[a]\nk=1\n\n[b]\nk=3\n" {
		t.Errorf("joined:\n%s", j)
	}
}

func TestIptablesRule(t *testing.T) {
	r := strings.Join(iptablesRule(map[string]interface{}{"protocol": "tcp", "destination_port": 22, "ctstate": []interface{}{"NEW", "ESTABLISHED"}, "source": "!10.0.0.0/8", "comment": "ssh", "jump": "ACCEPT"}), " ")
	for _, want := range []string{"-p tcp", "--dport 22", "-m conntrack --ctstate NEW,ESTABLISHED", "! -s 10.0.0.0/8", "-m comment --comment ssh", "-j ACCEPT"} {
		if !strings.Contains(r, want) {
			t.Errorf("rule lacks %q: %s", want, r)
		}
	}
}

func TestExpectScript(t *testing.T) {
	s := expectScript("passwd bob", map[string]interface{}{"(?i)password": "s3cret"}, 20)
	for _, want := range []string{"set timeout 20", `spawn sh -c "passwd bob"`, `-re "(?i)password" { send -- "s3cret\r"; exp_continue }`, "timeout { exit 124 }"} {
		if !strings.Contains(s, want) {
			t.Errorf("script lacks %q:\n%s", want, s)
		}
	}
}

func TestParseServiceAndMountFacts(t *testing.T) {
	sv := parseServiceFacts("ssh.service\tactive\trunning\tsystemd\nssh.service\tfile\tenabled\tsystemd\ncron.service\tinactive\tdead\tsystemd\n")
	ssh := sv["ssh.service"].(map[string]interface{})
	if ssh["state"] != "running" || ssh["status"] != "enabled" || sv["cron.service"].(map[string]interface{})["state"] != "stopped" {
		t.Errorf("services: %v", sv)
	}
	mp := parseMountFacts("/dev/sda1 / ext4 rw,relatime 0 0\ntmpfs /run tmpfs rw 0 0\n", []string{"ext4"}, nil)
	if len(mp) != 1 || mp["/"].(map[string]interface{})["device"] != "/dev/sda1" {
		t.Errorf("mounts: %v", mp)
	}
}
