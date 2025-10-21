package engine

import (
	"testing"

	"github.com/onigirazu-cfg/onigirazu/internal/tagfilter"
	"github.com/stretchr/testify/assert"
)

func TestExecutionEngine_SetTagFilter(t *testing.T) {
	engine, _, _, _, _, _ := createTestEngine()

	// Create a tag filter
	filter, err := tagfilter.New("setup,config", "debug")
	assert.NoError(t, err)

	// Set the filter
	engine.SetTagFilter(filter)

	// Verify it was set
	assert.NotNil(t, engine.tagFilter)
	assert.Equal(t, filter, engine.tagFilter)
}

func TestExecutionEngine_DefaultTagFilter(t *testing.T) {
	engine, _, _, _, _, _ := createTestEngine()

	// Verify default tag filter is set and allows all tags
	assert.NotNil(t, engine.tagFilter)
	assert.True(t, engine.tagFilter.ShouldRun([]string{}))
	assert.True(t, engine.tagFilter.ShouldRun([]string{"setup"}))
	assert.True(t, engine.tagFilter.ShouldRun([]string{"any", "tag"}))
}

func TestExecutionEngine_TagFilter_AllowsAlways(t *testing.T) {
	engine, _, _, _, _, _ := createTestEngine()

	// Create a filter that only allows "setup" tag
	filter, _ := tagfilter.New("setup", "")
	engine.SetTagFilter(filter)

	// Verify the filter allows "always" tag
	assert.True(t, engine.tagFilter.ShouldRun([]string{"always"}))
}

func TestExecutionEngine_TagFilter_BlocksNever(t *testing.T) {
	engine, _, _, _, _, _ := createTestEngine()

	// Create a filter that allows all tags (no restrictions)
	filter, _ := tagfilter.New("", "")
	engine.SetTagFilter(filter)

	// Verify the filter blocks "never" tag
	assert.False(t, engine.tagFilter.ShouldRun([]string{"never"}))
}

func TestExecutionEngine_TagFilter_SkipTags(t *testing.T) {
	engine, _, _, _, _, _ := createTestEngine()

	// Create a filter that skips "debug" tag
	filter, _ := tagfilter.New("", "debug")
	engine.SetTagFilter(filter)

	// Verify the filter blocks "debug" tag but allows others
	assert.False(t, engine.tagFilter.ShouldRun([]string{"debug"}))
	assert.True(t, engine.tagFilter.ShouldRun([]string{"production"}))
	assert.False(t, engine.tagFilter.ShouldRun([]string{"debug", "production"}))
}

func TestExecutionEngine_TagFilter_TaggedMode(t *testing.T) {
	engine, _, _, _, _, _ := createTestEngine()

	// Create a filter in "tagged" mode
	filter, _ := tagfilter.New("tagged", "")
	engine.SetTagFilter(filter)

	// Verify the filter only allows tagged tasks
	assert.False(t, engine.tagFilter.ShouldRun([]string{}))
	assert.True(t, engine.tagFilter.ShouldRun([]string{"production"}))
	assert.True(t, engine.tagFilter.ShouldRun([]string{"always"}))
}

func TestExecutionEngine_TagFilter_UntaggedMode(t *testing.T) {
	engine, _, _, _, _, _ := createTestEngine()

	// Create a filter in "untagged" mode
	filter, _ := tagfilter.New("untagged", "")
	engine.SetTagFilter(filter)

	// Verify the filter only allows untagged tasks (plus "always" override)
	assert.True(t, engine.tagFilter.ShouldRun([]string{}))
	assert.False(t, engine.tagFilter.ShouldRun([]string{"production"}))
	assert.True(t, engine.tagFilter.ShouldRun([]string{"always"}))
}

func TestExecutionEngine_TagFilter_SpecificTags(t *testing.T) {
	engine, _, _, _, _, _ := createTestEngine()

	// Create a filter for specific tags
	filter, _ := tagfilter.New("production,staging", "")
	engine.SetTagFilter(filter)

	// Verify the filter only allows specified tags
	assert.True(t, engine.tagFilter.ShouldRun([]string{"production"}))
	assert.True(t, engine.tagFilter.ShouldRun([]string{"staging"}))
	assert.False(t, engine.tagFilter.ShouldRun([]string{"dev"}))
	assert.True(t, engine.tagFilter.ShouldRun([]string{"always"}))
}

func TestExecutionEngine_TagFilter_Combined(t *testing.T) {
	engine, _, _, _, _, _ := createTestEngine()

	// Create a filter that allows "production,staging" but skips "test"
	filter, _ := tagfilter.New("production,staging", "test")
	engine.SetTagFilter(filter)

	// Verify combined behavior
	assert.True(t, engine.tagFilter.ShouldRun([]string{"production"}))
	assert.True(t, engine.tagFilter.ShouldRun([]string{"staging"}))
	assert.False(t, engine.tagFilter.ShouldRun([]string{"test"}))
	assert.False(t, engine.tagFilter.ShouldRun([]string{"production", "test"})) // skip takes precedence
	assert.False(t, engine.tagFilter.ShouldRun([]string{"dev"}))
	assert.True(t, engine.tagFilter.ShouldRun([]string{"always"}))
}
