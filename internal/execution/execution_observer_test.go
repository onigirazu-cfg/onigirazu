package execution

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/stretchr/testify/assert"
)

// MockExecutionObserver implements ExecutionObserver for testing
type MockExecutionObserver struct {
	ExecutionStartCalls int32
	PlayStartCalls      int32
	PlayEndCalls        int32
	TaskStartCalls      int32
	TaskEndCalls        int32
	ExecutionEndCalls   int32
	ErrorCalls          int32

	LastPlaybookName string
	LastPlayName     string
	LastTaskName     string
	LastHostName     string
}

func (m *MockExecutionObserver) OnExecutionStart(playbookName string, playCount int, taskCount int) {
	atomic.AddInt32(&m.ExecutionStartCalls, 1)
	m.LastPlaybookName = playbookName
}

func (m *MockExecutionObserver) OnPlayStart(playName string, playIndex int, totalPlays int) {
	atomic.AddInt32(&m.PlayStartCalls, 1)
	m.LastPlayName = playName
}

func (m *MockExecutionObserver) OnPlayEnd(playName string, playIndex int, success bool, duration time.Duration) {
	atomic.AddInt32(&m.PlayEndCalls, 1)
}

func (m *MockExecutionObserver) OnTaskStart(taskName string, hostName string) {
	atomic.AddInt32(&m.TaskStartCalls, 1)
	m.LastTaskName = taskName
	m.LastHostName = hostName
}

func (m *MockExecutionObserver) OnTaskEnd(taskResult *types.TaskResult) {
	atomic.AddInt32(&m.TaskEndCalls, 1)
}

func (m *MockExecutionObserver) OnExecutionEnd(result *types.PlaybookResult, duration time.Duration) {
	atomic.AddInt32(&m.ExecutionEndCalls, 1)
}

func (m *MockExecutionObserver) OnError(taskName string, hostName string, error string) {
	atomic.AddInt32(&m.ErrorCalls, 1)
}

func TestMultiObserver_Attach(t *testing.T) {
	observer := NewMultiObserver()
	assert.NotNil(t, observer)

	mockObs := &MockExecutionObserver{}
	observer.Attach(mockObs)
	assert.Equal(t, 1, len(observer.observers))
}

func TestMultiObserver_AttachNil(t *testing.T) {
	observer := NewMultiObserver()
	observer.Attach(nil)
	assert.Equal(t, 0, len(observer.observers))
}

func TestMultiObserver_OnExecutionStart(t *testing.T) {
	observer := NewMultiObserver()
	mock1 := &MockExecutionObserver{}
	mock2 := &MockExecutionObserver{}

	observer.Attach(mock1)
	observer.Attach(mock2)

	observer.OnExecutionStart("test-playbook", 2, 5)

	assert.Equal(t, int32(1), mock1.ExecutionStartCalls)
	assert.Equal(t, int32(1), mock2.ExecutionStartCalls)
	assert.Equal(t, "test-playbook", mock1.LastPlaybookName)
	assert.Equal(t, "test-playbook", mock2.LastPlaybookName)
}

func TestMultiObserver_OnPlayStart(t *testing.T) {
	observer := NewMultiObserver()
	mock := &MockExecutionObserver{}
	observer.Attach(mock)

	observer.OnPlayStart("test-play", 1, 3)

	assert.Equal(t, int32(1), mock.PlayStartCalls)
	assert.Equal(t, "test-play", mock.LastPlayName)
}

func TestMultiObserver_OnPlayEnd(t *testing.T) {
	observer := NewMultiObserver()
	mock := &MockExecutionObserver{}
	observer.Attach(mock)

	observer.OnPlayEnd("test-play", 1, true, 5*time.Second)

	assert.Equal(t, int32(1), mock.PlayEndCalls)
}

func TestMultiObserver_OnTaskStart(t *testing.T) {
	observer := NewMultiObserver()
	mock := &MockExecutionObserver{}
	observer.Attach(mock)

	observer.OnTaskStart("test-task", "localhost")

	assert.Equal(t, int32(1), mock.TaskStartCalls)
	assert.Equal(t, "test-task", mock.LastTaskName)
	assert.Equal(t, "localhost", mock.LastHostName)
}

func TestMultiObserver_OnTaskEnd(t *testing.T) {
	observer := NewMultiObserver()
	mock := &MockExecutionObserver{}
	observer.Attach(mock)

	taskResult := &types.TaskResult{
		TaskName: "test-task",
	}
	observer.OnTaskEnd(taskResult)

	assert.Equal(t, int32(1), mock.TaskEndCalls)
}

func TestMultiObserver_OnExecutionEnd(t *testing.T) {
	observer := NewMultiObserver()
	mock := &MockExecutionObserver{}
	observer.Attach(mock)

	playbookResult := &types.PlaybookResult{
		Plays: []types.PlayResult{},
	}
	observer.OnExecutionEnd(playbookResult, 10*time.Second)

	assert.Equal(t, int32(1), mock.ExecutionEndCalls)
}

func TestMultiObserver_OnError(t *testing.T) {
	observer := NewMultiObserver()
	mock := &MockExecutionObserver{}
	observer.Attach(mock)

	observer.OnError("test-task", "localhost", "connection failed")

	assert.Equal(t, int32(1), mock.ErrorCalls)
}

func TestMultiObserver_MultipleObservers(t *testing.T) {
	observer := NewMultiObserver()
	mocks := make([]*MockExecutionObserver, 3)
	for i := 0; i < 3; i++ {
		mocks[i] = &MockExecutionObserver{}
		observer.Attach(mocks[i])
	}

	observer.OnExecutionStart("test", 1, 1)
	observer.OnPlayStart("play", 0, 1)
	observer.OnTaskStart("task", "host")
	observer.OnError("task", "host", "error")
	observer.OnTaskEnd(&types.TaskResult{
		TaskName: "test-task",
	})
	observer.OnPlayEnd("play", 0, true, 1*time.Second)
	observer.OnExecutionEnd(&types.PlaybookResult{}, 1*time.Second)

	// Verify all observers received all calls
	for _, mock := range mocks {
		assert.Equal(t, int32(1), mock.ExecutionStartCalls)
		assert.Equal(t, int32(1), mock.PlayStartCalls)
		assert.Equal(t, int32(1), mock.TaskStartCalls)
		assert.Equal(t, int32(1), mock.ErrorCalls)
		assert.Equal(t, int32(1), mock.TaskEndCalls)
		assert.Equal(t, int32(1), mock.PlayEndCalls)
		assert.Equal(t, int32(1), mock.ExecutionEndCalls)
	}
}

func TestMultiObserver_EmptyObservers(t *testing.T) {
	observer := NewMultiObserver()

	// Should not panic with no observers
	observer.OnExecutionStart("test", 1, 1)
	observer.OnPlayStart("play", 0, 1)
	observer.OnTaskStart("task", "host")
	observer.OnError("task", "host", "error")
	observer.OnTaskEnd(&types.TaskResult{
		TaskName: "task",
	})
	observer.OnPlayEnd("play", 0, true, 1*time.Second)
	observer.OnExecutionEnd(&types.PlaybookResult{}, 1*time.Second)
}
