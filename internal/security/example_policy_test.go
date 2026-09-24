package security

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExamplePolicyLoads(t *testing.T) {
	_, err := LoadPolicyFile("../../examples/security-policy.json")
	require.NoError(t, err)
}
