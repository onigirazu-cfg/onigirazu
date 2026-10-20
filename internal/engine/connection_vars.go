package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/onigirazu-cfg/onigirazu/internal/logger"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// resolveConnection freezes a host's connection settings before the play connects.
func (e *ExecutionEngine) resolveConnection(ctx context.Context, host types.Host, playVars map[string]interface{}) (types.Host, error) {
	vars := e.layerHostVariables(&host, playVars, false)
	// as for tasks: ansible_user: "{{ login }}" with login: "{{ deploy_user }}"
	e.renderLazyVars(ctx, host.Name, vars)
	resolve := func(secret bool, names ...string) (string, string, bool, error) {
		// -e wins whichever alias it is given under: -e ansible_user=x over
		// an inventory onigirazu_user
		pick := func() (string, interface{}) {
			for _, layer := range []map[string]interface{}{e.extraVars, vars} {
				for _, name := range names {
					if v, ok := layer[name]; ok && v != nil {
						return name, v
					}
				}
			}
			return "", nil
		}
		if name, v := pick(); name != "" {
			value := fmt.Sprint(v)
			if strings.Contains(value, "{{") || strings.Contains(value, "{%") {
				rendered, err := e.templateEngine.Render(ctx, value, vars)
				if err != nil {
					if secret {
						return "", name, true, fmt.Errorf("host %s: %s: template rendering failed", host.Name, name)
					}
					return "", name, true, fmt.Errorf("host %s: %s: %w", host.Name, name, err)
				}
				value = strings.TrimSpace(rendered)
			}
			return value, name, true, nil
		}
		return "", "", false, nil
	}
	fields := []struct {
		target *string
		secret bool
		names  []string
	}{
		{&host.Address, false, []string{"onigirazu_host", "ansible_host"}},
		{&host.User, false, []string{"onigirazu_user", "ansible_user", "ansible_ssh_user"}},
		{&host.Password, true, []string{"onigirazu_password", "ansible_password", "ansible_ssh_pass"}},
		{&host.KeyFile, false, []string{"onigirazu_ssh_private_key_file", "ansible_ssh_private_key_file", "ansible_private_key_file"}},
		{&host.BecomePassword, true, []string{"onigirazu_become_password", "ansible_become_password", "ansible_become_pass", "ansible_sudo_pass"}},
	}
	for _, field := range fields {
		value, name, present, err := resolve(field.secret, field.names...)
		if err != nil {
			return host, err
		}
		if present {
			if name == "onigirazu_ssh_private_key_file" && strings.HasPrefix(value, "~/") {
				if home, err := os.UserHomeDir(); err == nil {
					value = filepath.Join(home, value[2:])
				}
			}
			*field.target = value
			if field.secret {
				logger.Redact(value)
			}
		}
	}
	value, name, present, err := resolve(false, "onigirazu_port", "ansible_port", "ansible_ssh_port")
	if err != nil {
		return host, err
	}
	if present {
		port, err := strconv.Atoi(value)
		if err != nil || port <= 0 {
			return host, fmt.Errorf("host %s: %s %q is not a port", host.Name, name, value)
		}
		host.Port = port
	}
	var args []string
	argsPresent := false
	rawInsecure := strings.Contains(host.SSHArgs, "StrictHostKeyChecking=no")
	for _, name := range []string{"ansible_ssh_common_args", "ansible_ssh_extra_args"} {
		if raw, ok := vars[name]; ok {
			rawInsecure = rawInsecure || strings.Contains(fmt.Sprint(raw), "StrictHostKeyChecking=no")
		}
		value, _, present, err := resolve(false, name)
		if err != nil {
			return host, err
		}
		argsPresent = argsPresent || present
		if trimmed := strings.TrimSpace(value); present && trimmed != "" {
			args = append(args, trimmed)
		}
	}
	if argsPresent {
		host.SSHArgs = strings.Join(args, " ")
		explicitInsecure := false
		for _, name := range []string{"insecure_ignore_host_key", "onigirazu_insecure_ignore_host_key", "ansible_ssh_host_key_checking"} {
			value, _, present, err := resolve(false, name)
			if err != nil {
				return host, err
			}
			if present {
				switch strings.ToLower(value) {
				case "true", "yes", "1":
					explicitInsecure = explicitInsecure || name != "ansible_ssh_host_key_checking"
				case "false", "no", "0":
					explicitInsecure = explicitInsecure || name == "ansible_ssh_host_key_checking"
				}
			}
		}
		if strings.Contains(host.SSHArgs, "StrictHostKeyChecking=no") || explicitInsecure {
			host.InsecureIgnoreHostKey = true
		} else if rawInsecure {
			host.InsecureIgnoreHostKey = false
		}
	}
	return host, nil
}
