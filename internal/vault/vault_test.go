package vault

import (
	"os"
	"strings"
	"testing"
)

// testdata was written by ansible-vault (ansible-core 2.21) with the
// password "test-pass"
var testSecret = Secret{Password: []byte("test-pass")}

func TestDecryptAnsibleFile(t *testing.T) {
	data, err := os.ReadFile("testdata/vars.yml.vault")
	if err != nil {
		t.Fatal(err)
	}
	if !IsEncrypted(data) {
		t.Fatal("not recognized as vault data")
	}
	plain, err := Decrypt(data, []Secret{{Password: []byte("wrong")}, testSecret})
	if err != nil {
		t.Fatal(err)
	}
	if string(plain) != "db_password: s3cret\napi:\n  key: abc\n" {
		t.Errorf("plain = %q", plain)
	}
	if _, err := Decrypt(data, []Secret{{Password: []byte("wrong")}}); err != ErrNoPassword {
		t.Errorf("a wrong password must fail with ErrNoPassword, got %v", err)
	}
}

func TestDecryptLabelled(t *testing.T) {
	data, _ := os.ReadFile("testdata/labelled.vault")
	if !strings.HasPrefix(string(data), "$ANSIBLE_VAULT;1.2;AES256;prod") {
		t.Fatalf("fixture: %q", data[:40])
	}
	plain, err := Decrypt(data, []Secret{{Label: "dev", Password: []byte("x")}, {Label: "prod", Password: []byte("test-pass")}})
	if err != nil || string(plain) != "label"+"led\n" {
		t.Errorf("plain = %q, %v", plain, err)
	}
}

func TestRoundTrip(t *testing.T) {
	for _, s := range []Secret{testSecret, {Label: "prod", Password: []byte("p")}} {
		enc, err := Encrypt([]byte("hello\nworld"), s)
		if err != nil {
			t.Fatal(err)
		}
		plain, err := Decrypt(enc, []Secret{s})
		if err != nil || string(plain) != "hello\nworld" {
			t.Errorf("round trip = %q, %v", plain, err)
		}
	}
}
