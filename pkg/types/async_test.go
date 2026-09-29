package types

import "testing"

func TestTaskAsyncPoll(t *testing.T) {
	var task Task
	if err := task.UnmarshalYAML(func(v interface{}) error {
		*(v.(*map[string]interface{})) = map[string]interface{}{"command": "sleep 1", "async": 60, "poll": "5"}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if task.Module != "command" || task.Async != 60 || task.Poll != 5 || !task.PollSet {
		t.Errorf("task = module %q async %d poll %d set %v", task.Module, task.Async, task.Poll, task.PollSet)
	}
}
