package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestTaskResultCreation tests task result initialization
func TestTaskResultCreation(t *testing.T) {
	result := TaskResult{
		TaskName: "test",
		Host:     "localhost",
		Module:   "debug",
		Success:  true,
	}

	assert.Equal(t, "test", result.TaskName)
	assert.Equal(t, "localhost", result.Host)
	assert.Equal(t, "debug", result.Module)
	assert.True(t, result.Success)
}

// TestHostConfiguration tests Host struct
func TestHostConfiguration(t *testing.T) {
	host := Host{
		Name:     "web-server",
		Hostname: "web-server.local",
		IP:       "192.168.1.1",
		Port:     22,
		User:     "ansible",
	}

	assert.Equal(t, "web-server", host.Name)
	assert.Equal(t, "web-server.local", host.Hostname)
	assert.Equal(t, "192.168.1.1", host.IP)
	assert.Equal(t, 22, host.Port)
	assert.Equal(t, "ansible", host.User)
}

// TestPlaybookStructure tests Playbook configuration
func TestPlaybookStructure(t *testing.T) {
	playbook := Playbook{
		Name:        "Deploy",
		Description: "Deployment",
	}

	assert.Equal(t, "Deploy", playbook.Name)
	assert.Equal(t, "Deployment", playbook.Description)
}

// TestPlayStructure tests Play configuration
func TestPlayStructure(t *testing.T) {
	play := Play{
		Name:  "webservers",
		Hosts: []string{"web1", "web2"},
	}

	assert.Equal(t, "webservers", play.Name)
	assert.Len(t, play.Hosts, 2)
}

// TestTaskStructure tests Task configuration
func TestTaskStructure(t *testing.T) {
	task := Task{
		Name:   "install",
		Module: "apt",
		Tags:   []string{"deploy"},
	}

	assert.Equal(t, "install", task.Name)
	assert.Equal(t, "apt", task.Module)
	assert.Len(t, task.Tags, 1)
}

// TestResultWithTimestamp tests result with timing
func TestResultWithTimestamp(t *testing.T) {
	now := time.Now()
	result := TaskResult{
		TaskName:  "test",
		Timestamp: now,
		Duration:  100 * time.Millisecond,
	}

	assert.Equal(t, now, result.Timestamp)
	assert.Equal(t, 100*time.Millisecond, result.Duration)
}

// TestResultWithError tests result error handling
func TestResultWithError(t *testing.T) {
	result := TaskResult{
		TaskName: "test",
		Success:  false,
		Error:    "connection failed",
	}

	assert.False(t, result.Success)
	assert.Equal(t, "connection failed", result.Error)
}

// BenchmarkTaskResultCreation benchmarks task result
func BenchmarkTaskResultCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = TaskResult{
			TaskName: "test",
			Host:     "localhost",
			Module:   "debug",
			Success:  true,
		}
	}
}

// BenchmarkHostCreation benchmarks host creation
func BenchmarkHostCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Host{
			Name:     "server",
			Hostname: "server.local",
			IP:       "192.168.1.1",
			Port:     22,
			User:     "ansible",
		}
	}
}

// BenchmarkTaskCreation benchmarks task creation
func BenchmarkTaskCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Task{
			Name:   "task",
			Module: "debug",
			Tags:   []string{},
		}
	}
}
