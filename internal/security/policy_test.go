package security

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func writePolicy(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "security-policy.json")
	require.NoError(t, os.WriteFile(path, []byte(body), 0600))
	return path
}

func TestDefaultPolicyAllowsEverything(t *testing.T) {
	v := NewSecurityValidator(DefaultSecurityConfig())
	tasks := []types.Task{
		{Module: "apt", Args: map[string]interface{}{"name": "nginx"}},
		{Module: "template", Args: map[string]interface{}{"dest": "/etc/nginx/nginx.conf"}},
		{Module: "shell", Args: map[string]interface{}{"cmd": "cd /srv && ls | wc -l; rm -rf /var/cache/app"}},
		{Module: "user", Args: map[string]interface{}{"name": "root"}},
		{Module: "script", Retries: 10, Timeout: 2 * time.Hour},
	}
	for _, task := range tasks {
		result := v.ValidateTask(task)
		assert.True(t, result.Valid, "%s: %v", task.Module, result.Error())
	}
}

func TestLoadPolicyFile(t *testing.T) {
	path := writePolicy(t, `{
  "_comment": "ignored",
  "allowed_modules": ["copy"],
  "allowed_directories": ["/srv"],
  "max_timeout": "10m",
  "strict": true
}`)
	cfg, err := LoadPolicyFile(path)
	require.NoError(t, err)
	assert.Equal(t, []string{"copy"}, cfg.AllowedModules)
	assert.Equal(t, 10*time.Minute, cfg.MaxTimeout)
	assert.True(t, cfg.Strict)
	assert.Empty(t, cfg.BlockedCommands, "unset keys keep permissive defaults")

	v := NewSecurityValidator(cfg)
	assert.False(t, v.ValidateTask(types.Task{Module: "apt"}).Valid)
	assert.False(t, v.ValidateTask(types.Task{Module: "copy", Args: map[string]interface{}{"dest": "/etc/x"}}).Valid)
	assert.True(t, v.ValidateTask(types.Task{Module: "copy", Args: map[string]interface{}{"dest": "/srv/x"}}).Valid)
}

func TestLoadPolicyFile_Errors(t *testing.T) {
	_, err := LoadPolicyFile(writePolicy(t, `{"allowed_modules": ["copy"],}`))
	assert.Error(t, err, "invalid JSON")

	_, err = LoadPolicyFile(writePolicy(t, `{"allowed_module": ["copy"]}`))
	assert.ErrorContains(t, err, "allowed_module")

	_, err = LoadPolicyFile(writePolicy(t, `{"max_timeout": "soon"}`))
	assert.Error(t, err)
}

func TestLoadPolicy_Resolution(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("HOME", dir)
	t.Setenv(PolicyEnvVar, "")

	cfg, source, err := LoadPolicy("")
	require.NoError(t, err)
	assert.Empty(t, source)
	assert.Empty(t, cfg.AllowedModules)

	require.NoError(t, os.WriteFile("security-policy.json", []byte(`{"allowed_modules": ["file"]}`), 0600))
	cfg, source, err = LoadPolicy("")
	require.NoError(t, err)
	assert.Equal(t, "security-policy.json", source)
	assert.Equal(t, []string{"file"}, cfg.AllowedModules)

	envPath := writePolicy(t, `{"allowed_modules": ["copy"]}`)
	t.Setenv(PolicyEnvVar, envPath)
	cfg, source, err = LoadPolicy("")
	require.NoError(t, err)
	assert.Equal(t, envPath, source)
	assert.Equal(t, []string{"copy"}, cfg.AllowedModules)

	_, _, err = LoadPolicy(filepath.Join(dir, "missing.json"))
	assert.Error(t, err, "an explicit path must exist")
}

func TestPolicy_HostsPortsAndFileTypes(t *testing.T) {
	v := NewSecurityValidator(SecurityConfig{AllowedHosts: []string{"10.0.0.*"}, AllowedPorts: []int{22}, AllowedFileTypes: []string{".conf"}})
	assert.NoError(t, v.ValidateHostAccess(types.Host{Name: "web", Address: "10.0.0.5", Port: 22}))
	assert.Error(t, v.ValidateHostAccess(types.Host{Name: "db", Address: "192.168.1.5"}))
	assert.Error(t, v.ValidateHostAccess(types.Host{Name: "web", Address: "10.0.0.5", Port: 2222}))

	assert.True(t, v.ValidateTask(types.Task{Module: "template", Args: map[string]interface{}{"dest": "/etc/app.conf"}}).Valid)
	assert.False(t, v.ValidateTask(types.Task{Module: "template", Args: map[string]interface{}{"dest": "/etc/app.sh"}}).Valid)
}

func TestLoadPolicyFile_RejectsUnsupportedKeys(t *testing.T) {
	_, err := LoadPolicyFile(writePolicy(t, `{"audit_enabled": true}`))
	assert.ErrorContains(t, err, "audit_enabled")
}
