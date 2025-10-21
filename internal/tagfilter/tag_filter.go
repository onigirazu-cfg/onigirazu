package tagfilter

import (
	"fmt"
	"strings"
)

// Filter represents tag-based filtering options
type Filter struct {
	Tags     []string // List of tags to include (empty = all)
	SkipTags []string // List of tags to exclude
	Mode     FilterMode
}

// FilterMode represents different tag filtering modes
type FilterMode int

const (
	// ModeAll runs all tasks except those with 'never' tag (default)
	ModeAll FilterMode = iota
	// ModeTagged runs only tasks with at least one tag
	ModeTagged
	// ModeUntagged runs only tasks with no tags
	ModeUntagged
	// ModeSpecific runs only tasks matching specific tags
	ModeSpecific
)

const (
	// TagAlways - tasks with this tag always run
	TagAlways = "always"
	// TagNever - tasks with this tag never run
	TagNever = "never"
)

// New creates a new tag filter from raw tag and skip-tag strings
func New(tagsStr, skipTagsStr string) (*Filter, error) {
	f := &Filter{
		Tags:     []string{},
		SkipTags: []string{},
		Mode:     ModeAll,
	}

	// Parse skip tags
	if skipTagsStr != "" {
		for _, tag := range strings.Split(skipTagsStr, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				f.SkipTags = append(f.SkipTags, tag)
			}
		}
	}

	// Parse tags and determine mode
	if tagsStr != "" {
		tagsStr = strings.TrimSpace(tagsStr)
		switch tagsStr {
		case "tagged":
			f.Mode = ModeTagged
		case "untagged":
			f.Mode = ModeUntagged
		case "all":
			f.Mode = ModeAll
		default:
			f.Mode = ModeSpecific
			for _, tag := range strings.Split(tagsStr, ",") {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					f.Tags = append(f.Tags, tag)
				}
			}
		}
	}

	return f, nil
}

// ShouldRun determines if a task with given tags should be executed
func (f *Filter) ShouldRun(taskTags []string) bool {
	// Tasks with 'always' tag always run
	if hasTag(taskTags, TagAlways) {
		// Unless explicitly skipped
		return !hasTag(f.SkipTags, TagAlways)
	}

	// Tasks with 'never' tag never run
	if hasTag(taskTags, TagNever) {
		return false
	}

	// Check skip tags (highest priority after never/always)
	for _, skipTag := range f.SkipTags {
		if hasTag(taskTags, skipTag) {
			return false
		}
	}

	// Apply filter mode
	switch f.Mode {
	case ModeTagged:
		// Run only if has at least one tag
		return len(taskTags) > 0

	case ModeUntagged:
		// Run only if has no tags
		return len(taskTags) == 0

	case ModeSpecific:
		// Run if matches any of the specified tags
		if len(f.Tags) == 0 {
			// No tags specified in ModeSpecific shouldn't happen, but treat as run all
			return true
		}
		return hasAnyTag(taskTags, f.Tags)

	case ModeAll:
		// Run all tasks (default behavior)
		return true

	default:
		return true
	}
}

// hasTag checks if a specific tag is in the tag list
func hasTag(tags []string, tag string) bool {
	for _, t := range tags {
		if strings.EqualFold(t, tag) {
			return true
		}
	}
	return false
}

// hasAnyTag checks if any of the provided tags are in the tag list
func hasAnyTag(taskTags, filterTags []string) bool {
	for _, filterTag := range filterTags {
		if hasTag(taskTags, filterTag) {
			return true
		}
	}
	return false
}

// IsEmpty returns true if no filtering is active
func (f *Filter) IsEmpty() bool {
	return f.Mode == ModeAll && len(f.SkipTags) == 0
}

// String returns a string representation of the filter
func (f *Filter) String() string {
	var parts []string

	switch f.Mode {
	case ModeTagged:
		parts = append(parts, "tags=tagged")
	case ModeUntagged:
		parts = append(parts, "tags=untagged")
	case ModeSpecific:
		if len(f.Tags) > 0 {
			parts = append(parts, fmt.Sprintf("tags=%s", strings.Join(f.Tags, ",")))
		}
	}

	if len(f.SkipTags) > 0 {
		parts = append(parts, fmt.Sprintf("skip_tags=%s", strings.Join(f.SkipTags, ",")))
	}

	if len(parts) == 0 {
		return "no filtering"
	}
	return strings.Join(parts, " ")
}
