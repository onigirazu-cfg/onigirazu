package tagdiscovery

import (
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestDiscoverTags_NilPlaybook(t *testing.T) {
	result, err := DiscoverTags(nil)
	if err == nil {
		t.Error("Expected error for nil playbook, got nil")
	}
	if result != nil {
		t.Error("Expected nil result for nil playbook")
	}
}

func TestDiscoverTags_EmptyPlaybook(t *testing.T) {
	playbook := &types.Playbook{
		Plays: []types.Play{},
	}

	result, err := DiscoverTags(playbook)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if len(result.Tags) != 0 {
		t.Errorf("Expected 0 tags, got %d", len(result.Tags))
	}

	if result.Summary.TotalTags != 0 {
		t.Errorf("Expected 0 total tasks, got %d", result.Summary.TotalTags)
	}
}

func TestDiscoverTags_SimpleTags(t *testing.T) {
	playbook := &types.Playbook{
		Plays: []types.Play{
			{
				Name:  "Test Play",
				Hosts: "localhost",
				Tasks: []types.Task{
					{
						Name: "Task 1",
						Tags: []string{"setup", "deployment"},
					},
					{
						Name: "Task 2",
						Tags: []string{"setup"},
					},
					{
						Name: "Task 3",
						Tags: []string{"test"},
					},
				},
			},
		},
	}

	result, err := DiscoverTags(playbook)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result.Tags) != 3 {
		t.Errorf("Expected 3 unique tags, got %d", len(result.Tags))
	}

	if setupTag, exists := result.Tags["setup"]; exists {
		if setupTag.Count != 2 {
			t.Errorf("Expected setup count 2, got %d", setupTag.Count)
		}
	} else {
		t.Error("Expected 'setup' tag to exist")
	}

	if result.Summary.TotalTags != 3 {
		t.Errorf("Expected 3 total tasks, got %d", result.Summary.TotalTags)
	}

	if result.Summary.UniqueTags != 3 {
		t.Errorf("Expected 3 unique tags, got %d", result.Summary.UniqueTags)
	}
}

func TestDiscoverTags_SpecialTags(t *testing.T) {
	playbook := &types.Playbook{
		Plays: []types.Play{
			{
				Name:  "Test Play",
				Hosts: "localhost",
				Tasks: []types.Task{
					{
						Name: "Always Task",
						Tags: []string{"always"},
					},
					{
						Name: "Never Task",
						Tags: []string{"never"},
					},
					{
						Name: "Normal Task",
						Tags: []string{"setup"},
					},
				},
			},
		},
	}

	result, err := DiscoverTags(playbook)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result.Special) != 2 {
		t.Errorf("Expected 2 special tags, got %d", len(result.Special))
	}

	if result.Summary.AlwaysTasks != 1 {
		t.Errorf("Expected 1 always task, got %d", result.Summary.AlwaysTasks)
	}

	if result.Summary.NeverTasks != 1 {
		t.Errorf("Expected 1 never task, got %d", result.Summary.NeverTasks)
	}
}

func TestDiscoverTags_UntaggedTasks(t *testing.T) {
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

	result, err := DiscoverTags(playbook)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Summary.UntaggedTasks != 1 {
		t.Errorf("Expected 1 untagged task, got %d", result.Summary.UntaggedTasks)
	}

	if result.Summary.TaggedTasks != 1 {
		t.Errorf("Expected 1 tagged task, got %d", result.Summary.TaggedTasks)
	}
}

func TestDiscoverTags_AllTaskTypes(t *testing.T) {
	playbook := &types.Playbook{
		Plays: []types.Play{
			{
				Name:  "Test Play",
				Hosts: "localhost",
				PreTasks: []types.Task{
					{
						Name: "Pre Task",
						Tags: []string{"setup"},
					},
				},
				Tasks: []types.Task{
					{
						Name: "Main Task",
						Tags: []string{"setup"},
					},
				},
				PostTasks: []types.Task{
					{
						Name: "Post Task",
						Tags: []string{"cleanup"},
					},
				},
				Handlers: []types.Task{
					{
						Name: "Handler",
						Tags: []string{"notify"},
					},
				},
			},
		},
	}

	result, err := DiscoverTags(playbook)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Total tasks: 1 pre + 1 main + 1 post + 1 handler = 4
	if result.Summary.TotalTasks != 4 {
		t.Errorf("Expected 4 total tasks, got %d", result.Summary.TotalTasks)
	}

	// Should have 3 unique tags: setup (appears twice), cleanup, notify
	if result.Summary.UniqueTags != 3 {
		t.Errorf("Expected 3 unique tags, got %d", result.Summary.UniqueTags)
	}
}

