package taskpreview

import (
	"fmt"
	"strings"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// ExecutionStatus represents whether a task will execute or be skipped
type ExecutionStatus string

const (
	StatusExecute       ExecutionStatus = "execute"
	StatusSkipNever     ExecutionStatus = "skip_never"
	StatusSkipTags      ExecutionStatus = "skip_tags"
	StatusSkipSkipTags  ExecutionStatus = "skip_skip_tags"
	StatusSkipCondition ExecutionStatus = "skip_condition"
	StatusUnconditional ExecutionStatus = "unconditional"
)

// TaskPreview represents a single task preview
type TaskPreview struct {
	Name         string
	Module       string
	Tags         []string
	Status       ExecutionStatus
	SkipReason   string // Why task is skipped
	PlayIndex    int
	PlayName     string
	TaskIndex    int
	Type         string // "task", "pre_task", "post_task", "handler"
	HasCondition bool   // true if task has "when" condition
	HasLoop      bool   // true if task has loop
}

// PlayPreview represents tasks in a play
type PlayPreview struct {
	Index   int
	Name    string
	Hosts   string
	Tasks   []TaskPreview
	Summary PlaySummary
}

// PlaySummary contains play-level statistics
type PlaySummary struct {
	Total    int
	Would    int
	Skipped  int
	SkipInfo map[string]int // count of skips by reason
}

// PreviewResult contains the full preview
type PreviewResult struct {
	Plays         []PlayPreview
	GlobalSummary GlobalSummary
	Tags          []string // Applied tag filters
	SkipTags      []string // Applied skip-tag filters
}

// GlobalSummary contains overall statistics
type GlobalSummary struct {
	TotalTasks   int
	WouldExecute int
	Skipped      int
	SkipDetails  map[string]int // count by skip reason
}

// PreviewTasks creates a task execution preview
func PreviewTasks(playbook *types.Playbook, tags, skipTags string) (*PreviewResult, error) {
	if playbook == nil {
		return nil, fmt.Errorf("playbook cannot be nil")
	}

	// Parse tag filters
	tagList := parseTags(tags)
	skipTagList := parseTags(skipTags)

	result := &PreviewResult{
		Tags:     tagList,
		SkipTags: skipTagList,
		GlobalSummary: GlobalSummary{
			SkipDetails: make(map[string]int),
		},
	}

	// Process each play
	for playIdx, play := range playbook.Plays {
		playPreview := PlayPreview{
			Index: playIdx,
			Name:  play.Name,
			Hosts: play.Hosts,
			Tasks: []TaskPreview{},
			Summary: PlaySummary{
				SkipInfo: make(map[string]int),
			},
		}

		// Combine all task types
		allTasks := []struct {
			tasks   []types.Task
			typeStr string
		}{
			{play.PreTasks, "pre_task"},
			{play.Tasks, "task"},
			{play.PostTasks, "post_task"},
			{play.Handlers, "handler"},
		}

		for _, taskGroup := range allTasks {
			for taskIdx, task := range taskGroup.tasks {
				preview := previewTask(&task, playIdx, play.Name, taskIdx, taskGroup.typeStr, tagList, skipTagList)
				playPreview.Tasks = append(playPreview.Tasks, preview)

				playPreview.Summary.Total++
				result.GlobalSummary.TotalTasks++

				if preview.Status == StatusExecute || preview.Status == StatusUnconditional {
					playPreview.Summary.Would++
					result.GlobalSummary.WouldExecute++
				} else {
					playPreview.Summary.Skipped++
					result.GlobalSummary.Skipped++
					reason := getSkipReason(preview.Status)
					playPreview.Summary.SkipInfo[reason]++
					result.GlobalSummary.SkipDetails[reason]++
				}
			}
		}

		result.Plays = append(result.Plays, playPreview)
	}

	return result, nil
}

// previewTask determines if a task would execute
func previewTask(task *types.Task, playIdx int, playName string, taskIdx int, taskType string, tags, skipTags []string) TaskPreview {
	preview := TaskPreview{
		Name:         task.Name,
		Module:       task.Module,
		Tags:         task.Tags,
		PlayIndex:    playIdx,
		PlayName:     playName,
		TaskIndex:    taskIdx,
		Type:         taskType,
		HasCondition: task.When != "",
		HasLoop:      task.Loop != nil,
	}

	// Check for "never" tag
	for _, tag := range task.Tags {
		if tag == "never" {
			preview.Status = StatusSkipNever
			preview.SkipReason = "Task has 'never' tag"
			return preview
		}
	}

	// Check for "always" tag - always executes even if tags don't match
	hasAlwaysTag := false
	for _, tag := range task.Tags {
		if tag == "always" {
			hasAlwaysTag = true
			break
		}
	}

	if hasAlwaysTag {
		preview.Status = StatusUnconditional
		preview.SkipReason = ""
		return preview
	}

	// If tags are specified, check if task matches any tag
	if len(tags) > 0 {
		matches := false
		for _, filter := range tags {
			for _, taskTag := range task.Tags {
				if taskTag == filter {
					matches = true
					break
				}
			}
			if matches {
				break
			}
		}

		if !matches {
			preview.Status = StatusSkipTags
			preview.SkipReason = "Task tags don't match filters"
			return preview
		}
	}

	// If skip-tags are specified, check if task matches any skip-tag
	for _, skipFilter := range skipTags {
		for _, taskTag := range task.Tags {
			if taskTag == skipFilter {
				preview.Status = StatusSkipSkipTags
				preview.SkipReason = fmt.Sprintf("Task matches skip-tag: %s", skipFilter)
				return preview
			}
		}
	}

	// If task has no tags, it always executes (unless skipped by other rules)
	if len(task.Tags) == 0 {
		preview.Status = StatusExecute
		preview.SkipReason = ""
		return preview
	}

	// Task has tags and passes all filters
	preview.Status = StatusExecute
	preview.SkipReason = ""
	return preview
}

// getSkipReason returns a human-readable skip reason
func getSkipReason(status ExecutionStatus) string {
	switch status {
	case StatusSkipNever:
		return "never tag"
	case StatusSkipTags:
		return "tag mismatch"
	case StatusSkipSkipTags:
		return "skip-tag match"
	case StatusSkipCondition:
		return "condition failed"
	default:
		return "unknown"
	}
}

// parseTags parses a comma-separated string of tags
func parseTags(tagsStr string) []string {
	if tagsStr == "" {
		return []string{}
	}

	var tags []string
	for _, tag := range strings.Split(tagsStr, ",") {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			tags = append(tags, tag)
		}
	}
	return tags
}
