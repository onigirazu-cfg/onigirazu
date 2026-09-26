package modules

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetSysctlLine(t *testing.T) {
	out, changed := setSysctlLine("", "vm.swappiness", "10")
	assert.True(t, changed)
	assert.Equal(t, "vm.swappiness = 10\n", out)

	out, changed = setSysctlLine("# tuned\nvm.swappiness=10\n", "vm.swappiness", "10")
	assert.False(t, changed, "same value in another spacing")
	assert.Equal(t, "# tuned\nvm.swappiness=10\n", out)

	out, changed = setSysctlLine("vm.swappiness = 60\nnet.ipv4.ip_forward = 1\nvm.swappiness = 30\n", "vm.swappiness", "10")
	assert.True(t, changed)
	assert.Equal(t, "vm.swappiness = 10\nnet.ipv4.ip_forward = 1\n", out, "updated in place, duplicate dropped")

	_, changed = setSysctlLine("net.ipv4.ip_local_port_range = 1024\t65000\n", "net.ipv4.ip_local_port_range", normalizeSysctl("1024 65000"))
	assert.False(t, changed, "multi-value parameters compare by fields")

	out, changed = setSysctlLine("# vm.swappiness = 1\n", "vm.swappiness", "10")
	assert.True(t, changed)
	assert.Equal(t, "# vm.swappiness = 1\nvm.swappiness = 10\n", out, "comments are not settings")
}

func TestRemoveSysctlLine(t *testing.T) {
	out, changed := removeSysctlLine("a = 1\nvm.swappiness = 10\n", "vm.swappiness")
	assert.True(t, changed)
	assert.Equal(t, "a = 1\n", out)

	_, changed = removeSysctlLine("a = 1\n", "vm.swappiness")
	assert.False(t, changed)
}
