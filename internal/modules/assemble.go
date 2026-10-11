package modules

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// AssembleModule joins the files of a directory, in name order, into one
// file (Ansible's assemble): src (a directory on the host, or on the control
// machine with remote_src: false), dest, delimiter, regexp (only matching
// names), ignore_hidden, mode/owner/group, validate
type AssembleModule struct{ *BaseModule }

func NewAssembleModule() *AssembleModule {
	return &AssembleModule{BaseModule: NewBaseModule("assemble")}
}

func (m *AssembleModule) GetDescription() string {
	return "Join the files of a directory into one file"
}

func (m *AssembleModule) Validate(args map[string]interface{}) error {
	if getStringArg(args, "src", "") == "" || getStringArg(args, "dest", "") == "" {
		return fmt.Errorf("assemble requires 'src' and 'dest'")
	}
	return nil
}

func (m *AssembleModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	start := time.Now()
	result := types.TaskResult{TaskName: taskName(args), Host: host.Name, Module: m.name, Timestamp: start, Success: true, Output: map[string]interface{}{}}
	fail := func(msg string) (types.TaskResult, error) {
		result.Success, result.Error, result.Duration = false, msg, time.Since(start)
		return result, nil
	}
	if err := m.Validate(args); err != nil {
		return fail(err.Error())
	}
	src, dest := getStringArg(args, "src", ""), getStringArg(args, "dest", "")
	var filter *regexp.Regexp
	if re := getStringArg(args, "regexp", ""); re != "" {
		var err error
		if filter, err = regexp.Compile(re); err != nil {
			return fail("assemble: regexp: " + err.Error())
		}
	}
	// the fragments, from the host (default) or the control machine
	var names []string
	content := map[string][]byte{}
	if getBoolArg(args, "remote_src", true) {
		out, err := runShellOnHost(ctx, host, args, "ls -1A "+shellQuote(src))
		if err != nil {
			return fail("assemble: " + err.Error())
		}
		for _, n := range strings.Split(strings.TrimSpace(out), "\n") {
			if n != "" {
				names = append(names, n)
			}
		}
	} else {
		entries, err := localDirEntries(src)
		if err != nil {
			return fail("assemble: " + err.Error())
		}
		names = entries
	}
	sort.Strings(names)
	var b strings.Builder
	delimiter := getStringArg(args, "delimiter", "")
	first := true
	for _, n := range names {
		if getBoolArg(args, "ignore_hidden", false) && strings.HasPrefix(n, ".") {
			continue
		}
		if filter != nil && !filter.MatchString(n) {
			continue
		}
		var data []byte
		if getBoolArg(args, "remote_src", true) {
			d, _, err := readHostFile(ctx, host, args, strings.TrimRight(src, "/")+"/"+n)
			if err != nil {
				return fail("assemble: " + err.Error())
			}
			data = d
		} else {
			d, err := localReadFile(strings.TrimRight(src, "/") + "/" + n)
			if err != nil {
				return fail("assemble: " + err.Error())
			}
			data = d
		}
		content[n] = data
		if !first && delimiter != "" {
			b.WriteString(delimiter)
			if !strings.HasSuffix(delimiter, "\n") {
				b.WriteString("\n")
			}
		}
		b.Write(data)
		if len(data) > 0 && data[len(data)-1] != '\n' {
			b.WriteString("\n")
		}
		first = false
	}
	assembled := []byte(b.String())
	current, exists, err := readHostFile(ctx, host, args, dest)
	if err != nil {
		return fail("assemble: " + err.Error())
	}
	if exists && string(current) == string(assembled) {
		result.Duration = time.Since(start)
		return result, nil
	}
	result.Changed = true
	if inCheckMode(args) {
		result.Output["msg"] = "would assemble " + dest
		result.Duration = time.Since(start)
		return result, nil
	}
	if v := getStringArg(args, "validate", ""); v != "" {
		tmp := remoteTempName("assemble", "fragment")
		if err := writeHostFile(ctx, host, args, tmp, assembled, 0o600); err != nil {
			return fail(err.Error())
		}
		if _, err := runShellOnHost(ctx, host, args, strings.ReplaceAll(v, "%s", shellQuote(tmp))+"; rc=$?; rm -f "+shellQuote(tmp)+"; exit $rc"); err != nil {
			return fail("assemble: validate failed: " + err.Error())
		}
	}
	mode := modeArg(args, 0o644)
	if err := writeHostFile(ctx, host, args, dest, assembled, mode); err != nil {
		return fail(err.Error())
	}
	if owner, group := getStringArg(args, "owner", ""), getStringArg(args, "group", ""); owner != "" || group != "" {
		if _, err := runOnHost(ctx, host, args, "chown", strings.Trim(owner+":"+group, ":"), dest); err != nil {
			return fail("assemble: chown: " + err.Error())
		}
	}
	result.Output["dest"] = dest
	result.Duration = time.Since(start)
	return result, nil
}
