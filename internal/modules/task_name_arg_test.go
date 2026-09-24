package modules

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// TestExecute_NonStringName verifies modules do not panic when "name" is a list,
// e.g. `apt: {name: [curl, git]}`.
func TestExecute_NonStringName(t *testing.T) {
	args := map[string]interface{}{
		"name":  []interface{}{"curl", "git"},
		"state": "unsupported", // keeps the module from running package commands
	}
	assert.NotPanics(t, func() {
		_, _ = NewAptModule().Execute(context.Background(), types.Host{Name: "localhost"}, args)
	})
}
