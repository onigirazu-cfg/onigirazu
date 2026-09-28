package modules

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// hostnamePattern is an RFC 1123 host name (labels of letters, digits, -)
var hostnamePattern = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?(\.[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?)*$`)

// HostnameModule sets the host name (Ansible's hostname)
type HostnameModule struct {
	*BaseModule
}

// NewHostnameModule creates the hostname module
func NewHostnameModule() *HostnameModule {
	return &HostnameModule{BaseModule: NewBaseModule("hostname")}
}

func (m *HostnameModule) GetDescription() string { return "Set the host name" }

func (m *HostnameModule) Validate(args map[string]interface{}) error {
	name := getStringArg(args, "name", "")
	if name == "" {
		return fmt.Errorf("hostname requires 'name'")
	}
	if !hostnamePattern.MatchString(name) {
		return fmt.Errorf("hostname: %q is not a valid host name", name)
	}
	return nil
}

func (m *HostnameModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	start := time.Now()
	result := types.TaskResult{TaskName: taskName(args), Host: host.Name, Module: m.name,
		Timestamp: start, Success: true, Output: map[string]interface{}{}}
	fail := func(msg string) (types.TaskResult, error) {
		result.Success, result.Error, result.Duration = false, msg, time.Since(start)
		return result, nil
	}
	if err := m.Validate(args); err != nil {
		return fail(err.Error())
	}
	name := getStringArg(args, "name", "")
	current, err := runShellOnHost(ctx, host, args, "hostnamectl --static 2>/dev/null || cat /etc/hostname 2>/dev/null || hostname")
	if err != nil {
		return fail(fmt.Sprintf("failed to read the host name: %v", err))
	}
	current = strings.TrimSpace(current)
	result.Output["name"] = name
	result.Output["ansible_facts"] = map[string]interface{}{"ansible_hostname": strings.SplitN(name, ".", 2)[0]}
	if current == name {
		result.Duration = time.Since(start)
		return result, nil
	}
	result.Changed = true
	addDiff(args, &result, "hostname", current+"\n", name+"\n")
	if inCheckMode(args) {
		result.Duration = time.Since(start)
		return result, nil
	}
	q := shellQuote(name)
	if _, err := runShellOnHost(ctx, host, args, fmt.Sprintf(
		"if command -v hostnamectl >/dev/null 2>&1 && hostnamectl set-hostname %s 2>/dev/null; then :; else printf '%%s\\n' %s > /etc/hostname && hostname %s; fi", q, q, q)); err != nil {
		return fail(fmt.Sprintf("failed to set the host name: %v", err))
	}
	result.Duration = time.Since(start)
	return result, nil
}
