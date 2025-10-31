package modules

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScriptModuleCreation(t *testing.T) {
	m := &ScriptModule{BaseModule: NewBaseModule("script")}
	assert.NotNil(t, m)
	assert.Equal(t, "script", m.name)
}

func TestScriptModuleDescription(t *testing.T) {
	m := &ScriptModule{BaseModule: NewBaseModule("script")}
	desc := m.GetDescription()
	assert.NotEmpty(t, desc)
}