func TestDiscoverTags_MultiplePlays(t *testing.T) {
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
						Tags: []string{"setup", "deploy"},
					},
				},
			},
		},
	}

	result, err := DiscoverTags(playbook)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Summary.TotalTags != 2 {
		t.Errorf("Expected 2 total tasks, got %d", result.Summary.TotalTags)
	}

	if result.Summary.UniqueTags != 2 {
		t.Errorf("Expected 2 unique tags (setup, deploy), got %d", result.Summary.UniqueTags)
	}

	if setupTag, exists := result.Tags["setup"]; exists {
		if setupTag.Count != 2 {
			t.Errorf("Expected setup tag count 2, got %d", setupTag.Count)
		}
	} else {
		t.Error("Expected 'setup' tag to exist")
	}
}

func TestDiscoverTags_GetSortedTags(t *testing.T) {
	playbook := &types.Playbook{
		Plays: []types.Play{
			{
				Name:  "Test Play",
				Hosts: "localhost",
				Tasks: []types.Task{
					{
						Name: "Task 1",
						Tags: []string{"setup", "setup", "setup"},
					},
					{
						Name: "Task 2",
						Tags: []string{"deploy", "deploy"},
					},
					{
						Name: "Task 3",
						Tags: []string{"test"},
					},
				},
			},
		},
	}

	result, err := DiscoverTags(playbook)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	sorted := result.GetSortedTags()
	if len(sorted) != 3 {
		t.Errorf("Expected 3 sorted tags, got %d", len(sorted))
		return
	}

	// First tag should be 'setup' with count 3
	if sorted[0].Name != "setup" || sorted[0].Count != 3 {
		t.Errorf("Expected first tag to be 'setup' with count 3, got '%s' with count %d",
			sorted[0].Name, sorted[0].Count)
	}

	// Second tag should be 'deploy' with count 2
	if sorted[1].Name != "deploy" || sorted[1].Count != 2 {
		t.Errorf("Expected second tag to be 'deploy' with count 2, got '%s' with count %d",
			sorted[1].Name, sorted[1].Count)
	}

	// Third tag should be 'test' with count 1
	if sorted[2].Name != "test" || sorted[2].Count != 1 {
		t.Errorf("Expected third tag to be 'test' with count 1, got '%s' with count %d",
			sorted[2].Name, sorted[2].Count)
	}
}

func TestDiscoverTags_ComplexScenario(t *testing.T) {
	playbook := &types.Playbook{
		Plays: []types.Play{
			{
				Name:  "Production Deploy",
				Hosts: "prod",
				PreTasks: []types.Task{
					{
						Name: "Backup",
						Tags: []string{"backup"},
					},
				},
				Tasks: []types.Task{
					{
						Name: "Setup",
						Tags: []string{"setup", "always"},
					},
					{
						Name: "Configure",
						Tags: []string{"config", "production"},
					},
					{
						Name: "Deploy",
						Tags: []string{"deploy", "production"},
					},
					{
						Name: "Cleanup",
						Tags: []string{"never"},
					},
					{
						Name: "General",
						Tags: []string{},
					},
				},
				PostTasks: []types.Task{
					{
						Name: "Notify",
						Tags: []string{"notify", "production"},
					},
				},
			},
		},
	}

	result, err := DiscoverTags(playbook)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Total tasks: 1 pre + 5 main + 1 post = 7
	if result.Summary.TotalTasks != 7 {
		t.Errorf("Expected 7 total tasks, got %d", result.Summary.TotalTasks)
	}

	// Untagged: 1 (General)
	if result.Summary.UntaggedTasks != 1 {
		t.Errorf("Expected 1 untagged task, got %d", result.Summary.UntaggedTasks)
	}

	// Tagged: 6
	if result.Summary.TaggedTasks != 6 {
		t.Errorf("Expected 6 tagged tasks, got %d", result.Summary.TaggedTasks)
	}

	// Special tags: always, never
	if result.Summary.AlwaysTasks != 1 {
		t.Errorf("Expected 1 always task, got %d", result.Summary.AlwaysTasks)
	}

	if result.Summary.NeverTasks != 1 {
		t.Errorf("Expected 1 never task, got %d", result.Summary.NeverTasks)
	}

	// Unique tags: backup, setup, always, config, production, deploy, never, notify
	// That's 8 unique tags
	if result.Summary.UniqueTags != 8 {
		t.Errorf("Expected 8 unique tags, got %d", result.Summary.UniqueTags)
	}

	// production tag should appear 3 times
	if prodTag, exists := result.Tags["production"]; exists {
		if prodTag.Count != 3 {
			t.Errorf("Expected production tag count 3, got %d", prodTag.Count)
		}
	} else {
		t.Error("Expected 'production' tag to exist")
	}
}
