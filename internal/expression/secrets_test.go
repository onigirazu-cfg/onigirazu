package expression

import "testing"

func TestSecretFunctions(t *testing.T) {
	old := SecretLookup
	defer func() { SecretLookup = old }()
	SecretLookup = func(provider, item, field string) (string, error) {
		return provider + "/" + item + "/" + field, nil
	}
	cases := map[string]interface{}{
		"bitwarden('db')":                                               "bitwarden/db/password",
		"bitwarden('db', 'username')":                                   "bitwarden/db/username",
		"vault('app/db', 'password')":                                   "vault/app/db/password",
		"secret('bitwarden', 'db', 'notes')":                            "bitwarden/db/notes",
		"lookup('community.general.bitwarden', 'db', field='username')": "bitwarden/db/username",
		"lookup('bitwarden', 'db')":                                     "bitwarden/db/password",
		"query('bitwarden', 'a', 'b') | length":                         2,
	}
	for code, want := range cases {
		got, err := Eval(code, map[string]interface{}{})
		if err != nil || got != want {
			t.Errorf("%s = %v, %v; want %v", code, got, err, want)
		}
	}
}
