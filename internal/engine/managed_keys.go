package engine

import (
	"fmt"
	"strings"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// Task keys name a task across runs for the managed state registry:
// "play:web/role:nginx/tasks/Install nginx". The engine assigns them when it
// enters a scope (a play, a role, a block), so tasks of roles loaded at run
// time get theirs too. Every assigned key is recorded; a scope that was
// skipped as a whole (a role or include_role whose when is false) is
// recorded as kept: the registry must not drop what its tasks manage.

// ManagedScopes is what the run saw of the playbook
type ManagedScopes struct {
	// Keys are the keys of every task the run entered
	Keys map[string]bool
	// Kept are scopes the run skipped whole; keys under them stay
	Kept []string
	// KeptFor are scopes some hosts skipped: scope -> those hosts
	KeptFor map[string][]string
	// Hosts are the hosts the plays matched; AllPlays: every play got
	// that far (the run was not cut short)
	Hosts    map[string]bool
	AllPlays bool
}

// ManagedScopes returns the task keys the last run assigned
func (e *ExecutionEngine) ManagedScopes() ManagedScopes {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	keys := make(map[string]bool, len(e.taskKeys))
	for k := range e.taskKeys {
		keys[k] = true
	}
	hosts := make(map[string]bool, len(e.targetedHosts))
	for h := range e.targetedHosts {
		hosts[h] = true
	}
	keptFor := make(map[string][]string, len(e.keptFor))
	for k, v := range e.keptFor {
		keptFor[k] = append([]string(nil), v...)
	}
	return ManagedScopes{Keys: keys, Kept: append([]string(nil), e.keptScopes...), KeptFor: keptFor, Hosts: hosts,
		AllPlays: e.playsTotal > 0 && e.playsTargeted == e.playsTotal}
}

// playScopes names every play: by name, with its position when the name is
// empty or repeated
func playScopes(plays []types.Play) []string {
	count := map[string]int{}
	for _, p := range plays {
		count[p.Name]++
	}
	scopes := make([]string, len(plays))
	for i, p := range plays {
		if p.Name == "" || count[p.Name] > 1 {
			scopes[i] = fmt.Sprintf("play:#%d:%s", i+1, p.Name)
		} else {
			scopes[i] = "play:" + p.Name
		}
	}
	return scopes
}

// assignKeys gives every task of the list (and of its blocks) a key under
// scope: its name, or module#n for unnamed tasks; a repeated name gets #n
func (e *ExecutionEngine) assignKeys(tasks []types.Task, scope string) {
	seen := map[string]int{}
	for i := range tasks {
		t := &tasks[i]
		base := strings.TrimSpace(t.Name)
		if base == "" {
			base = t.Module
			if len(t.Block) > 0 {
				base = "block"
			}
			base += "#"
		}
		seen[base]++
		if n := seen[base]; n > 1 || strings.HasSuffix(base, "#") {
			base = fmt.Sprintf("%s#%d", strings.TrimSuffix(base, "#"), n)
		}
		t.Key = scope + "/" + base
		e.recordKey(t.Key)
		if len(t.Block) > 0 {
			e.assignKeys(t.Block, t.Key+"/block")
			e.assignKeys(t.Rescue, t.Key+"/rescue")
			e.assignKeys(t.Always, t.Key+"/always")
		}
	}
}

// assignRoleKeys keys a role's task lists under parent
func (e *ExecutionEngine) assignRoleKeys(role *types.Role, parent string) {
	scope := parent + "/role:" + role.Name
	e.assignKeys(role.PreTasks, scope+"/pre_tasks")
	e.assignKeys(role.Tasks, scope+"/tasks")
	e.assignKeys(role.PostTasks, scope+"/post_tasks")
	e.assignKeys(role.Handlers, scope+"/handlers")
}

func (e *ExecutionEngine) recordKey(key string) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	if e.taskKeys == nil {
		e.taskKeys = map[string]bool{}
	}
	e.taskKeys[key] = true
}

// keepScopeFor records a scope one host skipped
func (e *ExecutionEngine) keepScopeFor(scope, host string) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	if e.keptFor == nil {
		e.keptFor = map[string][]string{}
	}
	e.keptFor[scope] = append(e.keptFor[scope], host)
}

// keepScope records a scope the run skipped whole
func (e *ExecutionEngine) keepScope(scope string) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.keptScopes = append(e.keptScopes, scope)
}
