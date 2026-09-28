package cli

import (
	"github.com/onigirazu-cfg/onigirazu/internal/expression"
	"github.com/onigirazu-cfg/onigirazu/internal/secrets"
)

// Every command can read secrets in templates with the environment's
// credentials; apply adds the secrets block of onigirazu.yml
func init() {
	expression.SecretLookup = secrets.NewResolver(secrets.Config{}).Get
}
