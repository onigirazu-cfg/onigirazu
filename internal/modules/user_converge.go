package modules

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// The user module converges an account as Ansible's does: a missing user is
// created with every option given, an existing one gets usermod for the
// attributes that differ (shell, home, uid, primary group, groups, comment,
// password), and state: absent removes the home only with remove: true.

// normalizeUserArgs applies the Ansible defaults and forms: state present,
// groups as a list or a comma separated string
func normalizeUserArgs(args map[string]interface{}) {
	if _, ok := args["state"]; !ok {
		args["state"] = "present"
	}
	if list, ok := args["groups"].([]interface{}); ok {
		names := make([]string, 0, len(list))
		for _, g := range list {
			names = append(names, fmt.Sprint(g))
		}
		args["groups"] = strings.Join(names, ",")
	}
}

// account is what getent and id report about a user
type account struct {
	uid, gid, comment, home, shell string
	primary                        string   // primary group name
	groups                         []string // supplementary group names
}

func readAccount(ctx context.Context, host types.Host, args map[string]interface{}, name string) (*account, error) {
	out, err := runOnHost(ctx, host, args, "getent", "passwd", name)
	if err != nil {
		return nil, nil // no such user
	}
	f := strings.Split(strings.TrimSpace(out), ":")
	if len(f) < 7 {
		return nil, fmt.Errorf("unexpected passwd entry for %s: %q", name, out)
	}
	a := &account{uid: f[2], gid: f[3], comment: f[4], home: f[5], shell: f[6]}
	primary, err := runOnHost(ctx, host, args, "id", "-gn", name)
	if err != nil {
		return nil, fmt.Errorf("id -gn %s: %w", name, err)
	}
	a.primary = strings.TrimSpace(primary)
	all, err := runOnHost(ctx, host, args, "id", "-Gn", name)
	if err != nil {
		return nil, fmt.Errorf("id -Gn %s: %w", name, err)
	}
	for _, g := range strings.Fields(all) {
		if g != a.primary {
			a.groups = append(a.groups, g)
		}
	}
	return a, nil
}

// splitGroups turns "a, b,c" into a sorted set
func splitGroups(s string) []string {
	seen := map[string]bool{}
	var out []string
	for _, g := range strings.Split(s, ",") {
		if g = strings.TrimSpace(g); g != "" && !seen[g] {
			seen[g] = true
			out = append(out, g)
		}
	}
	sort.Strings(out)
	return out
}

// usermodArgs are the usermod options that bring an existing account to the
// requested attributes; empty when it already matches
func usermodArgs(ctx context.Context, host types.Host, args map[string]interface{}, a *account) ([]string, error) {
	var opts []string
	if shell := getStringArg(args, "shell", ""); shell != "" && shell != a.shell {
		opts = append(opts, "-s", shell)
	}
	if home := getStringArg(args, "home", ""); home != "" && home != a.home {
		opts = append(opts, "-d", home)
		if getBoolArg(args, "move_home", false) {
			opts = append(opts, "-m")
		}
	}
	if uid := getStringArg(args, "uid", ""); uid != "" && uid != a.uid {
		opts = append(opts, "-u", uid)
	}
	if group := getStringArg(args, "group", ""); group != "" && group != a.primary && group != a.gid {
		opts = append(opts, "-g", group)
	}
	if comment, ok := args["comment"].(string); ok && comment != a.comment {
		opts = append(opts, "-c", comment)
	}
	if groups, ok := args["groups"].(string); ok {
		// the primary group is not a supplementary one: leave it out of the
		// comparison, or every run would see a difference
		var want []string
		for _, g := range splitGroups(groups) {
			if g != a.primary {
				want = append(want, g)
			}
		}
		have := map[string]bool{}
		for _, g := range a.groups {
			have[g] = true
		}
		missing := false
		for _, g := range want {
			if !have[g] {
				missing = true
			}
		}
		if getBoolArg(args, "append", false) {
			if missing {
				opts = append(opts, "-a", "-G", strings.Join(want, ","))
			}
		} else if missing || len(want) != len(a.groups) {
			opts = append(opts, "-G", strings.Join(want, ","))
		}
	}
	if password := getStringArg(args, "password", ""); password != "" {
		// the stored hash needs root to read
		out, err := runOnHost(ctx, host, args, "getent", "shadow", getStringArg(args, "name", ""))
		if err != nil {
			return nil, fmt.Errorf("cannot read the password hash (needs become): %w", err)
		}
		f := strings.Split(strings.TrimSpace(out), ":")
		if len(f) < 2 || f[1] != password {
			opts = append(opts, "-p", password)
		}
	}
	return opts, nil
}

// convergeUser is the present/absent logic of the user module
func (m *UserModuleFixed) convergeUser(ctx context.Context, host types.Host, args map[string]interface{}, result types.TaskResult, start time.Time) (types.TaskResult, error) {
	name := getStringArg(args, "name", "")
	state := getStringArg(args, "state", "present")
	fail := func(msg string) (types.TaskResult, error) {
		result.Success, result.Error = false, msg
		result.Duration = time.Since(start)
		return result, nil
	}
	done := func(changed bool, msg string) (types.TaskResult, error) {
		result.Success, result.Changed = true, changed
		result.Output = map[string]interface{}{"msg": msg, "name": name, "state": state}
		result.Duration = time.Since(start)
		return result, nil
	}

	a, err := readAccount(ctx, host, args, name)
	if err != nil {
		return fail(err.Error())
	}

	if state == "absent" {
		if a == nil {
			return done(false, fmt.Sprintf("user %s does not exist", name))
		}
		if inCheckMode(args) {
			return done(true, fmt.Sprintf("user %s would be removed", name))
		}
		argv := []string{"userdel"}
		if getBoolArg(args, "remove", false) {
			argv = append(argv, "-r")
		}
		if _, err := runOnHost(ctx, host, args, append(argv, name)...); err != nil {
			return fail(fmt.Sprintf("error removing user: %v", err))
		}
		return done(true, fmt.Sprintf("user %s removed", name))
	}

	if a == nil {
		if inCheckMode(args) {
			return done(true, fmt.Sprintf("user %s would be created", name))
		}
		argv := m.buildUserAddCommand(name, args)
		// useradd refuses to create the user's own group when it exists:
		// make that group the primary one
		if getStringArg(args, "group", "") == "" {
			if _, err := runOnHost(ctx, host, args, "getent", "group", name); err == nil {
				argv = append(argv[:len(argv)-1], "-g", name, name)
			}
		}
		if _, err := runOnHost(ctx, host, args, argv...); err != nil {
			return fail(fmt.Sprintf("error creating user: %v", err))
		}
		return done(true, fmt.Sprintf("user %s created", name))
	}

	opts, err := usermodArgs(ctx, host, args, a)
	if err != nil {
		return fail(err.Error())
	}
	if len(opts) == 0 {
		return done(false, fmt.Sprintf("user %s is up to date", name))
	}
	if inCheckMode(args) {
		return done(true, fmt.Sprintf("user %s would be modified", name))
	}
	if _, err := runOnHost(ctx, host, args, append(append([]string{"usermod"}, opts...), name)...); err != nil {
		return fail(fmt.Sprintf("error modifying user: %v", err))
	}
	return done(true, fmt.Sprintf("user %s modified", name))
}
