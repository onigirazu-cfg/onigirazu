package modules

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFailModuleCreation(t *testing.T) {
	m := &FailModule{BaseModule: NewBaseModule("fail")}
	assert.NotNil(t, m)
	assert.Equal(t, "fail", m.name)
}

func TestFailModuleDescription(t *testing.T) {
	m := &FailModule{BaseModule: NewBaseModule("fail")}
	desc := m.GetDescription()
	assert.NotEmpty(t, desc)
}
