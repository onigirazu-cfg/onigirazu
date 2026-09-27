package modules

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// zoneName is an IANA name such as Europe/Madrid or Etc/GMT+3
var zoneName = regexp.MustCompile(`^[A-Za-z0-9_+-]+(/[A-Za-z0-9_+-]+)*$`)

// TimezoneModule sets the system time zone (Ansible's timezone module)
type TimezoneModule struct {
	*BaseModule
}

// NewTimezoneModule creates a new timezone module
func NewTimezoneModule() *TimezoneModule {
	return &TimezoneModule{BaseModule: NewBaseModule("timezone")}
}

func (m *TimezoneModule) GetDescription() string {
	return "Set the system time zone"
}

func (m *TimezoneModule) Validate(args map[string]interface{}) error {
	if err := m.BaseModule.Validate(args); err != nil {
		return err
	}
	name, _ := args["name"].(string)
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("timezone module requires 'name' parameter")
	}
	if !zoneName.MatchString(strings.TrimSpace(name)) {
		return fmt.Errorf("timezone: %q is not a time zone name", name)
	}
	return nil
}

func (m *TimezoneModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	start := time.Now()
	result := types.TaskResult{
		TaskName: taskName(args), Host: host.Name, Module: m.name,
		Timestamp: start, Success: true, Output: map[string]interface{}{},
	}
	fail := func(msg string) (types.TaskResult, error) {
		result.Success = false
		result.Error = msg
		result.Duration = time.Since(start)
		return result, nil
	}
	name := strings.TrimSpace(fmt.Sprint(args["name"]))

	current, err := runShellOnHost(ctx, host, args,
		`timedatectl show -p Timezone --value 2>/dev/null || { [ -L /etc/localtime ] && readlink /etc/localtime | sed 's#^.*/zoneinfo/##'; } || cat /etc/timezone 2>/dev/null`)
	if err != nil {
		return fail(fmt.Sprintf("failed to read the time zone: %v", err))
	}
	current = strings.TrimSpace(current)
	result.Output["name"] = name
	result.Output["previous"] = current
	if current == name {
		result.Output["msg"] = "time zone is " + name
		result.Duration = time.Since(start)
		return result, nil
	}
	if _, err := runOnHost(ctx, host, args, "test", "-f", "/usr/share/zoneinfo/"+name); err != nil {
		return fail(fmt.Sprintf("unknown time zone %q (no /usr/share/zoneinfo/%s)", name, name))
	}
	result.Changed = true
	if inCheckMode(args) {
		result.Output["msg"] = fmt.Sprintf("would change the time zone from %s to %s", current, name)
		result.Duration = time.Since(start)
		return result, nil
	}
	q := shellQuote(name)
	script := fmt.Sprintf(`if command -v timedatectl >/dev/null 2>&1 && timedatectl set-timezone %s 2>/dev/null; then :; else
  ln -sf %s /etc/localtime
  if [ -e /etc/timezone ]; then echo %s > /etc/timezone; fi
fi`, q, shellQuote("/usr/share/zoneinfo/"+name), q)
	if _, err := runShellOnHost(ctx, host, args, script); err != nil {
		return fail(fmt.Sprintf("failed to set the time zone: %v", err))
	}
	result.Output["msg"] = fmt.Sprintf("time zone changed from %s to %s", current, name)
	result.Duration = time.Since(start)
	return result, nil
}
