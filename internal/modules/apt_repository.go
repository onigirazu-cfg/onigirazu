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

// AptRepositoryModule adds or removes an APT source line (Ansible's
// apt_repository). A "deb ..." line is searched in every .list file; a new
// one goes to sources.list.d/<filename>.list. "ppa:owner/name" goes through
// add-apt-repository, which also installs the PPA's key.
type AptRepositoryModule struct {
	*BaseModule
}

// NewAptRepositoryModule creates a new apt_repository module
func NewAptRepositoryModule() *AptRepositoryModule {
	return &AptRepositoryModule{BaseModule: NewBaseModule("apt_repository")}
}

func (m *AptRepositoryModule) GetDescription() string {
	return "Add or remove an APT repository"
}

func (m *AptRepositoryModule) Validate(args map[string]interface{}) error {
	if err := m.BaseModule.Validate(args); err != nil {
		return err
	}
	repo, _ := args["repo"].(string)
	repo = strings.TrimSpace(repo)
	if repo == "" {
		return fmt.Errorf("apt_repository module requires 'repo' parameter")
	}
	if !strings.HasPrefix(repo, "ppa:") && !strings.HasPrefix(repo, "deb ") && !strings.HasPrefix(repo, "deb-src ") {
		return fmt.Errorf("apt_repository: repo must be a \"deb ...\" / \"deb-src ...\" line or \"ppa:owner/name\"")
	}
	if name := getStringArg(args, "filename", ""); name != "" && !safeFileName.MatchString(name) {
		return fmt.Errorf("apt_repository: filename %q may contain only letters, digits, '.', '_' and '-'", name)
	}
	switch getStringArg(args, "state", "present") {
	case "present", "absent":
	default:
		return fmt.Errorf("apt_repository: state must be present or absent")
	}
	return nil
}

var (
	nonWord      = regexp.MustCompile(`[^A-Za-z0-9]+`)
	safeFileName = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
)

// repoFilename is Ansible's default file name: the URI without its scheme,
// every other character than letters and digits as "_"
func repoFilename(line string) string {
	for _, field := range strings.Fields(line) {
		if i := strings.Index(field, "://"); i > 0 {
			return strings.Trim(nonWord.ReplaceAllString(field[i+3:], "_"), "_")
		}
	}
	return strings.Trim(nonWord.ReplaceAllString(line, "_"), "_")
}

func normalizeRepoLine(line string) string {
	return strings.Join(strings.Fields(line), " ")
}

func (m *AptRepositoryModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
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
	repo := normalizeRepoLine(fmt.Sprint(args["repo"]))
	state := getStringArg(args, "state", "present")
	result.Output["repo"] = repo
	result.Output["state"] = state

	// every uncommented line of the .list files, as "file<TAB>line"
	out, err := runShellOnHost(ctx, host, args, `for f in /etc/apt/sources.list /etc/apt/sources.list.d/*.list; do
  [ -f "$f" ] && awk -v f="$f" '!/^[[:space:]]*#/ && NF {print f "\t" $0}' "$f"
done; true`)
	if err != nil {
		return fail(fmt.Sprintf("failed to read the APT sources: %v", err))
	}
	matchFiles := map[string]bool{}
	for _, l := range strings.Split(out, "\n") {
		file, line, ok := strings.Cut(l, "\t")
		if !ok {
			continue
		}
		if repoLineMatches(repo, normalizeRepoLine(line)) {
			matchFiles[file] = true
		}
	}
	// a deb822 .sources file that already describes the repository counts
	// as present; such files are not edited
	inSources := false
	if state == "present" && !strings.HasPrefix(repo, "ppa:") {
		sources, err := runShellOnHost(ctx, host, args, `for f in /etc/apt/sources.list.d/*.sources; do [ -f "$f" ] && { cat "$f"; echo; }; done; true`)
		if err != nil {
			return fail(fmt.Sprintf("failed to read the APT sources: %v", err))
		}
		inSources = deb822Contains(sources, repo)
	}
	present := len(matchFiles) > 0 || inSources
	if present == (state == "present") {
		result.Output["msg"] = fmt.Sprintf("repository is %s", state)
		return done()
	}
	result.Changed = true
	if inCheckMode(args) {
		result.Output["msg"] = fmt.Sprintf("would make the repository %s", state)
		return done()
	}

	if strings.HasPrefix(repo, "ppa:") {
		cmd := []string{"add-apt-repository", "-y"}
		if state == "absent" {
			cmd = append(cmd, "--remove")
		}
		if !getBoolArg(args, "update_cache", true) {
			cmd = append(cmd, "--no-update")
		}
		if _, err := runOnHost(ctx, host, args, append(cmd, repo)...); err != nil {
			return fail(fmt.Sprintf("add-apt-repository failed (is software-properties-common installed?): %v", err))
		}
		result.Output["msg"] = fmt.Sprintf("repository %s", state)
		return done()
	}

	if state == "present" {
		name := getStringArg(args, "filename", "")
		if name == "" {
			name = repoFilename(repo)
		}
		path := "/etc/apt/sources.list.d/" + strings.TrimSuffix(name, ".list") + ".list"
		data, _, err := readHostFile(ctx, host, args, path)
		if err != nil {
			return fail(err.Error())
		}
		content := string(data)
		if content != "" && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		content += repo + "\n"
		if err := writeHostFile(ctx, host, args, path, []byte(content), 0644); err != nil {
			return fail(err.Error())
		}
		result.Output["sources_file"] = path
	} else {
		files := make([]string, 0, len(matchFiles))
		for f := range matchFiles {
			files = append(files, f)
		}
		sort.Strings(files)
		for _, path := range files {
			if err := removeRepoLine(ctx, host, args, path, repo); err != nil {
				return fail(err.Error())
			}
		}
	}
	if getBoolArg(args, "update_cache", true) {
		if _, err := aptGet(ctx, host, args, "update"); err != nil {
			return fail(fmt.Sprintf("failed to update the package lists: %v", err))
		}
	}
	result.Output["msg"] = fmt.Sprintf("repository %s", state)
	return done()
}

