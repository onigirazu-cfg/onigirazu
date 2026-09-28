package parser

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Where roles are found besides roles/ next to the play's playbook, as in
// Ansible: roles_path and collections_path from onigirazu.yml, the
// ANSIBLE_ROLES_PATH / ANSIBLE_COLLECTIONS_PATH environment, ansible.cfg
// (ANSIBLE_CONFIG, ./ansible.cfg, next to the playbook, ~/.ansible.cfg,
// /etc/ansible/ansible.cfg; the first found), then ~/.ansible/roles and
// ~/.ansible/collections.

var (
	searchMu        sync.Mutex
	extraRoles      []string // onigirazu.yml roles_path
	extraCollection []string // onigirazu.yml collections_path
)

// SetRoleSearch adds the roles_path and collections_path of onigirazu.yml
// (relative paths start at the config file's directory)
func SetRoleSearch(roles, collections []string) {
	searchMu.Lock()
	defer searchMu.Unlock()
	extraRoles, extraCollection = roles, collections
}

// roleSearch returns the role and collection directories for a playbook in
// playbookDir
func roleSearch(playbookDir string) (roles, collections []string) {
	searchMu.Lock()
	roles = append(roles, extraRoles...)
	collections = append(collections, extraCollection...)
	searchMu.Unlock()

	cfgRoles, cfgCollections := ansibleCfgPaths(playbookDir)
	roles = append(roles, envPaths("ANSIBLE_ROLES_PATH")...)
	roles = append(roles, cfgRoles...)
	collections = append(collections, envPaths("ANSIBLE_COLLECTIONS_PATH")...)
	collections = append(collections, envPaths("ANSIBLE_COLLECTIONS_PATHS")...)
	collections = append(collections, cfgCollections...)
	if home, err := os.UserHomeDir(); err == nil {
		roles = append(roles, filepath.Join(home, ".ansible", "roles"))
		collections = append(collections, filepath.Join(home, ".ansible", "collections"))
	}
	roles = append(roles, "/usr/share/ansible/roles", "/etc/ansible/roles")
	collections = append(collections, "/usr/share/ansible/collections")
	return roles, collections
}

func envPaths(name string) []string {
	return splitPaths(os.Getenv(name), "")
}

// splitPaths reads a colon separated path list; ~ is the home directory,
// relative entries start at base
func splitPaths(value, base string) []string {
	var out []string
	for _, p := range strings.Split(value, ":") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if strings.HasPrefix(p, "~") {
			if home, err := os.UserHomeDir(); err == nil {
				p = filepath.Join(home, strings.TrimPrefix(p, "~"))
			}
		}
		if !filepath.IsAbs(p) && base != "" {
			p = filepath.Join(base, p)
		}
		out = append(out, p)
	}
	return out
}

// ansibleCfgPaths reads roles_path and collections_path from the first
// ansible.cfg Ansible would read
func ansibleCfgPaths(playbookDir string) (roles, collections []string) {
	var candidates []string
	if env := os.Getenv("ANSIBLE_CONFIG"); env != "" {
		candidates = append(candidates, env)
	}
	candidates = append(candidates, "ansible.cfg")
	if playbookDir != "" {
		candidates = append(candidates, filepath.Join(playbookDir, "ansible.cfg"), filepath.Join(playbookDir, "..", "ansible.cfg"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, ".ansible.cfg"))
	}
	candidates = append(candidates, "/etc/ansible/ansible.cfg")
	for _, c := range candidates {
		f, err := os.Open(c) // #nosec G304 -- Ansible's config file
		if err != nil {
			continue
		}
		base, _ := filepath.Abs(filepath.Dir(c))
		section := ""
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
				continue
			}
			if strings.HasPrefix(line, "[") {
				section = strings.Trim(line, "[] ")
				continue
			}
			key, value, ok := strings.Cut(line, "=")
			if !ok || section != "defaults" {
				continue
			}
			switch strings.TrimSpace(key) {
			case "roles_path":
				roles = splitPaths(strings.TrimSpace(value), base)
			case "collections_path", "collections_paths":
				collections = splitPaths(strings.TrimSpace(value), base)
			}
		}
		_ = f.Close()
		return roles, collections
	}
	return nil, nil
}

// findRole resolves a role name: roles/ of the play, a path relative to the
// playbook, the roles path, and ns.collection.role in the collections path.
// It returns the directory and the places it looked.
func findRole(name, playRoles string) (string, []string) {
	playbookDir := filepath.Dir(playRoles)
	var tried []string
	check := func(p string) bool {
		tried = append(tried, p)
		info, err := os.Stat(p)
		return err == nil && info.IsDir()
	}
	if filepath.IsAbs(name) {
		if check(name) {
			return name, tried
		}
		return "", tried
	}
	if check(filepath.Join(playRoles, name)) {
		return filepath.Join(playRoles, name), tried
	}
	if check(filepath.Join(playbookDir, name)) {
		return filepath.Join(playbookDir, name), tried
	}
	roles, collections := roleSearch(playbookDir)
	for _, dir := range roles {
		if check(filepath.Join(dir, name)) {
			return filepath.Join(dir, name), tried
		}
	}
	if parts := strings.Split(name, "."); len(parts) == 3 {
		for _, dir := range collections {
			base := dir
			if filepath.Base(dir) != "ansible_collections" {
				base = filepath.Join(dir, "ansible_collections")
			}
			p := filepath.Join(base, parts[0], parts[1], "roles", parts[2])
			if check(p) {
				return p, tried
			}
		}
	}
	return "", tried
}
