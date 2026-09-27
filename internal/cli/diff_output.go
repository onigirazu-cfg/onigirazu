package cli

import (
	"fmt"
	"io"

	"github.com/onigirazu-cfg/onigirazu/internal/diffview"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// printDiffs writes the diffs of every changed task of a run
func printDiffs(w io.Writer, result *types.PlaybookResult) {
	first := true
	for _, play := range result.Plays {
		for _, host := range play.Hosts {
			for _, t := range host.Tasks {
				text := diffview.TaskDiff(t)
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
