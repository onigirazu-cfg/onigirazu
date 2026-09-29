package modules

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	sshpkg "github.com/onigirazu-cfg/onigirazu/internal/ssh"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// UnarchiveModule extracts a tar or zip archive on the host (Ansible's
// unarchive). The archive comes from the control machine, or from the host
// with remote_src. It is extracted when a file of the archive is missing in
// dest; creates makes the check a single path.
type UnarchiveModule struct {
	*BaseModule
}

// NewUnarchiveModule creates a new unarchive module
func NewUnarchiveModule() *UnarchiveModule {
	return &UnarchiveModule{BaseModule: NewBaseModule("unarchive")}
}

func (m *UnarchiveModule) GetDescription() string {
	return "Extract a tar or zip archive on the host"
}

func (m *UnarchiveModule) Validate(args map[string]interface{}) error {
	if err := m.BaseModule.Validate(args); err != nil {
		return err
	}
	for _, key := range []string{"src", "dest"} {
		if v, _ := args[key].(string); v == "" {
			return fmt.Errorf("unarchive module requires '%s' parameter", key)
		}
	}
	return nil
}

func (m *UnarchiveModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
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
	done := func() (types.TaskResult, error) {
		result.Duration = time.Since(start)
		return result, nil
	}

	src, _ := args["src"].(string)
	dest, _ := args["dest"].(string)
	remoteSrc := getBoolArg(args, "remote_src", false)
	owner := getStringArg(args, "owner", "")
	group := getStringArg(args, "group", "")
	result.Output["src"] = src
	result.Output["dest"] = dest

	if creates := getStringArg(args, "creates", ""); creates != "" {
		if _, err := runOnHost(ctx, host, args, "test", "-e", creates); err == nil {
			result.Output["msg"] = creates + " exists, skipped"
			return done()
		}
	}
	if _, err := runOnHost(ctx, host, args, "test", "-d", dest); err != nil {
		return fail(fmt.Sprintf("destination %s is not a directory", dest))
	}

	zip := strings.HasSuffix(strings.ToLower(src), ".zip")
	tool := "tar"
	if zip {
		tool = "unzip"
	}
	if _, err := runShellOnHost(ctx, host, args, "command -v "+tool); err != nil {
		return fail(fmt.Sprintf("%s is not installed on %s; it is needed to extract %s", tool, host.Name, src))
	}
	var archive string
	if !remoteSrc {
		uploaded, cleanup, err := uploadArchive(ctx, host, args, src)
		if err != nil {
			return fail(err.Error())
		}
		defer cleanup()
		archive = uploaded
	} else {
		abs, err := runOnHost(ctx, host, args, "readlink", "-f", src)
		if _, statErr := runOnHost(ctx, host, args, "test", "-f", src); err != nil || statErr != nil {
			return fail(fmt.Sprintf("source %s not found on %s", src, host.Name))
		}
		archive = strings.TrimSpace(abs)
	}

	qa := shellQuote(archive)
	// stderr carries warnings (unknown extended headers), not members
	list := "tar -tf " + qa + " 2>/dev/null"
	if zip {
		list = "unzip -Z1 " + qa + " 2>/dev/null"
	}
	out, err := runShellOnHost(ctx, host, args, list)
	if err != nil {
		return fail(fmt.Sprintf("failed to read the archive %s: %v", src, err))
	}
	members := archiveMembers(out)
	if getBoolArg(args, "list_files", false) {
		result.Output["files"] = members
	}

	missing, err := missingMembers(ctx, host, args, dest, list)
	if err != nil {
		return fail(err.Error())
	}
	if missing == 0 {
		result.Output["msg"] = "all files of the archive exist in " + dest
		return done()
	}
	result.Changed = true
	if inCheckMode(args) {
		result.Output["msg"] = fmt.Sprintf("would extract %s (%d file(s) missing in %s)", src, missing, dest)
		return done()
	}

	extract := fmt.Sprintf("tar -xf %s -C %s", qa, shellQuote(dest))
	if zip {
		extract = fmt.Sprintf("unzip -o -q %s -d %s", qa, shellQuote(dest))
	}
	for _, opt := range stringList(args["extra_opts"]) {
		extract += " " + shellQuote(opt)
	}
	if _, err := runShellOnHost(ctx, host, args, extract); err != nil {
		return fail(fmt.Sprintf("failed to extract %s: %v", src, err))
	}
	if owner != "" || group != "" {
		spec := owner
		if group != "" {
			spec += ":" + group
		}
		for _, top := range topLevel(members) {
			if _, err := runOnHost(ctx, host, args, "chown", "-R", spec, filepath.Join(dest, top)); err != nil {
				return fail(fmt.Sprintf("failed to set the owner of %s: %v", top, err))
			}
		}
	}
	result.Output["msg"] = fmt.Sprintf("extracted %s into %s", src, dest)
	return done()
}

