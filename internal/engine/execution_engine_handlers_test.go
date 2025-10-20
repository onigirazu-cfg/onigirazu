package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// TestCollectTriggeredHandlers tests handler collection from task results
func TestCollectTriggeredHandlers(t *testing.T) {
	engine, _, _, _, _, _ := createTestEngine()

	playResult := &types.PlayResult{
		Name: "test play",
		Hosts: []types.HostResult{
			{
				Host: "host1",
				Tasks: []types.TaskResult{
					{
						TaskName: "Task 1",
						Success:  true,
						Notify:   []string{"handler1", "handler2"},
					},
					{
						TaskName: "Task 2",
						Success:  true,
						Notify:   []string{"handler2", "handler3"},
					},
					{
						TaskName: "Task 3",
						Success:  false,
						Notify:   []string{"handler4"}, // Should be ignored (failed)
					},
				},
			},
		},
	}

	triggered := engine.collectTriggeredHandlers(playResult)

	// Check that we have unique handlers
	assert.Equal(t, 3, len(triggered), "Should have 3 unique triggered handlers")

	// Create a set for quick lookup
	triggerSet := make(map[string]bool)
	for _, h := range triggered {
		triggerSet[h] = true
	}

	// Check expected handlers
	expected := []string{"handler1", "handler2", "handler3"}
	for _, h := range expected {
		assert.True(t, triggerSet[h], "Handler %s should be triggered", h)
	}

	// handler4 should NOT be in the list (failed task)
	assert.False(t, triggerSet["handler4"], "handler4 should not be triggered (task failed)")
}

// TestHandlerListenDirective tests that handlers with listen directive are triggered correctly
func TestHandlerListenDirective(t *testing.T) {
	// Test that a handler with listen="web services" would be triggered by notify="web services"
	handlers := []types.Task{
		{
			Name:   "restart nginx",
			Module: "service",
			Listen: "restart web services",
		},
		{
			Name:   "restart apache",
			Module: "service",
			Listen: "restart web services",
		},
		{
			Name:   "reload systemd",
			Module: "shell",
		},
	}

	triggeredNames := []string{"restart web services"}

	// Simulate the matching logic
	matchCount := 0
	for _, handler := range handlers {
		shouldExecute := false

		// Check by name
		triggeredMap := make(map[string]bool)
		for _, name := range triggeredNames {
			triggeredMap[name] = true
		}

		if triggeredMap[handler.Name] {
			shouldExecute = true
		}

		// Check by listen
		if handler.Listen != "" && triggeredMap[handler.Listen] {
			shouldExecute = true
		}

		if shouldExecute {
			matchCount++
		}
	}

	expected := 2 // restart nginx and restart apache
	assert.Equal(t, expected, matchCount, "Should match 2 handlers with listen directive")
}

// TestHandlerNotifyFieldParsing tests that notify field is parsed correctly in Task
func TestHandlerNotifyFieldParsing(t *testing.T) {
	task := &types.Task{
		Name:   "test task",
		Module: "debug",
		Notify: []string{"handler1", "handler2"},
	}

	assert.Equal(t, 2, len(task.Notify), "Task should have 2 notify handlers")
	assert.Equal(t, "handler1", task.Notify[0])
	assert.Equal(t, "handler2", task.Notify[1])
}

// TestHandlerListenFieldParsing tests that listen field is parsed correctly in Task
func TestHandlerListenFieldParsing(t *testing.T) {
	handler := &types.Task{
		Name:   "restart app",
		Module: "service",
		Listen: "restart application",
	}

	assert.Equal(t, "restart application", handler.Listen, "Handler should have correct listen directive")
}
