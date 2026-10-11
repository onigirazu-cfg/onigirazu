package modules

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	sshpkg "github.com/onigirazu-cfg/onigirazu/internal/ssh"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// RouterosCommandModule runs CLI commands over SSH (community.routeros.command)
type RouterosCommandModule struct{ *BaseModule }

func NewRouterosCommandModule() *RouterosCommandModule {
	return &RouterosCommandModule{BaseModule: NewBaseModule("routeros_command")}
}

func (m *RouterosCommandModule) GetDescription() string { return "Run RouterOS CLI commands over SSH" }

func (m *RouterosCommandModule) Validate(args map[string]interface{}) error {
	if len(listArg(args, "commands")) == 0 {
		return fmt.Errorf("routeros_command requires 'commands'")
	}
	return nil
}

// routerosSSHDial opens the CLI session's connection
var routerosSSHDial = func(host types.Host) (*ssh.Client, error) {
	address := host.Address
	if address == "" {
		address = host.Name
	}
	port := host.Port
	if port == 0 {
		port = 22
	}
	if host.User == "" {
		return nil, fmt.Errorf("routeros_command: ansible_user is required")
	}
	// the device's key is checked against known_hosts like any SSH host's;
	// insecure_ignore_host_key / StrictHostKeyChecking=no turn it off per host
	cfg := &ssh.ClientConfig{User: host.User + "+ct", Auth: []ssh.AuthMethod{ssh.Password(host.Password)}, Timeout: 20 * time.Second,
		HostKeyCallback: sshpkg.HostKeyCallbackFor(host)}
	return ssh.Dial("tcp", net.JoinHostPort(address, strconv.Itoa(port)), cfg)
}

func (m *RouterosCommandModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	start := time.Now()
	result := types.TaskResult{TaskName: taskName(args), Host: host.Name, Module: m.name, Timestamp: start, Success: true, Output: map[string]interface{}{}}
	if err := m.Validate(args); err != nil {
		result.Success, result.Error = false, err.Error()
		return result, nil
	}
	client, err := routerosSSHDial(host)
	if err != nil {
		result.Success, result.Error = false, err.Error()
		return result, nil
	}
	defer client.Close()
	var outputs []string
	for _, cmd := range listArg(args, "commands") {
		sess, err := client.NewSession()
		if err != nil {
			result.Success, result.Error = false, err.Error()
			return result, nil
		}
		out, err := sess.CombinedOutput(cmd)
		_ = sess.Close()
		text := strings.TrimSpace(strings.ReplaceAll(string(out), "\r", ""))
		if err != nil {
			result.Success, result.Error = false, fmt.Sprintf("%s: %v: %s", cmd, err, text)
			return result, nil
		}
		outputs = append(outputs, text)
	}
	result.Output["stdout"] = outputs
	lines := make([][]string, len(outputs))
	for i, o := range outputs {
		lines[i] = splitLines(o)
	}
	result.Output["stdout_lines"] = lines
	// commands that print are not changes: the caller says with changed_when
	result.Duration = time.Since(start)
	return result, nil
}

func splitLines(s string) []string {
	if s == "" {
		return []string{}
	}
	return strings.Split(s, "\n")
}
