package modules

import (
	"context"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// ServiceFactsModule lists the services as ansible_facts.services (Ansible's
// service_facts): name -> {name, state, status, source}
type ServiceFactsModule struct{ *BaseModule }

func NewServiceFactsModule() *ServiceFactsModule {
	return &ServiceFactsModule{BaseModule: NewBaseModule("service_facts")}
}

func (m *ServiceFactsModule) GetDescription() string { return "Services and their states as facts" }

const serviceFactsScript = `if command -v systemctl >/dev/null 2>&1; then
  systemctl list-units --type=service --all --no-legend --no-pager --plain 2>/dev/null | awk '{print $1"\t"$3"\t"$4"\tsystemd"}'
  systemctl list-unit-files --type=service --no-legend --no-pager --plain 2>/dev/null | awk '{print $1"\tfile\t"$2"\tsystemd"}'
elif command -v rc-status >/dev/null 2>&1; then
  rc-status -a 2>/dev/null | awk '/\[/{gsub(/[\[\] ]/,"",$3); print $1"\t"$3"\t-\topenrc"}'
else
  for f in /etc/init.d/*; do [ -x "$f" ] || continue; n=${f##*/}; st=unknown; "$f" status >/dev/null 2>&1 && st=running || st=stopped; echo "$n\t$st\t-\tsysv"; done
fi`

func parseServiceFacts(out string) map[string]interface{} {
	services := map[string]interface{}{}
	for _, line := range strings.Split(out, "\n") {
		f := strings.Split(strings.TrimSpace(line), "\t")
		if len(f) < 4 || f[0] == "" {
			continue
		}
		name := f[0]
		entry, _ := services[name].(map[string]interface{})
		if entry == nil {
			entry = map[string]interface{}{"name": name, "source": f[3], "state": "stopped", "status": "unknown"}
			services[name] = entry
		}
		switch f[1] {
		case "file":
			entry["status"] = f[2] // enabled, disabled, static, masked, ...
		case "active":
			entry["state"] = "running"
			if f[2] == "exited" {
				entry["state"] = "stopped"
			}
		case "inactive", "failed", "stopped":
			entry["state"] = "stopped"
		default:
			if f[3] != "systemd" {
				entry["state"] = map[bool]string{true: "running", false: "stopped"}[f[1] == "started" || f[1] == "running"]
			}
		}
	}
	return services
}

func (m *ServiceFactsModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	start := time.Now()
	result := types.TaskResult{TaskName: taskName(args), Host: host.Name, Module: m.name, Timestamp: start, Success: true, Output: map[string]interface{}{}}
	out, err := runShellOnHost(ctx, host, args, serviceFactsScript)
	if err != nil {
		result.Success, result.Error = false, "service_facts: "+err.Error()
		return result, nil
	}
	services := parseServiceFacts(out)
	result.Output["ansible_facts"] = map[string]interface{}{"services": services}
	result.Output["msg"] = strings.TrimSpace(strings.Join([]string{"services:", itoa(len(services))}, " "))
	result.Duration = time.Since(start)
	return result, nil
}