// uploadArchive puts a local archive into a temporary file on the host
func uploadArchive(ctx context.Context, host types.Host, args map[string]interface{}, src string) (string, func(), error) {
	data, err := os.ReadFile(src) // #nosec G304 -- src is the task's archive
	if err != nil {
		return "", nil, fmt.Errorf("failed to read %s: %w", src, err)
	}
	if sshpkg.IsLocal(host) {
		abs, err := filepath.Abs(src)
		return abs, func() {}, err
	}
	tmp := remoteTempName(".onigirazu-unarchive-", filepath.Base(src))
	if inContainer(host) {
		if err := putContainerFile(ctx, host, tmp, data, 0600); err != nil {
			return "", nil, err
		}
	} else {
		pool := sshpkg.GetGlobalPool()
		client, err := pool.GetConnection(host)
		if err != nil {
			return "", nil, fmt.Errorf("failed to get SSH connection: %w", err)
		}
		defer pool.ReleaseConnection(host)
		if err := client.WriteFile(tmp, data, 0600); err != nil {
			return "", nil, fmt.Errorf("failed to upload %s: %w", src, err)
		}
	}
	// with become the archive must be readable for the other user
	if _, err := runShellOnHost(ctx, host, map[string]interface{}{}, "chmod 0644 "+shellQuote(tmp)); err != nil {
		return "", nil, err
	}
	return tmp, func() {
		_, _ = runShellOnHost(context.Background(), host, map[string]interface{}{}, "rm -f "+shellQuote(tmp))
	}, nil
}

// archiveMembers are the paths of an archive listing, without "./"
func archiveMembers(listing string) []string {
	var members []string
	for _, line := range strings.Split(listing, "\n") {
		line = strings.TrimPrefix(strings.TrimSpace(line), "./")
		line = strings.TrimSuffix(line, "/")
		if line != "" && line != "." {
			members = append(members, line)
		}
	}
	return members
}

// topLevel are the first path components of the members
func topLevel(members []string) []string {
	seen := map[string]bool{}
	for _, m := range members {
		first, _, _ := strings.Cut(m, "/")
		seen[first] = true
	}
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// missingMembers counts the members of the archive (listed by list) that
// do not exist in dest
func missingMembers(ctx context.Context, host types.Host, args map[string]interface{}, dest, list string) (int, error) {
	script := fmt.Sprintf(`set -e
cd %s
%s | sed -e 's#^\./##' -e 's#/$##' | {
  n=0
  while IFS= read -r f; do
    [ -z "$f" ] || [ "$f" = . ] || [ -e "$f" ] || [ -L "$f" ] || n=$((n+1))
  done
  echo $n
}`, shellQuote(dest), list)
	out, err := runShellOnHost(ctx, host, args, script)
	if err != nil {
		return 0, fmt.Errorf("failed to compare the archive with %s: %w", dest, err)
	}
	var n int
	if _, err := fmt.Sscan(strings.TrimSpace(out), &n); err != nil {
		return 0, fmt.Errorf("unexpected output comparing the archive: %q", out)
	}
	return n, nil
}

// stringList reads a string or a list of strings
func stringList(v interface{}) []string {
	switch t := v.(type) {
	case string:
		if t != "" {
			return []string{t}
		}
	case []interface{}:
		var out []string
		for _, item := range t {
			out = append(out, fmt.Sprint(item))
		}
		return out
	}
	return nil
}