// repoLineMatches compares source lines, or a PPA with its launchpad URL
func repoLineMatches(repo, line string) bool {
	if name, ok := strings.CutPrefix(repo, "ppa:"); ok {
		return strings.Contains(line, "ppa.launchpad.net/"+name+"/") || strings.Contains(line, "ppa.launchpadcontent.net/"+name+"/")
	}
	return line == repo
}

// removeRepoLine drops the line from a sources file; a .list file left
// without sources is deleted
func removeRepoLine(ctx context.Context, host types.Host, args map[string]interface{}, path, repo string) error {
	data, _, err := readHostFile(ctx, host, args, path)
	if err != nil {
		return err
	}
	var kept []string
	sources := 0
	for _, line := range strings.Split(strings.TrimRight(string(data), "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "#") && normalizeRepoLine(line) == repo {
			continue
		}
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			sources++
		}
		kept = append(kept, line)
	}
	if sources == 0 && strings.HasPrefix(path, "/etc/apt/sources.list.d/") {
		_, err := runOnHost(ctx, host, args, "rm", "-f", path)
		return err
	}
	return writeHostFile(ctx, host, args, path, []byte(strings.Join(kept, "\n")+"\n"), 0)
}

// deb822Contains tells whether a stanza of deb822 sources covers a one-line
// source: its type, URI and suite, and every component
func deb822Contains(sources, repo string) bool {
	fields := strings.Fields(repo)
	if len(fields) < 3 {
		return false
	}
	typ, rest := fields[0], fields[1:]
	if strings.HasPrefix(rest[0], "[") { // options
		for len(rest) > 0 && !strings.HasSuffix(rest[0], "]") {
			rest = rest[1:]
		}
		if len(rest) > 0 {
			rest = rest[1:]
		}
	}
	if len(rest) < 2 {
		return false
	}
	uri, suite, comps := strings.TrimSuffix(rest[0], "/"), rest[1], rest[2:]
	for _, stanza := range strings.Split(strings.ReplaceAll(sources, "\r", ""), "\n\n") {
		st := map[string][]string{}
		for _, line := range strings.Split(stanza, "\n") {
			k, v, ok := strings.Cut(line, ":")
			if !ok || strings.HasPrefix(strings.TrimSpace(line), "#") {
				continue
			}
			st[strings.ToLower(strings.TrimSpace(k))] = strings.Fields(v)
		}
		if strings.EqualFold(strings.Join(st["enabled"], ""), "no") {
			continue
		}
		var uris []string
		for _, u := range st["uris"] {
			uris = append(uris, strings.TrimSuffix(u, "/"))
		}
		if inList(st["types"], typ) && inList(uris, uri) && inList(st["suites"], suite) && containsAll(st["components"], comps) {
			return true
		}
	}
	return false
}

func inList(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func containsAll(list, want []string) bool {
	for _, w := range want {
		if !inList(list, w) {
			return false
		}
	}
	return true
}
