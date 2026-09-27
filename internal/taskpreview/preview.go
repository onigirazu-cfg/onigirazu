package taskpreview

import (
	"fmt"
	"strings"

	"github.com/onigirazu-cfg/onigirazu/internal/tagfilter"
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

	// Parse tag filters; the engine's filter decides, as in apply
	tagList := parseTags(tags)
	skipTagList := parseTags(skipTags)
	filter, err := tagfilter.New(tags, skipTags)
	if err != nil {
		return nil, err
	}

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

		for taskIdx, flat := range Flatten(&play) {
			preview := previewTask(flat, playIdx, play.Name, taskIdx, filter, skipTagList)
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

		result.Plays = append(result.Plays, playPreview)
	}

	return result, nil
}

// previewTask determines if a task would execute, with the tag filter the
// engine uses
func previewTask(flat FlatTask, playIdx int, playName string, taskIdx int, filter *tagfilter.Filter, skipTags []string) TaskPreview {
	task := flat.Task
	name := task.Name
	if flat.Role != "" {
		name = flat.Role + " : " + name
	}
	preview := TaskPreview{
		Name:         name,
		Module:       task.Module,
		Tags:         flat.Tags,
		PlayIndex:    playIdx,
		PlayName:     playName,
		TaskIndex:    taskIdx,
		Type:         flat.Type,
		HasCondition: task.When != "",
		HasLoop:      task.Loop != nil,
	}
	runs := filter.ShouldRun(flat.Tags)
	switch {
	case runs && hasTagFold(flat.Tags, "always"):
		preview.Status = StatusUnconditional
	case runs:
		preview.Status = StatusExecute
	case hasTagFold(flat.Tags, "never"):
		preview.Status = StatusSkipNever
		preview.SkipReason = "Task has 'never' tag"
	default:
		for _, skip := range skipTags {
			if hasTagFold(flat.Tags, skip) {
				preview.Status = StatusSkipSkipTags
				preview.SkipReason = fmt.Sprintf("Task matches skip-tag: %s", skip)
				return preview
			}
		}
		preview.Status = StatusSkipTags
		preview.SkipReason = "Task tags don't match filters"
	}
	return preview
}

func hasTagFold(tags []string, tag string) bool {
	for _, t := range tags {
		if strings.EqualFold(t, tag) {
			return true
		}
	}
	return false
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
