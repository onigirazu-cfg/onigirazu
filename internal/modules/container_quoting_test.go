package modules

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeContainerTool puts a fake docker/podman on PATH that fails "inspect"
// and records the words of every other call, one per line
func fakeContainerTool(t *testing.T, tool string) string {
	dir := t.TempDir()
	log := filepath.Join(dir, "calls")
	script := "#!/bin/sh\n[ \"$1\" = inspect ] && exit 1\nfor a in \"$@\"; do printf '%s\\n' \"$a\"; done > " + shellQuote(log) + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, tool), []byte(script), 0o755))
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))
	return log
}

func TestContainerRunQuoting(t *testing.T) {
	for _, tool := range []string{"docker", "podman"} {
		t.Run(tool, func(t *testing.T) {
			log := fakeContainerTool(t, tool)
			var module interface {
				Execute(context.Context, types.Host, map[string]interface{}) (types.TaskResult, error)
			} = NewDockerContainerModule()
			if tool == "podman" {
				module = NewPodmanModule()
			}
			args := map[string]interface{}{
				"name":    "web",
				"image":   "alpine:3",
				"state":   "present",
				"env":     map[string]interface{}{"MSG": "hello world; touch /tmp/pwned"},
				"command": `sh -c "echo 'a b'"`,
			}
			_, err := module.Execute(context.Background(), types.Host{Name: "localhost", Address: "127.0.0.1"}, args)
			require.NoError(t, err)
			data, err := os.ReadFile(log)
			require.NoError(t, err)
			words := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
			assert.Contains(t, words, "MSG=hello world; touch /tmp/pwned")
			assert.Equal(t, []string{"alpine:3", "sh", "-c", "echo 'a b'"}, words[len(words)-4:])
		})
	}
}

func TestContainerCheckMode(t *testing.T) {
	log := fakeContainerTool(t, "docker")
	args := map[string]interface{}{"name": "web", "image": "alpine:3", "state": "started", "_check_mode": true}
	result, err := NewDockerContainerModule().Execute(context.Background(), types.Host{Name: "localhost", Address: "127.0.0.1"}, args)
	require.NoError(t, err)
	assert.True(t, result.Changed)
	assert.Equal(t, "started", result.Output["action"])
	_, err = os.Stat(log)
	assert.True(t, os.IsNotExist(err), "check mode must not run docker")
}

func TestPlannedContainerAction(t *testing.T) {
	assert.Equal(t, "created", plannedContainerAction("present", false, false))
	assert.Equal(t, "", plannedContainerAction("present", true, false))
	assert.Equal(t, "", plannedContainerAction("started", true, true))
	assert.Equal(t, "stopped", plannedContainerAction("stopped", true, true))
	assert.Equal(t, "", plannedContainerAction("absent", false, false))
	assert.Equal(t, "removed", plannedContainerAction("absent", true, false))
}
