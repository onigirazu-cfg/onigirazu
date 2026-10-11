package modules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// SubversionModule checks out or updates a Subversion working copy
// (Ansible's subversion): repo, dest, revision (HEAD), force, username,
// password, export, switch, update, checkout, in_place
type SubversionModule struct{ *BaseModule }

func NewSubversionModule() *SubversionModule {
	return &SubversionModule{BaseModule: NewBaseModule("subversion")}
}

func (m *SubversionModule) GetDescription() string {
	return "Check out or update a Subversion working copy"
}

func (m *SubversionModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	start := time.Now()
	result := types.TaskResult{TaskName: taskName(args), Host: host.Name, Module: m.name, Timestamp: start, Success: true, Output: map[string]interface{}{}}
	fail := func(msg string) (types.TaskResult, error) {
		result.Success, result.Error, result.Duration = false, msg, time.Since(start)
		return result, nil
	}
	repo, dest := getStringArg(args, "repo", ""), getStringArg(args, "dest", "")
	if repo == "" || dest == "" {
		return fail("subversion requires 'repo' and 'dest'")
	}
	svn := getStringArg(args, "executable", "svn") + " --non-interactive --trust-server-cert"
	if u := getStringArg(args, "username", ""); u != "" {
		svn += " --username " + shellQuote(u)
	}
	if p := getStringArg(args, "password", ""); p != "" {
		svn += " --password " + shellQuote(p)
	}
	rev := getStringArg(args, "revision", "HEAD")
	info, _ := runShellOnHost(ctx, host, args, svn+" info "+shellQuote(dest)+" 2>/dev/null")
	var before, url string
	for _, line := range strings.Split(info, "\n") {
		if v, ok := strings.CutPrefix(line, "Revision: "); ok {
			before = strings.TrimSpace(v)
		}
		if v, ok := strings.CutPrefix(line, "URL: "); ok {
			url = strings.TrimSpace(v)
		}
	}
	result.Output["before"] = before
	if inCheckMode(args) {
		result.Changed = before == "" || url != repo
		result.Duration = time.Since(start)
		return result, nil
	}
	var cmd string
	switch {
	case getBoolArg(args, "export", false):
		cmd = svn + " export --force -r " + shellQuote(rev) + " " + shellQuote(repo) + " " + shellQuote(dest)
	case before == "":
		if !getBoolArg(args, "checkout", true) {
			return fail("subversion: " + dest + " is no working copy and checkout is false")
		}
		cmd = svn + " checkout -r " + shellQuote(rev) + " " + shellQuote(repo) + " " + shellQuote(dest)
		if getBoolArg(args, "in_place", false) {
			cmd = svn + " checkout --force -r " + shellQuote(rev) + " " + shellQuote(repo) + " " + shellQuote(dest)
		}
	case url != repo && getBoolArg(args, "switch", true):
		cmd = svn + " switch --force -r " + shellQuote(rev) + " " + shellQuote(repo) + " " + shellQuote(dest)
	case getBoolArg(args, "update", true):
		cmd = svn + " update -r " + shellQuote(rev) + " " + shellQuote(dest)
		if getBoolArg(args, "force", false) {
			cmd = svn + " revert -R " + shellQuote(dest) + " >/dev/null && " + cmd
		}
	default:
		result.Duration = time.Since(start)
		return result, nil
	}
	if out, err := runShellOnHost(ctx, host, args, cmd+" 2>&1"); err != nil {
		return fail(fmt.Sprintf("subversion: %v: %s", err, strings.TrimSpace(out)))
	}
	info, _ = runShellOnHost(ctx, host, args, svn+" info "+shellQuote(dest)+" 2>/dev/null | awk '/^Revision:/{print $2}'")
	after := strings.TrimSpace(info)
	result.Output["after"] = after
	result.Changed = before != after || getBoolArg(args, "export", false)
	result.Duration = time.Since(start)
	return result, nil
}
