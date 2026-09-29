package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestVaultSecretSources(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string, mode os.FileMode) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(content), mode); err != nil {
			t.Fatal(err)
		}
		return p
	}
	plain := write("pw", "from-file\n", 0o600)
	prod := write("prod", "from-prod\n", 0o600)
	envFile := write("envpw", "from-env\n", 0o600)
	cfgPw := write("cfgpw", "from-cfg\n", 0o600)
	write("ansible.cfg", "[defaults]\nvault_password_file = cfgpw\n", 0o600)

	t.Setenv("ANSIBLE_CONFIG", filepath.Join(dir, "ansible.cfg"))
	t.Setenv("ANSIBLE_VAULT_PASSWORD_FILE", envFile)
	t.Setenv("ANSIBLE_VAULT_IDENTITY_LIST", "")
	vaultPasswordFiles, vaultIDs, askVaultPass = []string{plain}, []string{"prod@" + prod}, false
	defer func() { vaultPasswordFiles, vaultIDs = nil, nil }()

	got, err := vaultSecrets()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"prod=from-prod", "default=from-file", "default=from-env", "default=from-cfg"}
	if len(got) != len(want) {
		t.Fatalf("got %d secrets: %+v", len(got), got)
	}
	for i, s := range got {
		if s.Label+"="+string(s.Password) != want[i] {
			t.Errorf("secret %d = %s=%s, want %s", i, s.Label, s.Password, want[i])
		}
	}
	_ = cfgPw

	if runtime.GOOS != "windows" {
		script := write("pw.sh", "#!/bin/sh\necho from-script\n", 0o700)
		pw, err := readVaultPasswordFile(script)
		if err != nil || string(pw) != "from-script" {
			t.Errorf("script password = %q, %v", pw, err)
		}
	}
}
