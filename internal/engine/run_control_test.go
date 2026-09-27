package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRunControl(t *testing.T) {
	var nilControl *RunControl
	assert.NoError(t, nilControl.Checkpoint(context.Background()))

	c := NewRunControl()
	assert.NoError(t, c.Checkpoint(context.Background()))

	c.Pause()
	done := make(chan error, 1)
	go func() { done <- c.Checkpoint(context.Background()) }()
	select {
	case <-done:
		t.Fatal("a paused run must wait")
	case <-time.After(50 * time.Millisecond):
	}
	c.Resume()
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("resume must release the run")
	}

	// stop wakes a paused run and ends it
	c.Pause()
	go func() { done <- c.Checkpoint(context.Background()) }()
	time.Sleep(20 * time.Millisecond)
	c.Stop()
	select {
	case err := <-done:
		assert.True(t, errors.Is(err, ErrStoppedByUser))
	case <-time.After(time.Second):
		t.Fatal("stop must release the run")
	}
	assert.True(t, c.Stopped())

	// a canceled context ends the wait
	p := NewRunControl()
	p.Pause()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	assert.Error(t, p.Checkpoint(ctx))
}
