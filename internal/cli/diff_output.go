package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/pmezard/go-difflib/difflib"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// taskDiffs renders the before/after a module reported (--diff) as unified
// diffs
func taskDiffs(t types.TaskResult) string {
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

// printDiffs writes the diffs of every changed task of a run
func printDiffs(w io.Writer, result *types.PlaybookResult) {
	first := true
	for _, play := range result.Plays {
		for _, host := range play.Hosts {
			for _, t := range host.Tasks {
				text := taskDiffs(t)
				if text == "" {
					continue
				}
				if first {
					fmt.Fprintln(w, "\nChanges:")
					first = false
				}
				fmt.Fprintf(w, "\n--- %s: %s\n%s", host.Host, t.TaskName, text)
			}
		}
	}
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
