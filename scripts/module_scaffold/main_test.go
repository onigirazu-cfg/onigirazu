package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// the generated module and test compile inside the modules package; the
// repository is not touched (go build -overlay)
func TestGeneratedModuleCompiles(t *testing.T) {
	if testing.Short() {
		t.Skip("runs go vet")
	}
	s, err := newSpec("zz_scaffold_check", "", "path,state")
	if err != nil {
		t.Fatal(err)
	}
	files, err := render(s)
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	replace := map[string]string{}
	for name, src := range files {
		tmp := filepath.Join(dir, name)
		if err := os.WriteFile(tmp, src, 0o600); err != nil {
			t.Fatal(err)
		}
		replace[filepath.Join(root, "internal", "modules", name)] = tmp
	}
	overlay := filepath.Join(dir, "overlay.json")
	data, _ := json.Marshal(map[string]interface{}{"Replace": replace})
	if err := os.WriteFile(overlay, data, 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "vet", "-overlay", overlay, "./internal/modules/")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated code does not compile: %v\n%s", err, out)
	}
}

func TestBadNames(t *testing.T) {
	for _, n := range []string{"", "Bad", "has-dash", "1x"} {
		if _, err := newSpec(n, "", "path"); err == nil {
			t.Errorf("%q accepted", n)
		}
	}
	if _, err := newSpec("ok", "", " , "); err == nil {
		t.Error("no params accepted")
	}
}
