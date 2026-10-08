package modules

import (
	"context"
	"strings"
	"sync"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// A loop of a file module captures the targets of all its items in one
// round trip before the first item runs (LoopProbes), instead of one probe
// per item. The captures hold until an item changes something: items may
// touch the same files (lineinfile on one file), so after a change every
// item probes for itself again.

// probeRecordSep ends each record of a batched probe
const probeRecordSep = "\x1e\n"

// maxProbeBatch paths go in one command (it stays under Linux's limit of one
// argument)
const maxProbeBatch = 256

// LoopProbes are the captures of a loop's targets on one host
type LoopProbes struct {
	mu    sync.Mutex
	out   map[string]string
	stale bool
}

type loopProbesKey struct{}

// WithLoopProbes makes the captures available to the tasks run with ctx
func WithLoopProbes(ctx context.Context, lp *LoopProbes) context.Context {
	if lp == nil {
		return ctx
	}
	return context.WithValue(ctx, loopProbesKey{}, lp)
}

// loopProbe is the batched capture of path, if there is a current one
func loopProbe(ctx context.Context, path string) (string, bool) {
	lp, _ := ctx.Value(loopProbesKey{}).(*LoopProbes)
	if lp == nil {
		return "", false
	}
	lp.mu.Lock()
	defer lp.mu.Unlock()
	if lp.stale {
		return "", false
	}
	out, ok := lp.out[path]
	return out, ok
}

// invalidateLoopProbes: an item changed something, the captures are old
func invalidateLoopProbes(ctx context.Context) {
	if lp, _ := ctx.Value(loopProbesKey{}).(*LoopProbes); lp != nil {
		lp.mu.Lock()
		lp.stale = true
		lp.mu.Unlock()
	}
}

// CapturePathKeys are the arguments that name the target of a captured file
// module (nil for other modules)
func CapturePathKeys(module string) []string {
	return captureModules[module]
}

// PrefetchLoop captures paths on host with the task's escalation; nil when
// that is not possible (the items then probe one by one)
func PrefetchLoop(ctx context.Context, host types.Host, task *types.Task, paths []string) *LoopProbes {
	args := map[string]interface{}{}
	if task.Become {
		args["_become"] = true
		args["_become_user"] = task.BecomeUser
		args["_become_method"] = task.BecomeMethod
	}
	host.Become, host.BecomeUser, host.BecomeMethod = task.Become, task.BecomeUser, task.BecomeMethod
	seen := map[string]bool{}
	var unique []string
	for _, p := range paths {
		if p != "" && !seen[p] && !strings.ContainsAny(p, "\n\x00") {
			seen[p] = true
			unique = append(unique, p)
		}
	}
	if len(unique) < 2 {
		return nil
	}
	lp := &LoopProbes{out: make(map[string]string, len(unique))}
	for start := 0; start < len(unique); start += maxProbeBatch {
		chunk := unique[start:min(start+maxProbeBatch, len(unique))]
		out, err := probeMany(ctx, host, args, chunk)
		if err != nil {
			return nil
		}
		records := strings.Split(out, probeRecordSep)
		if len(records) != len(chunk)+1 || records[len(chunk)] != "" {
			return nil
		}
		for i, p := range chunk {
			lp.out[p] = records[i]
		}
	}
	return lp
}

// probeMany probes paths in one round trip: through the Python server, or
// one shell script
func probeMany(ctx context.Context, host types.Host, args map[string]interface{}, paths []string) (string, error) {
	exec, err := executor.NewCommandExecutor(host)
	if err == nil {
		if become, ok := args["_become"].(bool); ok && become {
			becomeUser, _ := args["_become_user"].(string)
			becomeMethod, _ := args["_become_method"].(string)
			exec.SetBecome(true, becomeUser, becomeMethod)
		}
		out, served, perr := exec.ProbeMany(ctx, paths, maxCaptureSize)
		_ = exec.Close()
		if served {
			return out, perr
		}
	}
	var script strings.Builder
	script.WriteString(shellProbeFunc)
	for _, p := range paths {
		script.WriteString("probe " + shellQuote(p) + "; printf '\\036\\n'\n")
	}
	return runShellOnHost(ctx, host, args, script.String())
}
