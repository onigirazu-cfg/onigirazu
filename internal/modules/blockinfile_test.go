package modules

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBlockInFileModuleCreation(t *testing.T) {
	m := &BlockInFileModule{BaseModule: NewBaseModule("blockinfile")}
	assert.NotNil(t, m)
	assert.Equal(t, "blockinfile", m.name)
}

func TestBlockInFileModuleDescription(t *testing.T) {
	m := &BlockInFileModule{BaseModule: NewBaseModule("blockinfile")}
	desc := m.GetDescription()
	assert.NotEmpty(t, desc)
}
