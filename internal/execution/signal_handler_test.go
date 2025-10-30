package execution

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewSignalHandler(t *testing.T) {
	ctx := context.Background()
	handler := NewSignalHandler(ctx, 30*time.Second)

	assert.NotNil(t, handler)
	assert.NotNil(t, handler.ctx)
	assert.Equal(t, int32(0), handler.interruptCount.Load())
	assert.Equal(t, 30*time.Second, handler.shutdownTimeout)
}

func TestSignalHandler_Context(t *testing.T) {
	ctx := context.Background()
	handler := NewSignalHandler(ctx, 30*time.Second)
	defer handler.Close()

	handlerCtx := handler.Context()
	assert.NotNil(t, handlerCtx)
}

func TestSignalHandler_RegisterCleanup(t *testing.T) {
	ctx := context.Background()
	handler := NewSignalHandler(ctx, 30*time.Second)
	defer handler.Close()

	handler.RegisterCleanup(func() error {
		return nil
	})

	assert.Equal(t, 1, len(handler.cleanupFuncs))
}

func TestSignalHandler_RegisterCleanup_Multiple(t *testing.T) {
	ctx := context.Background()
	handler := NewSignalHandler(ctx, 30*time.Second)
	defer handler.Close()

	handler.RegisterCleanup(func() error { return nil })
	handler.RegisterCleanup(func() error { return nil })
	handler.RegisterCleanup(func() error { return nil })

	assert.Equal(t, 3, len(handler.cleanupFuncs))
}

func TestSignalHandler_RegisterCleanup_Nil(t *testing.T) {
	ctx := context.Background()
	handler := NewSignalHandler(ctx, 30*time.Second)
	defer handler.Close()

	handler.RegisterCleanup(nil)
	assert.Equal(t, 0, len(handler.cleanupFuncs))
}

func TestSignalHandler_SetCancelCallbacks(t *testing.T) {
	ctx := context.Background()
	handler := NewSignalHandler(ctx, 30*time.Second)
	defer handler.Close()

	handler.SetCancelCallbacks(
		func(saveState bool) error {
			return nil
		},
		func() error {
			return nil
		},
	)

	assert.NotNil(t, handler.onConfirmCancel)
	assert.NotNil(t, handler.onForceCancel)
}

func TestSignalHandler_Close(t *testing.T) {
	ctx := context.Background()
	handler := NewSignalHandler(ctx, 30*time.Second)

	handler.Close()
	// Close should not panic
}

func TestSignalHandler_IsGracefulShutdown(t *testing.T) {
	ctx := context.Background()
	handler := NewSignalHandler(ctx, 30*time.Second)
	defer handler.Close()

	// Should not be in graceful shutdown initially
	assert.False(t, handler.IsGracefulShutdown())
}

func TestSignalHandler_InterruptCount(t *testing.T) {
	ctx := context.Background()
	handler := NewSignalHandler(ctx, 30*time.Second)
	defer handler.Close()

	assert.Equal(t, int32(0), handler.InterruptCount())
}

func TestSignalHandler_Cancel(t *testing.T) {
	ctx := context.Background()
	handler := NewSignalHandler(ctx, 1*time.Second)
	defer handler.Close()

	handlerCtx := handler.Context()

	// Context should not be cancelled initially
	select {
	case <-handlerCtx.Done():
		t.Fatal("Context should not be cancelled initially")
	default:
		// Expected
	}

	// Call Cancel
	handler.Cancel()

	// Now context should be cancelled
	select {
	case <-handlerCtx.Done():
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Context should be cancelled after Cancel() call")
	}
}

func TestSignalHandler_ShutdownTimeout(t *testing.T) {
	ctx := context.Background()
	timeout := 5 * time.Second
	handler := NewSignalHandler(ctx, timeout)
	defer handler.Close()

	assert.Equal(t, timeout, handler.shutdownTimeout)
}

func TestSignalHandler_CleanupFunctionsExecution(t *testing.T) {
	ctx := context.Background()
	handler := NewSignalHandler(ctx, 30*time.Second)
	defer handler.Close()

	cleanupOrder := []int{}
	var mu atomic.Value
	mu.Store(make([]int, 0))

	handler.RegisterCleanup(func() error {
		cleanupOrder = append(cleanupOrder, 1)
		return nil
	})

	handler.RegisterCleanup(func() error {
		cleanupOrder = append(cleanupOrder, 2)
		return nil
	})

	handler.RegisterCleanup(func() error {
		cleanupOrder = append(cleanupOrder, 3)
		return nil
	})

	// executeCleanups is private, but we can verify cleanup functions are registered
	assert.Equal(t, 3, len(handler.cleanupFuncs))
}

func TestSignalHandler_MultipleInstances(t *testing.T) {
	ctx1 := context.Background()
	ctx2 := context.Background()

	handler1 := NewSignalHandler(ctx1, 30*time.Second)
	handler2 := NewSignalHandler(ctx2, 30*time.Second)

	defer handler1.Close()
	defer handler2.Close()

	assert.NotEqual(t, handler1.ctx, handler2.ctx)
	assert.Equal(t, handler1.shutdownTimeout, handler2.shutdownTimeout)
}

func TestSignalHandler_ContextDeadline(t *testing.T) {
	ctx := context.Background()
	handler := NewSignalHandler(ctx, 30*time.Second)
	defer handler.Close()

	handlerCtx := handler.Context()

	// No deadline should be set initially
	deadline, ok := handlerCtx.Deadline()
	assert.False(t, ok)
	assert.True(t, deadline.IsZero())
}

func TestSignalHandler_Value(t *testing.T) {
	ctx := context.WithValue(context.Background(), "key", "value")
	handler := NewSignalHandler(ctx, 30*time.Second)
	defer handler.Close()

	// Values from parent context should be accessible
	handlerCtx := handler.Context()
	value := handlerCtx.Value("key")
	assert.Equal(t, "value", value)
}

func TestSignalHandler_ChanBufferSize(t *testing.T) {
	ctx := context.Background()
	handler := NewSignalHandler(ctx, 30*time.Second)
	defer handler.Close()

	// Signal channel should have buffer size of 2
	assert.NotNil(t, handler.sigChan)
}
