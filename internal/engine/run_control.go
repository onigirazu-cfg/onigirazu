package engine

import (
	"context"
	"errors"
	"sync"
)

// ErrStoppedByUser ends a run that was asked to stop gracefully
var ErrStoppedByUser = errors.New("stopped by user")

// RunControl pauses and stops a run between tasks: running tasks finish,
// the next one waits while paused and does not start once stopped
type RunControl struct {
	mu      sync.Mutex
	paused  bool
	stopped bool
	wake    chan struct{}
}

// NewRunControl creates a run control in the running state
func NewRunControl() *RunControl {
	return &RunControl{wake: make(chan struct{})}
}

// Pause holds the run before its next task
func (c *RunControl) Pause() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.paused = true
}

// Resume lets a paused run go on
func (c *RunControl) Resume() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.paused {
		c.paused = false
		c.broadcast()
	}
}

// Stop makes the run end before its next task
func (c *RunControl) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.stopped {
		c.stopped = true
		c.broadcast()
	}
}

// Paused tells whether the run is paused
func (c *RunControl) Paused() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.paused
}

// Stopped tells whether a stop was asked for
func (c *RunControl) Stopped() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.stopped
}

func (c *RunControl) broadcast() {
	close(c.wake)
	c.wake = make(chan struct{})
}

// Checkpoint is called before a task starts: it waits while the run is
// paused and returns ErrStoppedByUser once a stop was asked for
func (c *RunControl) Checkpoint(ctx context.Context) error {
	if c == nil {
		return nil
	}
	for {
		c.mu.Lock()
		stopped, paused, wake := c.stopped, c.paused, c.wake
		c.mu.Unlock()
		if stopped {
			return ErrStoppedByUser
		}
		if !paused {
			return nil
		}
		select {
		case <-wake:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
