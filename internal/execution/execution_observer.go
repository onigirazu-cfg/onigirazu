package execution

import (
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// ExecutionObserver receives events during execution
type ExecutionObserver interface {
	// OnExecutionStart is called when execution begins
	OnExecutionStart(playbookName string, playCount int)

	// OnPlayStart is called when a play starts
	OnPlayStart(playName string, playIndex int, totalPlays int)

	// OnPlayEnd is called when a play ends
	OnPlayEnd(playName string, playIndex int, success bool, duration time.Duration)

	// OnTaskStart is called when a task starts
	OnTaskStart(taskName string, hostName string)

	// OnTaskEnd is called when a task ends
	OnTaskEnd(taskResult *types.TaskResult)

	// OnExecutionEnd is called when execution completes
	OnExecutionEnd(result *types.PlaybookResult, duration time.Duration)

	// OnError is called when an error occurs
	OnError(taskName string, hostName string, error string)
}

// MultiObserver allows multiple observers to be attached
type MultiObserver struct {
	observers []ExecutionObserver
}

// NewMultiObserver creates a new multi-observer
func NewMultiObserver() *MultiObserver {
	return &MultiObserver{
		observers: make([]ExecutionObserver, 0),
	}
}

// Attach adds an observer
func (m *MultiObserver) Attach(observer ExecutionObserver) {
	if observer != nil {
		m.observers = append(m.observers, observer)
	}
}

// OnExecutionStart broadcasts to all observers
func (m *MultiObserver) OnExecutionStart(playbookName string, playCount int) {
	for _, obs := range m.observers {
		obs.OnExecutionStart(playbookName, playCount)
	}
}

// OnPlayStart broadcasts to all observers
func (m *MultiObserver) OnPlayStart(playName string, playIndex int, totalPlays int) {
	for _, obs := range m.observers {
		obs.OnPlayStart(playName, playIndex, totalPlays)
	}
}

// OnPlayEnd broadcasts to all observers
func (m *MultiObserver) OnPlayEnd(playName string, playIndex int, success bool, duration time.Duration) {
	for _, obs := range m.observers {
		obs.OnPlayEnd(playName, playIndex, success, duration)
	}
}

// OnTaskStart broadcasts to all observers
func (m *MultiObserver) OnTaskStart(taskName string, hostName string) {
	for _, obs := range m.observers {
		obs.OnTaskStart(taskName, hostName)
	}
}

// OnTaskEnd broadcasts to all observers
func (m *MultiObserver) OnTaskEnd(taskResult *types.TaskResult) {
	for _, obs := range m.observers {
		obs.OnTaskEnd(taskResult)
	}
}

// OnExecutionEnd broadcasts to all observers
func (m *MultiObserver) OnExecutionEnd(result *types.PlaybookResult, duration time.Duration) {
	for _, obs := range m.observers {
		obs.OnExecutionEnd(result, duration)
	}
}

// OnError broadcasts to all observers
func (m *MultiObserver) OnError(taskName string, hostName string, error string) {
	for _, obs := range m.observers {
		obs.OnError(taskName, hostName, error)
	}
}
