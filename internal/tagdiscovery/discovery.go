package tagdiscovery

import (
	"fmt"
	"sort"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// TagInfo contains information about a tag
type TagInfo struct {
	Name      string
	Count     int
	IsSpecial bool // true for 'always', 'never'
	TaskNames []string
}

// TagsSummary contains summary statistics about tags
type TagsSummary struct {
	TotalTags     int
	TotalTasks    int
	AlwaysTasks   int
	NeverTasks    int
	TaggedTasks   int
	UntaggedTasks int
	UniqueTags    int
}

// ListTagsResult contains all discovered tags
type ListTagsResult struct {
	Tags    map[string]*TagInfo
	Special map[string]*TagInfo // 'always', 'never'
	Summary TagsSummary
}

// DiscoverTags extracts all unique tags from a playbook
func DiscoverTags(playbook *types.Playbook) (*ListTagsResult, error) {
	if playbook == nil {
		return nil, fmt.Errorf("playbook cannot be nil")
	}

	result := &ListTagsResult{
		Tags:    make(map[string]*TagInfo),
		Special: make(map[string]*TagInfo),
		Summary: TagsSummary{},
	}

	allTags := make(map[string]*TagInfo)
	taggedTaskCount := 0
	alwaysCount := 0
	neverCount := 0
	totalTaskCount := 0

	// Process all plays
	for _, play := range playbook.Plays {
		// Process regular tasks
		for _, task := range play.Tasks {
			totalTaskCount++
			result.processTaskTags(&task, allTags, &alwaysCount, &neverCount, &taggedTaskCount)
		}

		// Process pre-tasks
		for _, task := range play.PreTasks {
			totalTaskCount++
			result.processTaskTags(&task, allTags, &alwaysCount, &neverCount, &taggedTaskCount)
		}

		// Process post-tasks
		for _, task := range play.PostTasks {
			totalTaskCount++
			result.processTaskTags(&task, allTags, &alwaysCount, &neverCount, &taggedTaskCount)
		}

		// Process handlers
		for _, task := range play.Handlers {
			totalTaskCount++
			result.processTaskTags(&task, allTags, &alwaysCount, &neverCount, &taggedTaskCount)
		}
	}

	// Separate regular tags from special tags
	for tagName, tagInfo := range allTags {
		if tagName == "always" || tagName == "never" {
			result.Special[tagName] = tagInfo
		} else {
			result.Tags[tagName] = tagInfo
		}
	}

	// Build summary
	result.Summary = TagsSummary{
		TotalTags:     len(result.Tags),
		TotalTasks:    totalTaskCount,
		AlwaysTasks:   alwaysCount,
		NeverTasks:    neverCount,
		TaggedTasks:   taggedTaskCount,
		UntaggedTasks: totalTaskCount - taggedTaskCount,
		UniqueTags:    len(result.Tags) + len(result.Special),
	}

	return result, nil
}

// processTaskTags processes tags for a single task
func (r *ListTagsResult) processTaskTags(task *types.Task, allTags map[string]*TagInfo, alwaysCount, neverCount, taggedTaskCount *int) {
	if len(task.Tags) == 0 {
		return
	}

	*taggedTaskCount++

	for _, tag := range task.Tags {
		if tag == "always" {
			*alwaysCount++
		} else if tag == "never" {
			*neverCount++
		}

		if _, exists := allTags[tag]; !exists {
			allTags[tag] = &TagInfo{
				Name:      tag,
				TaskNames: []string{},
			}
		}

		allTags[tag].Count++
		allTags[tag].TaskNames = append(allTags[tag].TaskNames, task.Name)
	}
}

// GetSortedTags returns tags sorted by count (descending) then by name
func (r *ListTagsResult) GetSortedTags() []*TagInfo {
	var tags []*TagInfo
	for _, tag := range r.Tags {
		tags = append(tags, tag)
	}

	sort.Slice(tags, func(i, j int) bool {
		if tags[i].Count != tags[j].Count {
			return tags[i].Count > tags[j].Count
		}
		return tags[i].Name < tags[j].Name
	})

	return tags
}

// GetSortedSpecialTags returns special tags (always, never)
func (r *ListTagsResult) GetSortedSpecialTags() []*TagInfo {
	var tags []*TagInfo
	for _, tag := range r.Special {
		tags = append(tags, tag)
	}

	sort.Slice(tags, func(i, j int) bool {
		// always comes before never
		if tags[i].Name == "always" && tags[j].Name == "never" {
			return true
		}
		if tags[i].Name == "never" && tags[j].Name == "always" {
			return false
		}
		return tags[i].Name < tags[j].Name
	})

	return tags
}
