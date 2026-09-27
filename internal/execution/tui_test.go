package execution

import (
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

var ansi = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// fakeControl records what the dashboard asked of the run
type fakeControl struct {
	mu              sync.Mutex
	paused, stopped bool
}

func (c *fakeControl) Pause()  { c.mu.Lock(); c.paused = true; c.mu.Unlock() }
func (c *fakeControl) Resume() { c.mu.Lock(); c.paused = false; c.mu.Unlock() }
func (c *fakeControl) Stop()   { c.mu.Lock(); c.stopped = true; c.mu.Unlock() }
func (c *fakeControl) Paused() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.paused
}

func newTestModel() (*EnhancedTUIModel, *fakeControl) {
	m := NewEnhancedTUIModel()
	m.width, m.height = 130, 40
	c := &fakeControl{}
	m.SetRunController(c)
	m.processEvent(ExecutionEvent{Type: "execution_start", PlayName: "site.yml", TotalTaskCount: 3})
	return m, c
}

func key(m *EnhancedTUIModel, k string) tea.Cmd {
	var msg tea.KeyMsg
	switch k {
	case "enter":
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		msg = tea.KeyMsg{Type: tea.KeyEscape}
	case "up":
		msg = tea.KeyMsg{Type: tea.KeyUp}
	case "end":
		msg = tea.KeyMsg{Type: tea.KeyEnd}
	case "ctrl+c":
		msg = tea.KeyMsg{Type: tea.KeyCtrlC}
	default:
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
	}
	_, cmd := m.Update(msg)
	return cmd
}

func screen(m *EnhancedTUIModel) string { return ansi.ReplaceAllString(m.View(), "") }

func taskEnd(m *EnhancedTUIModel, r types.TaskResult) {
	m.processEvent(ExecutionEvent{Type: "task_end", TaskName: r.TaskName, HostName: r.Host,
		TaskFailed: r.Failed, TaskChanged: r.Changed, Result: &r})
}

func TestTUIDetailModes(t *testing.T) {
	m, _ := newTestModel()
	taskEnd(m, types.TaskResult{TaskName: "install", Host: "web1", Changed: true,
		Output: map[string]interface{}{"stdout": "line one\nline two"}})
	taskEnd(m, types.TaskResult{TaskName: "migrate", Host: "db1", Failed: true, Error: "exit status 2"})
	m.AddLog("2026-09-27 10:00:00 [\x1b[32mINFO\x1b[0m] connected to web1 {host=web1}")
	m.AddLog("2026-09-27 10:00:01 [WARN] slow host {host=db1}")

	s := screen(m)
	assert.Contains(t, s, "web1: install [changed]")
	assert.Contains(t, s, "db1: migrate [failed]")
	assert.Contains(t, s, "exit status 2")
	assert.Contains(t, s, "slow host")
	assert.NotContains(t, s, "connected to web1") // info lines are VERBOSE
	assert.NotContains(t, s, "stdout: line one")

	key(m, "v")
	s = screen(m)
	assert.Contains(t, s, "connected to web1")
	assert.NotContains(t, s, "{host=web1}")
	assert.Contains(t, s, "stdout: line one")
}

func TestTUIScroll(t *testing.T) {
	m, _ := newTestModel()
	for i := 0; i < 100; i++ {
		m.processEvent(ExecutionEvent{Type: "error", Message: "line-" + strings.Repeat("x", 1) + string(rune('A'+i%26)) + "-" + time.Duration(i).String()})
	}
	assert.Contains(t, screen(m), "-99ns")
	key(m, "up")
	key(m, "up")
	s := screen(m)
	assert.NotContains(t, s, "-99ns")
	assert.Contains(t, s, "2 lines back")
	key(m, "end")
	assert.Contains(t, screen(m), "-99ns")
}

