package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/onigirazu-cfg/onigirazu/internal/tagdiscovery"
)

// FormatTagsText formats tags as human-readable text
func FormatTagsText(result *tagdiscovery.ListTagsResult) string {
	var output strings.Builder

	output.WriteString("Available tags in playbook:\n\n")

	// Regular tags
	sortedTags := result.GetSortedTags()
	if len(sortedTags) > 0 {
		output.WriteString("  Tag Name          Count\n")
		output.WriteString("  ─────────────────────────────────────\n")

		for _, tag := range sortedTags {
			output.WriteString(fmt.Sprintf("  %-17s %d\n", tag.Name, tag.Count))
		}
		output.WriteString("\n")
	}

	// Special tags
	specialTags := result.GetSortedSpecialTags()
	if len(specialTags) > 0 {
		output.WriteString("Special tags:\n")
		for _, tag := range specialTags {
			details := "Special tag"
			if tag.Name == "always" {
				details = "Always runs"
			} else if tag.Name == "never" {
				details = "Never runs by default"
			}
			output.WriteString(fmt.Sprintf("  %-17s %s (%d tasks)\n", tag.Name, details, tag.Count))
		}
		output.WriteString("\n")
	}

	// Summary
	output.WriteString("Summary:\n")
	output.WriteString(fmt.Sprintf("  Total unique tags:  %d\n", result.Summary.UniqueTags))
	output.WriteString(fmt.Sprintf("  Total tasks:        %d\n", result.Summary.TotalTasks))
	output.WriteString(fmt.Sprintf("  Tagged tasks:       %d\n", result.Summary.TaggedTasks))
	output.WriteString(fmt.Sprintf("  Untagged tasks:     %d\n", result.Summary.UntaggedTasks))

	if result.Summary.AlwaysTasks > 0 {
		output.WriteString(fmt.Sprintf("  Always tasks:       %d\n", result.Summary.AlwaysTasks))
	}
	if result.Summary.NeverTasks > 0 {
		output.WriteString(fmt.Sprintf("  Never tasks:        %d\n", result.Summary.NeverTasks))
	}

	return output.String()
}

// FormatTagsJSON formats tags as JSON
func FormatTagsJSON(result *tagdiscovery.ListTagsResult) string {
	output := map[string]interface{}{
		"tags": map[string]interface{}{
			"by_count": formatTagsForJSON(result.GetSortedTags()),
			"count":    len(result.Tags),
		},
		"special_tags": map[string]interface{}{
			"data":  formatTagsForJSON(result.GetSortedSpecialTags()),
			"count": len(result.Special),
		},
		"summary": result.Summary,
	}

	data, _ := json.MarshalIndent(output, "", "  ")
	return string(data)
}

// FormatTagsYAML formats tags as YAML
func FormatTagsYAML(result *tagdiscovery.ListTagsResult) string {
	output := map[string]interface{}{
		"tags": map[string]interface{}{
			"by_count": formatTagsForJSON(result.GetSortedTags()),
			"count":    len(result.Tags),
		},
		"special_tags": map[string]interface{}{
			"data":  formatTagsForJSON(result.GetSortedSpecialTags()),
			"count": len(result.Special),
		},
		"summary": result.Summary,
	}

	data, _ := yaml.Marshal(output)
	return string(data)
}

// FormatTagsCSV formats tags as CSV
func FormatTagsCSV(result *tagdiscovery.ListTagsResult) string {
	var output strings.Builder
	w := csv.NewWriter(&output)

	// Write header
	w.Write([]string{"Tag Name", "Type", "Count", "Tasks"})

	// Write regular tags
	for _, tag := range result.GetSortedTags() {
		w.Write([]string{
			tag.Name,
			"regular",
			fmt.Sprintf("%d", tag.Count),
			strings.Join(tag.TaskNames, "; "),
		})
	}

	// Write special tags
	for _, tag := range result.GetSortedSpecialTags() {
		tagType := "special"
		w.Write([]string{
			tag.Name,
			tagType,
			fmt.Sprintf("%d", tag.Count),
			strings.Join(tag.TaskNames, "; "),
		})
	}

	w.Flush()
	return output.String()
}

// formatTagsForJSON formats tags for JSON/YAML output
func formatTagsForJSON(tags []*tagdiscovery.TagInfo) []map[string]interface{} {
	var result []map[string]interface{}
	for _, tag := range tags {
		result = append(result, map[string]interface{}{
			"name":  tag.Name,
			"count": tag.Count,
			"tasks": tag.TaskNames,
		})
	}
	return result
}
