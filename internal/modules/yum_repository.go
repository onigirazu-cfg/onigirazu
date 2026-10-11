package modules

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// YumRepositoryModule writes a .repo file (Ansible's yum_repository): name,
// description, baseurl/metalink/mirrorlist, enabled, gpgcheck, gpgkey,
// file (default name), reposdir, state, and any other yum option
type YumRepositoryModule struct{ *BaseModule }

func NewYumRepositoryModule() *YumRepositoryModule {
	return &YumRepositoryModule{BaseModule: NewBaseModule("yum_repository")}
}

func (m *YumRepositoryModule) GetDescription() string {
	return "Add or remove a yum/dnf repository file"
}

// yumRepoOptions are the arguments written as repo options, in this order
var yumRepoOptions = []string{"baseurl", "metalink", "mirrorlist", "enabled", "gpgcheck", "repo_gpgcheck", "gpgkey", "gpgcakey",
	"exclude", "includepkgs", "priority", "cost", "sslverify", "sslcacert", "sslclientcert", "sslclientkey", "proxy", "proxy_username",
	"proxy_password", "username", "password", "metadata_expire", "module_hotfixes", "skip_if_unavailable", "keepcache", "retries",
	"timeout", "throttle", "bandwidth", "countme", "enablegroups", "failovermethod", "http_caching", "ip_resolve", "protect",
	"s3_enabled", "ui_repoid_vars", "deltarpm_metadata_percentage", "deltarpm_percentage", "async"}

func yumRepoContent(args map[string]interface{}) string {
	var b strings.Builder
	fmt.Fprintf(&b, "[%s]\n", getStringArg(args, "name", ""))
	fmt.Fprintf(&b, "name=%s\n", getStringArg(args, "description", getStringArg(args, "name", "")))
	for _, k := range yumRepoOptions {
		v, set := args[k]
		if !set || v == nil {
			continue
		}
		var text string
		switch x := v.(type) {
		case bool:
			text = map[bool]string{true: "1", false: "0"}[x]
		case []interface{}:
			text = strings.Join(listArg(args, k), " ")
		default:
			text = strings.TrimSpace(fmt.Sprint(x))
		}
		if text != "" {
			fmt.Fprintf(&b, "%s=%s\n", k, text)
		}
	}
	return b.String()
}

func (m *YumRepositoryModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	start := time.Now()
	result := types.TaskResult{TaskName: taskName(args), Host: host.Name, Module: m.name, Timestamp: start, Success: true, Output: map[string]interface{}{}}
	fail := func(msg string) (types.TaskResult, error) {
		result.Success, result.Error, result.Duration = false, msg, time.Since(start)
		return result, nil
	}
	name := getStringArg(args, "name", "")
	if name == "" {
		return fail("yum_repository requires 'name'")
	}
	state := getStringArg(args, "state", "present")
	if state == "present" && getStringArg(args, "baseurl", "") == "" && getStringArg(args, "metalink", "") == "" && getStringArg(args, "mirrorlist", "") == "" {
		return fail("yum_repository requires baseurl, metalink or mirrorlist")
	}
	file := getStringArg(args, "file", name)
	path := strings.TrimRight(getStringArg(args, "reposdir", "/etc/yum.repos.d"), "/") + "/" + file + ".repo"
	result.Output["repo"], result.Output["dest"] = name, path
	current, exists, err := readHostFile(ctx, host, args, path)
	if err != nil {
		return fail(err.Error())
	}
	// the file may hold several repos: replace or remove this one's section
	sections := splitIniSections(string(current))
	if state == "absent" {
		if _, ok := sections[name]; !ok {
			result.Duration = time.Since(start)
			return result, nil
		}
		delete(sections, name)
		result.Changed = true
		if !inCheckMode(args) {
			if len(sections) == 0 {
				if _, err := runOnHost(ctx, host, args, "rm", "-f", path); err != nil {
					return fail(err.Error())
				}
			} else if err := writeHostFile(ctx, host, args, path, []byte(joinIniSections(sections)), 0o644); err != nil {
				return fail(err.Error())
			}
		}
		result.Duration = time.Since(start)
		return result, nil
	}
	want := yumRepoContent(args)
	if exists && sections[name] == want {
		result.Duration = time.Since(start)
		return result, nil
	}
	sections[name] = want
	result.Changed = true
	if !inCheckMode(args) {
		if err := writeHostFile(ctx, host, args, path, []byte(joinIniSections(sections)), 0o644); err != nil {
			return fail(err.Error())
		}
	}
	result.Duration = time.Since(start)
	return result, nil
}

// splitIniSections maps "[name]" sections to their text (header included)
func splitIniSections(text string) map[string]string {
	out := map[string]string{}
	name := ""
	var b strings.Builder
	flush := func() {
		if name != "" {
			out[name] = strings.TrimRight(b.String(), "\n") + "\n"
		}
		b.Reset()
	}
	for _, line := range strings.Split(text, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "[") && strings.HasSuffix(t, "]") {
			flush()
			name = strings.Trim(t, "[]")
		}
		if name != "" {
			b.WriteString(line + "\n")
		}
	}
	flush()
	return out
}

func joinIniSections(sections map[string]string) string {
	names := make([]string, 0, len(sections))
	for n := range sections {
		names = append(names, n)
	}
	sort.Strings(names)
	parts := make([]string, 0, len(names))
	for _, n := range names {
		parts = append(parts, strings.TrimRight(sections[n], "\n"))
	}
	return strings.Join(parts, "\n\n") + "\n"
}
