package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_MissingExplicitFile(t *testing.T) {
	_, err := LoadConfig(filepath.Join(t.TempDir(), "nope.yml"))
	assert.Error(t, err)
}

func TestLoadConfig_KeyWarnings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "onigirazu.yml")
	require.NoError(t, os.WriteFile(path, []byte("max_concurrency: 5\nshow_diff: true\nvault_enabled: true\nmax_concurency: 3\n"), 0o600))
	cfg, err := LoadConfig(path)
	require.NoError(t, err)
	assert.Equal(t, 5, cfg.MaxConcurrency)
	assert.True(t, cfg.IsSet("show_diff"))
	assert.False(t, cfg.IsSet("default_timeout"))
	require.Len(t, cfg.Warnings, 2)
	assert.Contains(t, cfg.Warnings[0], `unknown key "max_concurency"`)
	assert.Contains(t, cfg.Warnings[1], `"vault_enabled" has no effect`)
}
