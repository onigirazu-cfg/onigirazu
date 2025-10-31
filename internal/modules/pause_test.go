package modules

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPauseModuleCreation(t *testing.T) {
	m := &PauseModule{BaseModule: NewBaseModule("pause")}
	assert.NotNil(t, m)
	assert.Equal(t, "pause", m.name)
}

func TestPauseModuleDescription(t *testing.T) {
	m := &PauseModule{BaseModule: NewBaseModule("pause")}
	desc := m.GetDescription()
	assert.NotEmpty(t, desc)
}
