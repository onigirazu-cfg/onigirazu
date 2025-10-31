package modules

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestURIModuleCreation(t *testing.T) {
	m := &URIModule{BaseModule: NewBaseModule("uri")}
	assert.NotNil(t, m)
	assert.Equal(t, "uri", m.name)
}

func TestURIModuleDescription(t *testing.T) {
	m := &URIModule{BaseModule: NewBaseModule("uri")}
	desc := m.GetDescription()
	assert.NotEmpty(t, desc)
}
