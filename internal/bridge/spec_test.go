package bridge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func knownHostsSpec() *Spec {
	return &Spec{Module: "known_hosts", Options: map[string]Option{
		"name":      {Type: "str", Required: true, Aliases: []string{"host"}},
		"key":       {Type: "str"},
		"state":     {Type: "str", Choices: []interface{}{"absent", "present"}},
		"hash_host": {Type: "bool", Choices: []interface{}{true, false}},
	}}
}

func TestCheckArgs(t *testing.T) {
	s := knownHostsSpec()
	msgs := func(args map[string]interface{}) string {
		var out []string
		for _, p := range s.CheckArgs(args) {
			out = append(out, p.Message)
		}
		return strings.Join(out, " | ")
	}
	got := msgs(map[string]interface{}{"nme": "x", "state": "gone", "_task_name": "t"})
	for _, want := range []string{`no argument "nme" (did you mean "name"?)`, `needs argument "name"`, "state is gone, not one of [absent present]"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}
	if got := msgs(map[string]interface{}{"host": "x", "state": "{{ s }}", "hash_host": "yes"}); got != "" {
		t.Errorf("alias, templated choice and yes for a bool: %q", got)
	}
	if got := msgs(map[string]interface{}{"name": "x", "zzzzzz": 1}); strings.Contains(got, "did you mean") {
		t.Errorf("no suggestion for a far name: %q", got)
	}
}

func TestText(t *testing.T) {
	if Text("a") != "a" || Text([]interface{}{"a", "b"}) != "a b" || Text(nil) != "" {
		t.Error("Text")
	}
}

// ansible-doc is called once per module and version; the answer is cached
func TestSpecForCaches(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", filepath.Join(dir, "cache"))
	t.Setenv("HOME", dir)
	calls := filepath.Join(dir, "calls")
	doc := filepath.Join(dir, "ansible-doc")
	script := "#!/bin/sh\nif [ \"$1\" = --version ]; then echo 'ansible-doc [core 9.9]'; exit 0; fi\necho x >> " + calls + "\n" +
		`if [ "$2" = none.such ]; then echo '{}'; exit 0; fi` + "\n" +
		`echo '{"m": {"doc": {"short_description": "demo", "options": {"a": {"type": "str", "required": true}}}}}'` + "\n"
	if err := os.WriteFile(doc, []byte(script), 0o755); err != nil { // #nosec G306 -- a test executable
		t.Fatal(err)
	}
	Configure(Config{AnsiblePlaybook: filepath.Join(dir, "ansible-playbook")})
	defer Configure(Config{})
	specMu.Lock()
	specCache, docVersions = map[string]*Spec{}, map[string]string{}
	specMu.Unlock()
	s, err := SpecFor("m")
	if err != nil || !s.Options["a"].Required || Text(s.Description) != "demo" {
		t.Fatalf("SpecFor = %+v, %v", s, err)
	}
	specMu.Lock()
	specCache = map[string]*Spec{} // the disk cache answers now
	specMu.Unlock()
	if _, err := SpecFor("m"); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(calls); strings.Count(string(data), "x") != 1 {
		t.Errorf("ansible-doc ran %d times", strings.Count(string(data), "x"))
	}
	if _, err := SpecFor("none.such"); err == nil || !strings.Contains(err.Error(), "does not know") {
		t.Errorf("unknown module: %v", err)
	}
}

func TestSpecForUsesConfiguredBinary(t *testing.T) {
	for _, name := range []string{"first", "second"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			script := "#!/bin/sh\nif [ \"$1\" = --version ]; then echo 'ansible-doc [core same-version]'; exit 0; fi\necho '{\"isolated.module\":{\"doc\":{\"short_description\":\"" + name + "\",\"options\":{}}}}'\n"
			if err := os.WriteFile(filepath.Join(dir, "ansible-doc"), []byte(script), 0o755); err != nil {
				t.Fatal(err)
			}
			cfg := Config{AnsiblePlaybook: filepath.Join(dir, "ansible-playbook")}
			spec, err := cfg.SpecFor("isolated.module")
			if err != nil {
				t.Fatal(err)
			}
			if got := Text(spec.Description); got != name {
				t.Fatalf("spec from another binary: %q", got)
			}
		})
	}
}

func TestSpecForCollectionPaths(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(dir, "cache"))
	t.Setenv("ANSIBLE_COLLECTIONS_PATH", "/inherited")
	if err := os.Mkdir("bin", 0o755); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\n" +
		`if [ "$1" = --version ]; then echo 'ansible-doc [core test-paths]'; exit 0; fi` + "\n" +
		`if [ "$ANSIBLE_COLLECTIONS_PATH" = /missing ]; then echo '{}'; exit 0; fi` + "\n" +
		`printf '{"m":{"doc":{"short_description":"%s","options":{}}}}' "$ANSIBLE_COLLECTIONS_PATH"` + "\n"
	if err := os.WriteFile(filepath.Join(dir, "bin", "ansible-doc"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := Config{AnsiblePlaybook: "bin/ansible-playbook", CollectionsPath: []string{"/missing"}}
	if _, err := cfg.SpecFor("collection.paths"); err == nil {
		t.Fatal("missing collection unexpectedly found")
	}
	for _, disk := range []bool{false, true} {
		for _, path := range []string{"/first", "/second", "/first"} {
			if disk {
				specMu.Lock()
				specCache = map[string]*Spec{}
				specMu.Unlock()
			}
			cfg.CollectionsPath = []string{path, path}
			spec, err := cfg.SpecFor("collection.paths")
			if err != nil {
				t.Fatal(err)
			}
			if got := Text(spec.Description); got != path {
				t.Fatalf("disk=%v: description = %q, want %q", disk, got, path)
			}
		}
	}
	cfg.CollectionsPath = []string{"/missing"}
	if _, err := cfg.SpecFor("collection.paths"); err == nil {
		t.Fatal("cached spec returned for missing collection")
	}
	if got := (Config{AnsiblePlaybook: "ansible-playbook"}).docBinary(); got != "ansible-doc" {
		t.Fatalf("bare doc binary = %q", got)
	}
}
