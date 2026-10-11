package modules

import (
	"strings"
	"testing"
)

func TestDeb822Content(t *testing.T) {
	args := map[string]interface{}{"name": "docker", "types": []interface{}{"deb"}, "uris": []interface{}{"https://download.docker.com/linux/ubuntu"},
		"suites": []interface{}{"noble"}, "components": []interface{}{"stable"}, "architectures": "amd64", "enabled": false}
	got := deb822Content(args, "/etc/apt/keyrings/docker.asc")
	want := "Types: deb\nURIs: https://download.docker.com/linux/ubuntu\nSuites: noble\nComponents: stable\nArchitectures: amd64\nSigned-By: /etc/apt/keyrings/docker.asc\nEnabled: no\n"
	if got != want {
		t.Errorf("content:\n%s\nwant:\n%s", got, want)
	}
	inline := deb822Content(map[string]interface{}{"uris": []interface{}{"u"}, "suites": []interface{}{"s"}}, "-----BEGIN PGP PUBLIC KEY BLOCK-----\nabc\n\ndef\n-----END PGP PUBLIC KEY BLOCK-----")
	if !strings.Contains(inline, "Signed-By:\n -----BEGIN PGP PUBLIC KEY BLOCK-----\n abc\n .\n def\n") || !strings.HasPrefix(inline, "Types: deb\n") {
		t.Errorf("inline key:\n%s", inline)
	}
}

func TestParsePackageFacts(t *testing.T) {
	out := "bash\t5.2-1\tamd64\tapt\nlibc6:i386\t2.39\ti386\tapt\nlibc6\t2.39\tamd64\tapt\n\nbad line\n"
	p := parsePackageFacts(out)
	if len(p) != 2 {
		t.Fatalf("packages: %v", p)
	}
	if libc, _ := p["libc6"].([]interface{}); len(libc) != 2 {
		t.Errorf("both libc6 entries under one name: %v", p["libc6"])
	}
	b := p["bash"].([]interface{})[0].(map[string]interface{})
	if b["version"] != "5.2-1" || b["arch"] != "amd64" || b["source"] != "apt" {
		t.Errorf("bash: %v", b)
	}
}
