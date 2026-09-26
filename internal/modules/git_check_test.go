package modules

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsCommitHash(t *testing.T) {
	for _, v := range []string{"a1b2c3d", "0123456789abcdef0123456789abcdef01234567"} {
		assert.True(t, isCommitHash(v), v)
	}
	for _, v := range []string{"main", "v1.63.1", "abc", "A1B2C3D", "0123456789abcdef0123456789abcdef012345678"} {
		assert.False(t, isCommitHash(v), v)
	}
}
