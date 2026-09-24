package modules

import (
	"context"
	"os"
	"os/user"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestShellQuote(t *testing.T) {
	assert.Equal(t, `'plain'`, shellQuote("plain"))
	assert.Equal(t, `'it'\''s; rm -rf /'`, shellQuote("it's; rm -rf /"))
}

func TestEnsureOwnership_Local(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file with space")
	require.NoError(t, os.WriteFile(path, []byte("x"), 0600))

	u, err := user.Current()
	require.NoError(t, err)
	g, err := user.LookupGroupId(u.Gid)
	require.NoError(t, err)

	host := types.Host{Name: "localhost", Address: "localhost"}
	ctx := context.Background()

	changed, err := ensureOwnership(ctx, host, nil, path, "", "")
	require.NoError(t, err)
	assert.False(t, changed, "nothing requested")

	changed, err = ensureOwnership(ctx, host, nil, path, u.Username, g.Name)
	require.NoError(t, err)
	assert.False(t, changed, "already owned by the current user")

	_, err = ensureOwnership(ctx, host, nil, path, "onigirazu-no-such-user", "")
	assert.Error(t, err)
}

func TestCopy_OwnerMatchingCurrentUser(t *testing.T) {
	u, err := user.Current()
	require.NoError(t, err)
	dest := filepath.Join(t.TempDir(), "copied")
	args := map[string]interface{}{"content": "x\n", "dest": dest, "owner": u.Username}
	host := types.Host{Name: "localhost", Address: "localhost"}

	result, err := NewCopyModule().Execute(context.Background(), host, args)
	require.NoError(t, err)
	assert.True(t, result.Changed)
	assert.NotContains(t, result.Output, "ownership_warning")

	result, err = NewCopyModule().Execute(context.Background(), host, args)
	require.NoError(t, err)
	assert.False(t, result.Changed)

	args["owner"] = "onigirazu-no-such-user"
	_, err = NewCopyModule().Execute(context.Background(), host, args)
	assert.Error(t, err)
}

func TestRunOnHost_QuotesArguments(t *testing.T) {
	host := types.Host{Name: "localhost", Address: "localhost"}
	out, err := runOnHost(context.Background(), host, nil, "printf", "%s|", "a b", "it's", "$HOME")
	require.NoError(t, err)
	assert.Equal(t, "a b|it's|$HOME|", out)
}
