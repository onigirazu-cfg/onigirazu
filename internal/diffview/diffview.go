package diffview

import (
	"strings"

	"github.com/pmezard/go-difflib/difflib"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// TaskDiff renders the before/after a module reported (--diff) as unified
// diffs
func TaskDiff(t types.TaskResult) string {
	list, _ := t.Output["diff"].([]interface{})
	var b strings.Builder
	for _, item := range list {
		d, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		before, _ := d["before"].(string)
		after, _ := d["after"].(string)
		header, _ := d["before_header"].(string)
		text, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
			A: diffLines(before), B: diffLines(after),
			FromFile: "before: " + header, ToFile: "after: " + header, Context: 3,
		})
		if err != nil {
			continue
		}
		b.WriteString(text)
		if text != "" && !strings.HasSuffix(text, "\n") {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// diffLines splits text into lines that keep their newline; no empty line
// after the last newline
func diffLines(text string) []string {
	if text == "" {
		return nil
	}
	lines := strings.SplitAfter(text, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	} else {
		lines[len(lines)-1] += "\n\\ No newline at end of file\n"
	}
	return lines
}
