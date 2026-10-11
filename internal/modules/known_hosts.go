package modules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// KnownHostsModule keeps a host key in a known_hosts file (Ansible's
// known_hosts): name, key (the full line "name type key"), path (default
// ~/.ssh/known_hosts), state present/absent, hash_host
type KnownHostsModule struct{ *BaseModule }

func NewKnownHostsModule() *KnownHostsModule {
	return &KnownHostsModule{BaseModule: NewBaseModule("known_hosts")}
}

func (m *KnownHostsModule) GetDescription() string { return "Manage entries of a known_hosts file" }

func (m *KnownHostsModule) Validate(args map[string]interface{}) error {
	if getStringArg(args, "name", "") == "" {
		return fmt.Errorf("known_hosts requires 'name'")
	}
	if getStringArg(args, "state", "present") == "present" && getStringArg(args, "key", "") == "" {
		return fmt.Errorf("known_hosts requires 'key' with state present")
	}
	return nil
}

func (m *KnownHostsModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	start := time.Now()
	result := types.TaskResult{TaskName: taskName(args), Host: host.Name, Module: m.name, Timestamp: start, Success: true, Output: map[string]interface{}{}}
	fail := func(msg string) (types.TaskResult, error) {
		result.Success, result.Error, result.Duration = false, msg, time.Since(start)
		return result, nil
	}
	if err := m.Validate(args); err != nil {
		return fail(err.Error())
	}
	name := getStringArg(args, "name", "")
	path := getStringArg(args, "path", "~/.ssh/known_hosts")
	if strings.HasPrefix(path, "~/") {
		path = "$HOME/" + path[2:]
	}
	state := getStringArg(args, "state", "present")
	key := strings.TrimSpace(getStringArg(args, "key", ""))
	// the file's lines for the name (ssh-keygen -F knows hashed names too)
	found, _ := runShellOnHost(ctx, host, args, "ssh-keygen -F "+shellQuote(name)+" -f "+path+" 2>/dev/null | grep -v '^#' || true")
	existing := strings.TrimSpace(found)
	wantLine := key
	if key != "" && !strings.HasPrefix(key, name) && !strings.HasPrefix(key, "|1|") {
		wantLine = name + " " + key
	}
	switch state {
	case "absent":
		if existing == "" {
			result.Duration = time.Since(start)
			return result, nil
		}
		result.Changed = true
		if !inCheckMode(args) {
			if _, err := runShellOnHost(ctx, host, args, "ssh-keygen -R "+shellQuote(name)+" -f "+path+" >/dev/null 2>&1 && rm -f "+path+".old"); err != nil {
				return fail("known_hosts: " + err.Error())
			}
		}
	default:
		// same key already there (ignoring the name's hashing)?
		keyPart := strings.Join(strings.Fields(wantLine)[1:], " ")
		for _, line := range strings.Split(existing, "\n") {
			f := strings.Fields(line)
			if len(f) >= 3 && strings.Join(f[1:], " ") == keyPart {
				result.Duration = time.Since(start)
				return result, nil
			}
		}
		result.Changed = true
		if inCheckMode(args) {
			break
		}
		line := wantLine
		if getBoolArg(args, "hash_host", false) {
			// ssh-keygen -H hashes the whole file; hash only this line through a scratch file
			line = ""
		}
		script := "d=$(dirname " + path + "); mkdir -p \"$d\"; touch " + path + "; chmod 600 " + path + "; "
		if existing != "" {
			script += "ssh-keygen -R " + shellQuote(name) + " -f " + path + " >/dev/null 2>&1; rm -f " + path + ".old; "
		}
		if line != "" {
			script += "printf '%s\\n' " + shellQuote(line) + " >> " + path
		} else {
			tmp := remoteTempName("known_hosts", "line")
			script += "printf '%s\\n' " + shellQuote(wantLine) + " > " + tmp + " && ssh-keygen -H -f " + tmp + " >/dev/null 2>&1 && cat " + tmp + " >> " + path + "; rm -f " + tmp + " " + tmp + ".old"
		}
		if _, err := runShellOnHost(ctx, host, args, script); err != nil {
			return fail("known_hosts: " + err.Error())
		}
	}
	result.Duration = time.Since(start)
	return result, nil
}
