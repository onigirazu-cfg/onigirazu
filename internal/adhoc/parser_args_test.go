package adhoc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseModuleArgs(t *testing.T) {
	tests := []struct {
		module string
		args   []string
		want   map[string]interface{}
	}{
		{"command", []string{"id -un"}, map[string]interface{}{"cmd": "id -un"}},
		{"command", []string{"echo a=b"}, map[string]interface{}{"cmd": "echo a=b"}},
		{"shell", []string{"pwd chdir=/tmp"}, map[string]interface{}{"cmd": "pwd", "chdir": "/tmp"}},
		{"shell", []string{"cmd=df -h | grep /dev"}, map[string]interface{}{"cmd": "df -h | grep /dev"}},
		{"file", []string{"path=/tmp/x state=touch"}, map[string]interface{}{"path": "/tmp/x", "state": "touch"}},
		{"file", []string{"path=/tmp/x", "state=touch"}, map[string]interface{}{"path": "/tmp/x", "state": "touch"}},
		{"debug", []string{"msg=hello world"}, map[string]interface{}{"msg": "hello world"}},
	}
	for _, tt := range tests {
		cmd, err := NewParser().Parse("", tt.module, tt.args)
		require.NoError(t, err, tt.args)
		assert.Equal(t, tt.want, cmd.Args, tt.args)
	}
}
