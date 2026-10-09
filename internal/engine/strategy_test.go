package engine

import (
	"context"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestPlayStrategy(t *testing.T) {
	for in, want := range map[string]bool{"": false, "linear": false, "free": true} {
		got, err := playStrategyFree(in)
		if err != nil || got != want {
			t.Errorf("%q: %v %v", in, got, err)
		}
	}
	if _, err := playStrategyFree("host_pinned"); err == nil {
		t.Error("an unknown strategy must be an error")
	}
}

func TestPlayThrottleAppliesToTasksWithoutTheirOwn(t *testing.T) {
	e := &ExecutionEngine{playThrottle: "2"}
	s, err := e.throttleSlots(context.Background(), &types.Task{}, nil)
	if err != nil || cap(s) != 2 {
		t.Fatalf("play throttle: cap=%d err=%v", cap(s), err)
	}
	s, err = e.throttleSlots(context.Background(), &types.Task{Throttle: "5"}, nil)
	if err != nil || cap(s) != 5 {
		t.Fatalf("task throttle wins: cap=%d err=%v", cap(s), err)
	}
	e.playThrottle = ""
	if s, _ := e.throttleSlots(context.Background(), &types.Task{}, nil); s != nil {
		t.Fatal("no throttle: nil slots")
	}
}
