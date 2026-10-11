package cli

import (
	"strconv"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestBootstrapPlaybook(t *testing.T) {
	o := &bootstrapOptions{newUser: "deploy", shell: "/bin/bash", disablePasswordAuth: true, keepRootLogin: false}
	data, err := bootstrapPlaybook(o, "ssh-ed25519 AAAA test\n")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Plays []struct {
			Become   bool                     `yaml:"become"`
			Tasks    []map[string]interface{} `yaml:"tasks"`
			Handlers []map[string]interface{} `yaml:"handlers"`
		} `yaml:"plays"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	p := doc.Plays[0]
	if !p.Become || len(p.Tasks) != 6 || len(p.Handlers) != 1 {
		t.Errorf("play: become=%v tasks=%d handlers=%d", p.Become, len(p.Tasks), len(p.Handlers))
	}
	text := string(data)
	for _, want := range []string{"deploy ALL=(ALL) NOPASSWD:ALL", "ssh-ed25519 AAAA test", "PasswordAuthentication no", "PermitRootLogin prohibit-password", "visudo -cf %s", "sshd -t -f %s"} {
		if !strings.Contains(text, want) {
			t.Errorf("playbook lacks %q", want)
		}
	}
	short, _ := bootstrapPlaybook(&bootstrapOptions{newUser: "ops", shell: "/bin/sh", keepRootLogin: true}, "k")
	if strings.Contains(string(short), "PasswordAuthentication") || strings.Contains(string(short), "PermitRootLogin") {
		t.Error("sshd is left alone by default")
	}
}

func TestSplitHostPort(t *testing.T) {
	for in, want := range map[string]string{"10.0.0.9": "10.0.0.9 22", "vm:2222": "vm 2222", "[fe80::1]:22": "fe80::1 22", "host:notaport": "host:notaport 22"} {
		h, p := splitHostPort(in)
		if got := h + " " + itoa(p); got != want {
			t.Errorf("%s -> %s, want %s", in, got, want)
		}
	}
}

func itoa(n int) string { return strconv.Itoa(n) }

func TestCommitChanges(t *testing.T) {
	o := &imageOptions{cmd: `["/usr/sbin/nginx"]`, entrypoint: `["/bin/sh","-c"]`, labels: []string{"team=web"}}
	args := strings.Join(commitChanges(o, "/x/app.yml"), " ")
	for _, want := range []string{`--change CMD ["/usr/sbin/nginx"]`, `--change ENTRYPOINT ["/bin/sh","-c"]`, "--change LABEL team=web", "--change LABEL onigirazu.playbook=app.yml", "LABEL onigirazu.built="} {
		if !strings.Contains(args, want) {
			t.Errorf("commit args lack %q: %s", want, args)
		}
	}
}
