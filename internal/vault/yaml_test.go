package vault

import (
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLoadInlineAndWholeFile(t *testing.T) {
	SetSecrets([]Secret{testSecret})
	defer SetSecrets(nil)

	inline, _ := os.ReadFile("testdata/inline.yml") // written by ansible-vault encrypt_string --name db_pass
	doc := "plain: 1\n" + string(inline) + "nested:\n  list: [a]\n"
	out, err := Load([]byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	var vars map[string]interface{}
	if err := yaml.Unmarshal(out, &vars); err != nil {
		t.Fatal(err)
	}
	if vars["db_pass"] != "inline-secret" || vars["plain"] != 1 {
		t.Errorf("vars = %v", vars)
	}

	whole, _ := os.ReadFile("testdata/vars.yml.vault")
	out, err = Load(whole)
	if err != nil || !strings.Contains(string(out), "db_password: s3cret") {
		t.Errorf("whole file = %q, %v", out, err)
	}

	plain := []byte("a: 1\n# !vault in a comment only\n")
	if out, _ := Load(plain); string(out) != string(plain) {
		t.Errorf("plain YAML must pass unchanged: %q", out)
	}

	SetSecrets(nil)
	if _, err := Load(whole); err != ErrNoPassword {
		t.Errorf("without a password: %v", err)
	}
}

func TestLazySource(t *testing.T) {
	calls := 0
	SetSource(func() ([]Secret, error) { calls++; return []Secret{testSecret}, nil })
	defer SetSecrets(nil)
	if _, err := Load([]byte("a: 1\n")); err != nil || calls != 0 {
		t.Errorf("no vault data, no password asked: calls=%d err=%v", calls, err)
	}
	whole, _ := os.ReadFile("testdata/vars.yml.vault")
	for i := 0; i < 2; i++ {
		if _, err := Load(whole); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 1 {
		t.Errorf("the source is asked once, got %d", calls)
	}
}
