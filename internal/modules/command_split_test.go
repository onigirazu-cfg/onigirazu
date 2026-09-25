package modules

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitCommandLine(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"uptime", []string{"uptime"}},
		{"  ls   -la  /tmp ", []string{"ls", "-la", "/tmp"}},
		{"sh -c 'id -u > /root/x'", []string{"sh", "-c", "id -u > /root/x"}},
		{`echo "a b" c`, []string{"echo", "a b", "c"}},
		{`echo "say \"hi\" \$HOME"`, []string{"echo", `say "hi" $HOME`}},
		{`echo a\ b`, []string{"echo", "a b"}},
		{`echo ''`, []string{"echo", ""}},
		{`printf '%s\n' x`, []string{"printf", `%s\n`, "x"}},
	}
	for _, tt := range tests {
		got, err := splitCommandLine(tt.in)
		require.NoError(t, err, tt.in)
		assert.Equal(t, tt.want, got, tt.in)
	}

	_, err := splitCommandLine("echo 'unterminated")
	assert.Error(t, err)
}
