package secrets

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type fakeProvider struct{ name string }

func (f *fakeProvider) GetSecret(_ context.Context, item, field string) (string, error) {
	return f.name + ":" + item + ":" + field, nil
}
func (f *fakeProvider) ListSecrets(context.Context, string) ([]string, error) { return nil, nil }
func (f *fakeProvider) Close() error                                          { return nil }
func (f *fakeProvider) Name() string                                          { return f.name }

func TestResolverCredentialsFromEnvironment(t *testing.T) {
	var seen []ProviderConfig
	r := NewResolver(Config{Vault: VaultConfig{Address: "https://vault.example:8200", Mount: "kv"}})
	r.newProvider = func(c ProviderConfig) (SecretProvider, error) {
		seen = append(seen, c)
		return &fakeProvider{name: c.Type}, nil
	}

	t.Setenv("BW_SESSION", "")
	if _, err := r.Get("bitwarden", "db", "password"); err == nil || !strings.Contains(err.Error(), "bw unlock") {
		t.Errorf("a locked vault must say how to unlock it, got %v", err)
	}
	t.Setenv("BW_SESSION", "sess")
	v, err := r.Get("bitwarden", "db", "password")
	if err != nil || v != "bitwarden:db:password" {
		t.Fatalf("Get = %q, %v", v, err)
	}
	if _, err := r.Get("bitwarden", "db", "username"); err != nil {
		t.Fatal(err)
	}
	if len(seen) != 1 || seen[0].Config["session_token"] != "sess" {
		t.Errorf("the provider is created once, with the session: %+v", seen)
	}

	t.Setenv("VAULT_TOKEN", "tok")
	if _, err := r.Get("vault", "app/db", "password"); err != nil {
		t.Fatal(err)
	}
	vc := seen[len(seen)-1].Config
	if vc["address"] != "https://vault.example:8200" || vc["token"] != "tok" || vc["mount"] != "kv" {
		t.Errorf("vault config = %v", vc)
	}
	if _, err := r.Get("keepass", "x", "y"); err == nil {
		t.Error("an unknown provider must fail")
	}
}

// the bw CLI gets the session in its environment, never on the command
// line, and a session the user unlocked is not locked afterwards
func TestBitwardenSessionNotOnCommandLine(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script stand-in for bw")
	}
	dir := t.TempDir()
	log := filepath.Join(dir, "calls")
	script := "#!/bin/sh\necho \"$* session=$BW_SESSION\" >> " + log + "\n" +
		"if [ \"$1\" = get ]; then echo '{\"id\":\"1\",\"name\":\"db\",\"login\":{\"username\":\"app\",\"password\":\"s3cret\"},\"fields\":[{\"name\":\"port\",\"value\":\"5432\"}]}'; fi\n"
	if err := os.WriteFile(filepath.Join(dir, "bw"), []byte(script), 0o755); err != nil { // #nosec G306 -- test stand-in must be executable
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	c, err := NewBitwardenClient(map[string]interface{}{"session_token": "sess-123"})
	if err != nil {
		t.Fatal(err)
	}
	for field, want := range map[string]string{"password": "s3cret", "username": "app", "port": "5432"} {
		got, err := c.GetSecret(context.Background(), "db", field)
		if err != nil || got != want {
			t.Errorf("%s = %q, %v", field, got, err)
		}
	}
	_ = c.Close()
	data, _ := os.ReadFile(log) // #nosec G304 -- test file
	calls := string(data)
	if strings.Contains(calls, "--session") || !strings.Contains(calls, "get item db session=sess-123") {
		t.Errorf("bw calls: %q", calls)
	}
	if strings.Contains(calls, "lock") {
		t.Errorf("an unlocked vault of the user must stay unlocked: %q", calls)
	}
}

func TestResolverVaultAppRole(t *testing.T) {
	var seen []ProviderConfig
	r := NewResolver(Config{Vault: VaultConfig{Address: "https://vault.example:8200"}})
	r.newProvider = func(c ProviderConfig) (SecretProvider, error) {
		seen = append(seen, c)
		return &fakeProvider{name: c.Type}, nil
	}
	t.Setenv("HOME", t.TempDir())
	t.Setenv("VAULT_TOKEN", "")
	t.Setenv("VAULT_ROLE_ID", "")
	t.Setenv("VAULT_SECRET_ID", "")
	if _, err := r.Get("vault", "app/db", "password"); err == nil || !strings.Contains(err.Error(), "VAULT_ROLE_ID") {
		t.Errorf("without a token the error names AppRole too: %v", err)
	}
	t.Setenv("VAULT_ROLE_ID", "role")
	t.Setenv("VAULT_SECRET_ID", "sid")
	if _, err := r.Get("vault", "app/db", "password"); err != nil {
		t.Fatal(err)
	}
	if len(seen) != 1 || seen[0].Config["role_id"] != "role" || seen[0].Config["secret_id"] != "sid" || seen[0].Config["auth_mount"] != "approle" || seen[0].Config["token"] != "" {
		t.Errorf("AppRole credentials reach the provider: %+v", seen)
	}
}
