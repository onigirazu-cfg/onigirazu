package modules

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// Deb822RepositoryModule writes an APT source in the deb822 format
// (Ansible's deb822_repository): /etc/apt/sources.list.d/<name>.sources with
// Types, URIs, Suites, Components, Signed-By (a key file, an URL fetched to
// /etc/apt/keyrings/<name>.asc, or inline armored text), Architectures,
// Enabled, and any other deb822 field given; state absent removes the file.
type Deb822RepositoryModule struct {
	*BaseModule
}

// NewDeb822RepositoryModule creates the module
func NewDeb822RepositoryModule() *Deb822RepositoryModule {
	return &Deb822RepositoryModule{BaseModule: NewBaseModule("deb822_repository")}
}

func (m *Deb822RepositoryModule) GetDescription() string {
	return "Add or remove an APT source in the deb822 format"
}

func (m *Deb822RepositoryModule) Validate(args map[string]interface{}) error {
	if getStringArg(args, "name", "") == "" {
		return fmt.Errorf("deb822_repository requires 'name'")
	}
	state := getStringArg(args, "state", "present")
	if state != "present" && state != "absent" {
		return fmt.Errorf("deb822_repository: state must be present or absent")
	}
	if state == "present" && (len(listArg(args, "uris")) == 0 || len(listArg(args, "suites")) == 0) {
		return fmt.Errorf("deb822_repository requires 'uris' and 'suites'")
	}
	return nil
}

// deb822Fields are the fields of the .sources file in their usual order
var deb822Fields = []struct{ arg, field string }{
	{"types", "Types"}, {"uris", "URIs"}, {"suites", "Suites"}, {"components", "Components"},
	{"architectures", "Architectures"}, {"languages", "Languages"}, {"targets", "Targets"},
	{"pdiffs", "PDiffs"}, {"by_hash", "By-Hash"}, {"allow_insecure", "Allow-Insecure"},
	{"allow_weak", "Allow-Weak"}, {"allow_downgrade_to_insecure", "Allow-Downgrade-To-Insecure"},
	{"trusted", "Trusted"}, {"check_valid_until", "Check-Valid-Until"}, {"valid_until_min", "Valid-Until-Min"},
	{"valid_until_max", "Valid-Until-Max"}, {"check_date", "Check-Date"}, {"date_max_future", "Date-Max-Future"},
	{"inrelease_path", "InRelease-Path"}, {"include", "Include"}, {"exclude", "Exclude"},
}

// deb822Content renders the .sources file; signedBy is the Signed-By value
// (a path or an inline key)
func deb822Content(args map[string]interface{}, signedBy string) string {
	var b strings.Builder
	if v := listArg(args, "types"); len(v) == 0 {
		b.WriteString("Types: deb\n")
	}
	for _, f := range deb822Fields {
		v, set := args[f.arg]
		if !set || v == nil {
			continue
		}
		var text string
		switch x := v.(type) {
		case bool:
			text = map[bool]string{true: "yes", false: "no"}[x]
		case []interface{}:
			text = strings.Join(listArg(args, f.arg), " ")
		default:
			text = strings.TrimSpace(fmt.Sprint(x))
		}
		if text != "" {
			fmt.Fprintf(&b, "%s: %s\n", f.field, text)
		}
	}
	if signedBy != "" {
		if strings.Contains(signedBy, "-----BEGIN") {
			b.WriteString("Signed-By:\n")
			for _, line := range strings.Split(strings.TrimSpace(signedBy), "\n") {
				if strings.TrimSpace(line) == "" {
					b.WriteString(" .\n")
				} else {
					b.WriteString(" " + line + "\n")
				}
			}
		} else {
			fmt.Fprintf(&b, "Signed-By: %s\n", signedBy)
		}
	}
	if v, set := args["enabled"]; set {
		if on, ok := v.(bool); ok && !on {
			b.WriteString("Enabled: no\n")
		}
	}
	return b.String()
}

func (m *Deb822RepositoryModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
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
	path := "/etc/apt/sources.list.d/" + name + ".sources"
	result.Output["dest"] = path
	current, exists, err := readHostFile(ctx, host, args, path)
	if err != nil {
		return fail(err.Error())
	}
	if getStringArg(args, "state", "present") == "absent" {
		if !exists {
			result.Duration = time.Since(start)
			return result, nil
		}
		result.Changed = true
		if !inCheckMode(args) {
			if _, err := runOnHost(ctx, host, args, "rm", "-f", path, "/etc/apt/keyrings/"+name+".asc"); err != nil {
				return fail(err.Error())
			}
		}
		result.Duration = time.Since(start)
		return result, nil
	}
	// the signing key: a path on the host, an URL to fetch, or key text
	signedBy := getStringArg(args, "signed_by", "")
	switch {
	case strings.HasPrefix(signedBy, "http://"), strings.HasPrefix(signedBy, "https://"):
		keyPath := "/etc/apt/keyrings/" + name + ".asc"
		if _, keyThere, _ := readHostFile(ctx, host, args, keyPath); !keyThere && !inCheckMode(args) {
			if _, err := runShellOnHost(ctx, host, args, "mkdir -p /etc/apt/keyrings && (curl -fsSL "+shellQuote(signedBy)+" -o "+shellQuote(keyPath)+" || wget -qO "+shellQuote(keyPath)+" "+shellQuote(signedBy)+")"); err != nil {
				return fail(fmt.Sprintf("fetching the key: %v", err))
			}
			result.Changed = true
		}
		signedBy = keyPath
	}
	content := deb822Content(args, signedBy)
	if exists && string(current) == content {
		result.Duration = time.Since(start)
		return result, nil
	}
	result.Changed = true
	if !inCheckMode(args) {
		if err := writeHostFile(ctx, host, args, path, []byte(content), 0o644); err != nil {
			return fail(err.Error())
		}
		if getBoolArg(args, "update_cache", true) {
			if _, err := runShellOnHost(ctx, host, args, "apt-get -o DPkg::Lock::Timeout=60 update -qq"); err != nil {
				result.Output["warning"] = fmt.Sprintf("apt-get update: %v", err)
			}
		}
	}
	keys := make([]string, 0, len(args))
	for k := range args {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	result.Duration = time.Since(start)
	return result, nil
}
