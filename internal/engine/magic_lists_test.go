package engine

import (
	"testing"
)

// the play host lists are built once per change of the hosts, and a host
// that failed leaves the batch
func TestPlayHostListsCache(t *testing.T) {
	e := &ExecutionEngine{playHosts: []string{"a", "b", "c"}, hostsVersion: 1}
	all, batch := e.playHostLists()
	if len(all) != 3 || len(batch) != 3 {
		t.Fatalf("all %v batch %v", all, batch)
	}
	again, _ := e.playHostLists()
	if &again[0] != &all[0] {
		t.Error("lists built again without a change")
	}

	e.failedHosts = map[string]bool{"b": true}
	e.hostsVersion++
	all, batch = e.playHostLists()
	if len(all) != 3 || len(batch) != 2 || batch[0] != "a" || batch[1] != "c" {
		t.Errorf("after b failed: all %v batch %v", all, batch)
	}

	vars := map[string]interface{}{}
	e.addMagicVars(vars)
	if p, _ := vars["ansible_play_hosts"].([]interface{}); len(p) != 2 {
		t.Errorf("ansible_play_hosts %v", vars["ansible_play_hosts"])
	}
}
