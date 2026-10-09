package cli

import (
	"fmt"
	"os"

	"github.com/onigirazu-cfg/onigirazu/internal/expression"
	"github.com/onigirazu-cfg/onigirazu/internal/secrets"
	"github.com/onigirazu-cfg/onigirazu/internal/vault"
)

// Every command can read secrets in templates with the environment's
// credentials; apply adds the secrets block of onigirazu.yml
func init() {
	expression.SecretLookup = secrets.NewResolver(secrets.Config{}).Get
	expression.SOPSFile = func(path string) (string, error) {
		data, err := os.ReadFile(path) // #nosec G304 -- the playbook names the file
		if err != nil {
			return "", err
		}
		if !vault.IsSOPS(data) {
			return "", fmt.Errorf("%s is not a SOPS-encrypted file", path)
		}
		plain, err := vault.DecryptSOPS(data)
		return string(plain), err
	}
}
