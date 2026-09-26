package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestBuildDriftReport(t *testing.T) {
	result := &types.PlaybookResult{Plays: []types.PlayResult{{Name: "p", Hosts: []types.HostResult{
		{Host: "web1", Tasks: []types.TaskResult{
			{TaskName: "conf", Module: "copy", Changed: true, Output: map[string]interface{}{"msg": "file would be updated"}},
			{TaskName: "pkg", Module: "apt"},
		}},
		{Host: "web2", Tasks: []types.TaskResult{{TaskName: "conf", Module: "copy"}}},
		{Host: "db", Tasks: []types.TaskResult{{TaskName: "probe", Failed: true, Error: "unreachable"}}},
	}}}}
	r := buildDriftReport("site.yml", result)
	assert.Equal(t, 3, r.Hosts)
	assert.Equal(t, []string{"web2"}, r.InSync)
	assert.Equal(t, []DriftItem{{Play: "p", Task: "conf", Module: "copy", Detail: "file would be updated"}}, r.Drift["web1"])
	assert.Equal(t, "unreachable", r.Errors["db"][0].Detail)

	var out bytes.Buffer
	writeDriftText(&out, r)
	assert.Contains(t, out.String(), "Drift: 1 task(s) on 1 of 3 host(s)")
	assert.Contains(t, out.String(), "~ conf (copy): file would be updated")
	assert.Contains(t, out.String(), "! db: probe: unreachable")
}

func TestDriftApplyArgs(t *testing.T) {
	o := driftCheckOptions{limit: "web", extraVars: []string{"a=1"}, become: true}
	args := o.applyArgs("site.yml", true)
	assert.Equal(t, []string{"site.yml", "--check", "--diff", "-e", "a=1"}, args[:5])
	assert.Contains(t, args, "--limit")
	assert.Contains(t, args, "--become")
	assert.NotContains(t, o.applyArgs("site.yml", false), "--check")
}

func TestPlanWording(t *testing.T) {
	r := &DriftReport{Playbook: "site.yml", Hosts: 2, Plan: true, Drift: map[string][]DriftItem{"web1": {{Task: "conf"}}}, DriftTasks: 1}
	var out bytes.Buffer
	writeDriftText(&out, r)
	assert.Contains(t, out.String(), "Plan: 1 task(s) would change 1 of 2 host(s)")
	out.Reset()
	writeDriftText(&out, &DriftReport{Playbook: "site.yml", Hosts: 2, Plan: true})
	assert.Contains(t, out.String(), "No changes: 2 host(s) already match site.yml")
}
