package modules

import (
	"context"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// MountFactsModule lists the mounted filesystems as ansible_facts.mount_points
// (Ansible's mount_facts): path -> {device, fstype, options}; fstypes and
// devices filter them
type MountFactsModule struct{ *BaseModule }

func NewMountFactsModule() *MountFactsModule {
	return &MountFactsModule{BaseModule: NewBaseModule("mount_facts")}
}

func (m *MountFactsModule) GetDescription() string { return "Mounted filesystems as facts" }

func parseMountFacts(out string, fstypes, devices []string) map[string]interface{} {
	keep := func(list []string, v string) bool {
		if len(list) == 0 {
			return true
		}
		for _, x := range list {
			if x == v {
				return true
			}
		}
		return false
	}
	points := map[string]interface{}{}
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		if len(f) < 4 {
			continue
		}
		device, path, fstype, opts := f[0], strings.ReplaceAll(f[1], `\040`, " "), f[2], f[3]
		if !keep(fstypes, fstype) || !keep(devices, device) {
			continue
		}
		points[path] = map[string]interface{}{"device": device, "mount": path, "fstype": fstype, "options": opts}
	}
	return points
}

func (m *MountFactsModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	start := time.Now()
	result := types.TaskResult{TaskName: taskName(args), Host: host.Name, Module: m.name, Timestamp: start, Success: true, Output: map[string]interface{}{}}
	out, err := runShellOnHost(ctx, host, args, "cat /proc/mounts 2>/dev/null || mount | awk '{print $1\" \"$3\" \"$5\" \"$6}' | tr -d '()'")
	if err != nil {
		result.Success, result.Error = false, "mount_facts: "+err.Error()
		return result, nil
	}
	points := parseMountFacts(out, listArg(args, "fstypes"), listArg(args, "devices"))
	result.Output["ansible_facts"] = map[string]interface{}{"mount_points": points}
	result.Duration = time.Since(start)
	return result, nil
}