func TestTUIPauseStopQuit(t *testing.T) {
	m, c := newTestModel()
	key(m, "p")
	assert.True(t, c.Paused())
	assert.Contains(t, screen(m), "PAUSED")
	key(m, "p")
	assert.False(t, c.Paused())

	// G asks, N keeps running
	key(m, "g")
	assert.Contains(t, screen(m), "Stop")
	key(m, "n")
	assert.False(t, c.stopped)

	// Q during the run asks too; Y stops the run and closes the dashboard
	key(m, "q")
	assert.False(t, m.shouldExit)
	key(m, "y")
	assert.True(t, c.stopped)
	assert.True(t, m.shouldExit)

	// events after close are dropped instead of blocking the run
	done := make(chan struct{})
	go func() {
		for i := 0; i < 500; i++ {
			m.OnTaskStart("t", "h")
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("sending events to a closed dashboard blocked")
	}
}

func TestTUIQuitAfterRun(t *testing.T) {
	m, c := newTestModel()
	m.processEvent(ExecutionEvent{Type: "execution_end", Success: true, Message: "done"})
	assert.Contains(t, screen(m), "COMPLETED")
	key(m, "q") // no question once the run is over
	assert.False(t, c.stopped)
	assert.True(t, m.shouldExit)
}

func TestTUIStoppedStatus(t *testing.T) {
	m, c := newTestModel()
	key(m, "g")
	key(m, "y")
	assert.True(t, c.stopped)
	m.processEvent(ExecutionEvent{Type: "execution_end", Success: false, Message: "failed"})
	assert.Contains(t, screen(m), "STOPPED")
}

func TestTUIResultsBrowser(t *testing.T) {
	m, _ := newTestModel()
	taskEnd(m, types.TaskResult{TaskName: "install", Host: "web1", Changed: true, Module: "apt"})
	taskEnd(m, types.TaskResult{TaskName: "migrate", Host: "db1", Failed: true, Error: "exit status 2",
		Output: map[string]interface{}{"stderr": "relation missing"}})
	key(m, "r")
	s := screen(m)
	assert.Contains(t, s, "TASK RESULTS (2)")
	assert.Contains(t, s, "install")
	key(m, "enter") // cursor is on the newest: migrate
	s = screen(m)
	assert.Contains(t, s, "Host:     db1")
	assert.Contains(t, s, "exit status 2")
	assert.Contains(t, s, "relation missing")
	key(m, "esc")
	key(m, "up")
	key(m, "enter")
	assert.Contains(t, screen(m), "Module:   apt")
	key(m, "esc")
	key(m, "f")
	assert.Contains(t, screen(m), "TASK RESULTS (1) - failed only")
	key(m, "esc")
	assert.Contains(t, screen(m), "Statistics")
}

func TestTUIHostsPanel(t *testing.T) {
	m, _ := newTestModel()
	m.OnTaskEnd(&types.TaskResult{TaskName: "a", Host: "web1", Changed: true})
	m.OnTaskEnd(&types.TaskResult{TaskName: "a", Host: "db1", Failed: true})
	for len(m.eventChan) > 0 {
		m.processEvent(<-m.eventChan)
	}
	s := screen(m)
	assert.Contains(t, s, "Hosts: 2")
	assert.Regexp(t, `db1\s+0/0/1`, s)
	assert.Regexp(t, `web1\s+0/1/0`, s)
}

func TestParseLogLine(t *testing.T) {
	level, msg := parseLogLine("2026-09-27 11:29:21 [\x1b[33mWARN\x1b[0m] Host(s) u2404 failed {play=p}")
	assert.Equal(t, "WARN", level)
	assert.Equal(t, "Host(s) u2404 failed", msg)
	level, msg = parseLogLine("plain text")
	assert.Equal(t, "INFO", level)
	assert.Equal(t, "plain text", msg)
}

func TestTUIHelpIsVisible(t *testing.T) {
	m, _ := newTestModel()
	key(m, "h")
	s := screen(m)
	assert.Contains(t, s, "KEYBOARD SHORTCUTS")
	assert.Contains(t, s, "G - Stop gracefully")
	key(m, "esc")
	assert.NotContains(t, screen(m), "KEYBOARD SHORTCUTS")
}

func TestTUIDetailDiffAndScroll(t *testing.T) {
	m, _ := newTestModel()
	m.height = 20
	var before, after strings.Builder
	for i := 0; i < 40; i++ {
		before.WriteString("line " + strings.Repeat("x", i%3) + "\n")
		after.WriteString("line " + strings.Repeat("x", i%3) + "\n")
	}
	after.WriteString("port=8080\n")
	taskEnd(m, types.TaskResult{TaskName: "config", Host: "web1", Changed: true, Module: "template",
		Output: map[string]interface{}{"stdout": before.String(), "diff": []interface{}{map[string]interface{}{
			"before_header": "/etc/app.conf", "before": before.String(), "after": after.String()}}}})
	key(m, "r")
	key(m, "enter")
	s := screen(m)
	assert.Contains(t, s, "Module:   template")
	assert.Contains(t, s, "lines 1-")
	assert.NotContains(t, s, "+port=8080")
	key(m, "end")
	s = screen(m)
	assert.Contains(t, s, "+port=8080")
	assert.NotContains(t, s, "Module:   template")
	key(m, "home")
	assert.Contains(t, screen(m), "Module:   template")
	key(m, "esc")
	assert.Contains(t, screen(m), "TASK RESULTS")
}
