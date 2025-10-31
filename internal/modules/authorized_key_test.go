package modules

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuthorizedKeyModuleCreation(t *testing.T) {
	m := NewAuthorizedKeyModule()
	assert.NotNil(t, m)
	assert.Equal(t, "authorized_key", m.name)
}

func TestAuthorizedKeyModuleDescription(t *testing.T) {
	m := NewAuthorizedKeyModule()
	desc := m.GetDescription()
	assert.NotEmpty(t, desc)
}
