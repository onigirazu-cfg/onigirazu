package taskpreview

import (
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestPreviewTasks_NilPlaybook(t *testing.T) {
	result, err := PreviewTasks(nil, "", "")
	if err == nil {
		t.Error("Expected error for nil playbook, got nil")
	}
	if result != nil {
		t.Error("Expected nil result for nil playbook")
	}
}

func TestPreviewTasks_EmptyPlaybook(t *testing.T) {
	playbook := &types.Playbook{
		Plays: []types.Play{},
	}

	result, err := PreviewTasks(playbook, "", "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if len(result.Plays) != 0 {
		t.Errorf("Expected 0 plays, got %d", len(result.Plays))
	}

	if result.GlobalSummary.TotalTasks != 0 {
		t.Errorf("Expected 0 total tasks, got %d", result.GlobalSummary.TotalTasks)
	}
}

func TestPreviewTasks_AllTasksExecute(t *testing.T) {
	playbook := &types.Playbook{
		Plays: []types.Play{
			{
				Name:  "Test Play",
				Hosts: "localhost",
				Tasks: []types.Task{
					{
						Name: "Task 1",
						Tags: []string{"setup"},
					},
					{
						Name: "Task 2",
						Tags: []string{"config"},
					},
				},
			},
		},
	}

	result, err := PreviewTasks(playbook, "", "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.GlobalSummary.TotalTasks != 2 {
		t.Errorf("Expected 2 total tasks, got %d", result.GlobalSummary.TotalTasks)
	}

	if result.GlobalSummary.WouldExecute != 2 {
		t.Errorf("Expected 2 tasks to execute, got %d", result.GlobalSummary.WouldExecute)
	}

	if result.GlobalSummary.Skipped != 0 {
		t.Errorf("Expected 0 skipped tasks, got %d", result.GlobalSummary.Skipped)
	}
}

func TestPreviewTasks_NeverTag(t *testing.T) {
	playbook := &types.Playbook{
		Plays: []types.Play{
			{
				Name:  "Test Play",
				Hosts: "localhost",
				Tasks: []types.Task{
					{
						Name: "Regular Task",
						Tags: []string{"setup"},
					},
					{
						Name: "Never Task",
						Tags: []string{"never"},
					},
				},
			},
		},
	}

	result, err := PreviewTasks(playbook, "", "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.GlobalSummary.TotalTasks != 2 {
		t.Errorf("Expected 2 total tasks, got %d", result.GlobalSummary.TotalTasks)
	}

	if result.GlobalSummary.WouldExecute != 1 {
		t.Errorf("Expected 1 task to execute, got %d", result.GlobalSummary.WouldExecute)
	}

	if result.GlobalSummary.Skipped != 1 {
		t.Errorf("Expected 1 skipped task, got %d", result.GlobalSummary.Skipped)
	}

	// Verify skip reason
	playPreview := result.Plays[0]
	for _, task := range playPreview.Tasks {
		if task.Name == "Never Task" {
			if task.Status != StatusSkipNever {
				t.Errorf("Expected status StatusSkipNever, got %s", task.Status)
			}
			break
		}
	}
}

func TestPreviewTasks_TagsFilter(t *testing.T) {
	playbook := &types.Playbook{
		Plays: []types.Play{
			{
				Name:  "Test Play",
				Hosts: "localhost",
				Tasks: []types.Task{
					{
						Name: "Task 1",
						Tags: []string{"setup"},
					},
					{
						Name: "Task 2",
						Tags: []string{"config"},
					},
					{
						Name: "Task 3",
						Tags: []string{"deploy"},
					},
				},
			},
		},
	}

	result, err := PreviewTasks(playbook, "setup", "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.GlobalSummary.WouldExecute != 1 {
		t.Errorf("Expected 1 task to execute with setup tag, got %d", result.GlobalSummary.WouldExecute)
	}

	if result.GlobalSummary.Skipped != 2 {
		t.Errorf("Expected 2 tasks to skip with setup tag filter, got %d", result.GlobalSummary.Skipped)
	}
}

func TestPreviewTasks_SkipTagsFilter(t *testing.T) {
	playbook := &types.Playbook{
		Plays: []types.Play{
			{
				Name:  "Test Play",
				Hosts: "localhost",
				Tasks: []types.Task{
					{
						Name: "Task 1",
						Tags: []string{"setup"},
					},
					{
						Name: "Task 2",
						Tags: []string{"debug"},
					},
					{
						Name: "Task 3",
						Tags: []string{"deploy"},
					},
				},
			},
		},
	}

	result, err := PreviewTasks(playbook, "", "debug")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.GlobalSummary.WouldExecute != 2 {
		t.Errorf("Expected 2 tasks to execute, got %d", result.GlobalSummary.WouldExecute)
	}

	if result.GlobalSummary.Skipped != 1 {
		t.Errorf("Expected 1 task to skip with debug skip-tag, got %d", result.GlobalSummary.Skipped)
	}
}

func TestPreviewTasks_UntaggedTasks(t *testing.T) {
	playbook := &types.Playbook{
		Plays: []types.Play{
			{
				Name:  "Test Play",
				Hosts: "localhost",
				Tasks: []types.Task{
					{
						Name: "Tagged Task",
						Tags: []string{"setup"},
					},
					{
						Name: "Untagged Task",
						Tags: []string{},
					},
				},
			},
		},
	}

	// Without filters, both should execute
	result, err := PreviewTasks(playbook, "", "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.GlobalSummary.WouldExecute != 2 {
		t.Errorf("Expected 2 tasks to execute (including untagged), got %d", result.GlobalSummary.WouldExecute)
	}

	// With tag filter, untagged tasks are skipped (Ansible behavior)
	result2, err := PreviewTasks(playbook, "setup", "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result2.GlobalSummary.WouldExecute != 1 {
		t.Errorf("Expected 1 task to execute (only setup tag, untagged is skipped), got %d", result2.GlobalSummary.WouldExecute)
	}
}

func TestPreviewTasks_MultipleTags(t *testing.T) {
	playbook := &types.Playbook{
		Plays: []types.Play{
			{
				Name:  "Test Play",
				Hosts: "localhost",
				Tasks: []types.Task{
					{
						Name: "Task 1",
						Tags: []string{"setup", "config"},
					},
					{
						Name: "Task 2",
						Tags: []string{"deploy", "config"},
					},
					{
						Name: "Task 3",
						Tags: []string{"cleanup"},
					},
				},
			},
		},
	}

	// Filter by config tag - should match tasks with config tag
	result, err := PreviewTasks(playbook, "config", "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.GlobalSummary.WouldExecute != 2 {
		t.Errorf("Expected 2 tasks with config tag, got %d", result.GlobalSummary.WouldExecute)
	}
}

func TestPreviewTasks_MultiplePlays(t *testing.T) {
	playbook := &types.Playbook{
		Plays: []types.Play{
			{
				Name:  "Play 1",
				Hosts: "localhost",
				Tasks: []types.Task{
					{
						Name: "Task 1",
						Tags: []string{"setup"},
					},
				},
			},
			{
				Name:  "Play 2",
				Hosts: "webservers",
				Tasks: []types.Task{
					{
						Name: "Task 2",
						Tags: []string{"setup"},
					},
					{
						Name: "Task 3",
						Tags: []string{"deploy"},
					},
				},
			},
		},
	}

	result, err := PreviewTasks(playbook, "", "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result.Plays) != 2 {
		t.Errorf("Expected 2 plays, got %d", len(result.Plays))
	}

	if result.GlobalSummary.TotalTasks != 3 {
		t.Errorf("Expected 3 total tasks, got %d", result.GlobalSummary.TotalTasks)
	}

	if result.GlobalSummary.WouldExecute != 3 {
		t.Errorf("Expected 3 tasks to execute, got %d", result.GlobalSummary.WouldExecute)
	}
}

func TestPlayPreview_Summary(t *testing.T) {
	playbook := &types.Playbook{
		Plays: []types.Play{
			{
				Name:  "Test Play",
				Hosts: "localhost",
				Tasks: []types.Task{
					{
						Name: "Task 1",
						Tags: []string{"setup"},
					},
					{
						Name: "Task 2",
						Tags: []string{"never"},
					},
					{
						Name: "Task 3",
						Tags: []string{"debug"},
					},
				},
			},
		},
	}

	result, err := PreviewTasks(playbook, "", "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	play := result.Plays[0]
	if play.Summary.Total != 3 {
		t.Errorf("Expected play summary total 3, got %d", play.Summary.Total)
	}

	if play.Summary.Would != 2 {
		t.Errorf("Expected play summary would execute 2, got %d", play.Summary.Would)
	}

	if play.Summary.Skipped != 1 {
		t.Errorf("Expected play summary skipped 1, got %d", play.Summary.Skipped)
	}
}
