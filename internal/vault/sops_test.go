package vault

import (
	"os/exec"
	"strings"
	"testing"
)

const sopsDoc = `db:
    password: ENC[AES256_GCM,data:abc,iv:def,tag:ghi,type:str]
sops:
    age:
        - recipient: age1example
          enc: |
            -----BEGIN AGE ENCRYPTED FILE-----
            -----END AGE ENCRYPTED FILE-----
    mac: ENC[AES256_GCM,data:mac,iv:x,tag:y,type:str]
    version: 3.9.0
`

func TestIsSOPS(t *testing.T) {
	if !IsSOPS([]byte(sopsDoc)) {
		t.Error("a document with sops metadata is SOPS")
	}
	if IsSOPS([]byte("sops: true\nmac: 1\n")) || IsSOPS([]byte("db:\n  password: x\n")) {
		t.Error("plain documents are not SOPS")
	}
	if !IsSOPS([]byte(`{"db": {"password": "ENC[AES256_GCM,data:a,iv:b,tag:c,type:str]"}, "sops": {"mac": "ENC[x]", "version": "3.9.0"}}`)) {
		t.Error("JSON documents are SOPS too")
	}
}

func TestLoadOpensSOPS(t *testing.T) {
	calls := 0
	old := sopsCommand
	defer func() { sopsCommand = old }()
	sopsCommand = func(args ...string) *exec.Cmd {
		calls++
		if strings.Join(args, " ") != "--decrypt --input-type yaml --output-type yaml /dev/stdin" {
			t.Errorf("sops arguments: %v", args)
		}
		return exec.Command("sh", "-c", "cat >/dev/null; printf 'db:\\n    password: plain\\n'")
	}
	for i := 0; i < 2; i++ {
		out, err := Load([]byte(sopsDoc))
		if err != nil || string(out) != "db:\n    password: plain\n" {
			t.Fatalf("Load = %q, %v", out, err)
		}
	}
	if calls != 1 {
		t.Errorf("a document is decrypted once, got %d calls", calls)
	}
	sopsCommand = func(args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "cat >/dev/null; echo 'Failed to get the data key required to decrypt the SOPS file.' >&2; exit 128")
	}
	_, err := Load([]byte(strings.Replace(sopsDoc, "abc", "other", 1)))
	if err == nil || !strings.Contains(err.Error(), "sops: Failed to get the data key") {
		t.Errorf("sops errors are reported: %v", err)
	}
}
