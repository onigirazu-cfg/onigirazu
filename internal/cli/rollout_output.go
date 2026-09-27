package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// printRollout writes how the batches of a checked rollout went
func printRollout(w io.Writer, batches []types.BatchReport) {
	if len(batches) == 0 {
		return
	}
	fmt.Fprintln(w, "\nRollout:")
	for _, b := range batches {
		name := fmt.Sprintf("batch %d/%d", b.Batch, b.Batches)
		if b.Canary {
			name = "canary " + name
		}
		hosts := strings.Join(b.Hosts, ", ")
		if len(hosts) > 60 {
			hosts = fmt.Sprintf("%d hosts", len(b.Hosts))
		}
		if b.Healthy {
			fmt.Fprintf(w, "  ✓ %s (%s): healthy\n", name, hosts)
			continue
		}
		fmt.Fprintf(w, "  ✗ %s (%s): unhealthy, %s: %s\n", name, hosts, b.Reason, strings.Join(b.UnhealthyHosts, ", "))
		if b.RolledBack {
			line := fmt.Sprintf("    rolled back: %d change(s) undone", b.Undone)
			if b.HealthyAfterRollback != nil {
				if *b.HealthyAfterRollback {
					line += ", healthy again"
				} else {
					line += ", STILL UNHEALTHY"
				}
			}
			fmt.Fprintln(w, line)
			for _, k := range b.Irreversible {
				fmt.Fprintf(w, "    kept (no previous state): %s\n", k)
			}
			for _, f := range b.RollbackErrors {
				fmt.Fprintf(w, "    NOT undone: %s\n", f)
			}
		}
	}
}
