package modules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// WinRebootModule reboots a Windows host and waits until it is back
type WinRebootModule struct {
	*BaseModule
	// poll is how often the host is asked whether it is back
	poll time.Duration
}

// NewWinRebootModule creates win_reboot
func NewWinRebootModule() *WinRebootModule {
	return &WinRebootModule{BaseModule: NewBaseModule("win_reboot"), poll: 5 * time.Second}
}

func (m *WinRebootModule) GetDescription() string {
	return "Reboot a Windows host and wait until it answers again"
}

const bootTimeScript = `(Get-CimInstance Win32_OperatingSystem).LastBootUpTime.ToUniversalTime().ToString('o')`

func (m *WinRebootModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	start := time.Now()
	res := types.TaskResult{TaskName: taskName(args), Host: host.Name, Module: m.name, Timestamp: start,
		Output: map[string]interface{}{"rebooted": false}}
	fail := func(format string, a ...interface{}) (types.TaskResult, error) {
		res.Failed, res.Error = true, fmt.Sprintf(format, a...)
		res.Duration = time.Since(start)
		return res, nil
	}
	seconds := func(n, def int) time.Duration {
		if n < 0 {
			n = def
		}
		return time.Duration(n) * time.Second
	}
	timeout := seconds(getIntArg(args, "reboot_timeout", 600), 600)
	preDelay := seconds(getIntArg(args, "pre_reboot_delay", 2), 2)
	postDelay := seconds(getIntArg(args, "post_reboot_delay", 0), 0)
	msg := getStringArg(args, "msg", "Reboot initiated by onigirazu")
	test := getStringArg(args, "test_command", "whoami")

	if inCheckMode(args) {
		res.Success, res.Changed = true, true
		res.Output["msg"] = "would reboot"
		return res, nil
	}
	c, err := winClient(host, args)
	if err != nil {
		return fail("%v", err)
	}
	before, err := c.RunPS(ctx, bootTimeScript)
	if err != nil || before.ExitCode != 0 {
		return fail("reading the boot time: %v %s", err, strings.TrimSpace(before.Stderr))
	}
	boot := strings.TrimSpace(before.Stdout)
	// shutdown.exe returns at once; the connection drops a moment later
	cmd := fmt.Sprintf("shutdown.exe /r /t %d /c %s", int(preDelay.Seconds()), psQuote(msg))
	if out, err := c.RunPS(ctx, cmd); err != nil || out.ExitCode != 0 {
		// 1190: a shutdown is already scheduled
		if !strings.Contains(out.Stdout+out.Stderr, "1190") {
			return fail("reboot: %v %s", err, strings.TrimSpace(out.Stderr+out.Stdout))
		}
	}
	res.Changed = true

	deadline := time.Now().Add(timeout + preDelay)
	for {
		select {
		case <-ctx.Done():
			return fail("canceled while waiting for %s", host.Name)
		case <-time.After(m.poll):
		}
		if time.Now().After(deadline) {
			return fail("%s did not come back within %s", host.Name, timeout)
		}
		now, err := c.RunPS(ctx, bootTimeScript)
		if err != nil || now.ExitCode != 0 || strings.TrimSpace(now.Stdout) == boot || strings.TrimSpace(now.Stdout) == "" {
			continue // still going down, down, or not answering yet
		}
		break
	}
	if postDelay > 0 {
		select {
		case <-ctx.Done():
			return fail("canceled after the reboot of %s", host.Name)
		case <-time.After(postDelay):
		}
	}
	// the host is up; the test command says it is usable
	for {
		out, err := c.RunPS(ctx, test)
		if err == nil && out.ExitCode == 0 {
			break
		}
		if time.Now().After(deadline) {
			return fail("test command %q still fails after the reboot: %v %s", test, err, strings.TrimSpace(out.Stderr))
		}
		select {
		case <-ctx.Done():
			return fail("canceled while testing %s", host.Name)
		case <-time.After(m.poll):
		}
	}
	res.Success = true
	res.Output["rebooted"] = true
	res.Output["elapsed"] = int(time.Since(start).Seconds())
	res.Duration = time.Since(start)
	return res, nil
}
