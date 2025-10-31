package modules

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWaitForModuleCreation(t *testing.T) {
	m := &WaitForModule{BaseModule: NewBaseModule("wait_for")}
	assert.NotNil(t, m)
	assert.Equal(t, "wait_for", m.name)
}

func TestWaitForModuleDescription(t *testing.T) {
	m := &WaitForModule{BaseModule: NewBaseModule("wait_for")}
	desc := m.GetDescription()
	assert.NotEmpty(t, desc)
}
