package expression

import "testing"

func TestSecretFunctions(t *testing.T) {
	old := SecretLookup
	defer func() { SecretLookup = old }()
	SecretLookup = func(provider, item, field string) (string, error) {
		if field == "" && provider == "vault" {
			return `{"password": "pw", "user": "app"}`, nil
		}
		return provider + "/" + item + "/" + field, nil
	}
	oldSops := SOPSFile
	defer func() { SOPSFile = oldSops }()
	SOPSFile = func(path string) (string, error) { return "secret of " + path + "\n", nil }
	cases := map[string]interface{}{
		"bitwarden('db')":                                               "bitwarden/db/password",
		"bitwarden('db', 'username')":                                   "bitwarden/db/username",
		"vault('app/db', 'password')":                                   "vault/app/db/password",
		"secret('bitwarden', 'db', 'notes')":                            "bitwarden/db/notes",
		"lookup('community.general.bitwarden', 'db', field='username')": "bitwarden/db/username",
		"lookup('bitwarden', 'db')":                                     "bitwarden/db/password",
		"query('bitwarden', 'a', 'b') | length":                         2,
		"lookup('community.hashi_vault.hashi_vault', 'secret=secret/data/app/db:password')": "vault/app/db/password",
		"lookup('community.hashi_vault.hashi_vault', secret='app/db:password')":             "vault/app/db/password",
		"lookup('hashi_vault', 'secret/data/app/db')['user']":                               "app",
		"lookup('community.hashi_vault.vault_kv2_get', 'app/db').secret.password":           "pw",
		"lookup('community.sops.sops', 'vars/secret.sops.yml')":                             "secret of vars/secret.sops.yml",
		"lookup('community.sops.sops', 'x.yml', rstrip=False)":                              "secret of x.yml\n",
	}
	for code, want := range cases {
		got, err := Eval(code, map[string]interface{}{})
		if err != nil || got != want {
			t.Errorf("%s = %v, %v; want %v", code, got, err, want)
		}
	}
}
