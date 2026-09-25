package inventory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// Variables from group_vars/ and host_vars/ next to an inventory source, as in
// Ansible: <dir>/group_vars/<group>.yml (or .yaml, .json, or a directory of
// such files merged in name order) and the same for host_vars/<host>. They
// override the variables written in the inventory itself.

var varsFileExts = map[string]bool{".yml": true, ".yaml": true, ".json": true, "": true}

// loadVarsDirs reads group_vars and host_vars in dir into the loader state;
// later sources override earlier ones
func (msl *MultiSourceLoader) loadVarsDirs(dir string) error {
	for kind, target := range map[string]map[string]map[string]interface{}{
		"group_vars": msl.groupVarsFiles,
		"host_vars":  msl.hostVarsFiles,
	} {
		entries, err := os.ReadDir(filepath.Join(dir, kind))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		for _, e := range entries {
			name := e.Name()
			if strings.HasPrefix(name, ".") {
				continue
			}
			path := filepath.Join(dir, kind, name)
			var vars map[string]interface{}
			if e.IsDir() {
				vars, err = readVarsDir(path)
			} else {
				ext := filepath.Ext(name)
				if !varsFileExts[ext] {
					continue
				}
				name = strings.TrimSuffix(name, ext)
				vars, err = readVarsFile(path)
			}
			if err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}
			if target[name] == nil {
				target[name] = make(map[string]interface{})
			}
			for k, v := range vars {
				target[name][k] = v
			}
			msl.logger.Debug("Loaded %d variables from %s", len(vars), path)
		}
	}
	return nil
}

func readVarsDir(dir string) (map[string]interface{}, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && !strings.HasPrefix(e.Name(), ".") && varsFileExts[filepath.Ext(e.Name())] {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	merged := make(map[string]interface{})
	for _, n := range names {
		vars, err := readVarsFile(filepath.Join(dir, n))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", n, err)
		}
		for k, v := range vars {
			merged[k] = v
		}
	}
	return merged, nil
}

func readVarsFile(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	vars := make(map[string]interface{})
	if filepath.Ext(path) == ".json" {
		err = json.Unmarshal(data, &vars)
	} else {
		err = yaml.Unmarshal(data, &vars)
	}
	return vars, err
}

// applyVarsFiles puts the collected group_vars and host_vars into the merged
// inventory: into group variables (creating groups such as "all" when only
// a file names them) and into every copy of a host
func (msl *MultiSourceLoader) applyVarsFiles() {
	for name, vars := range msl.groupVarsFiles {
		group := msl.groupMap[name]
		if group == nil {
			group = &types.Group{Name: name, Hosts: map[string]*types.Host{}}
			msl.groupMap[name] = group
		}
		if group.Vars == nil {
			group.Vars = make(map[string]interface{})
		}
		for k, v := range vars {
			group.Vars[k] = v
		}
	}
	setHostVars := func(name string, host *types.Host) {
		vars, ok := msl.hostVarsFiles[name]
		if !ok || host == nil {
			return
		}
		if host.Vars == nil {
			host.Vars = make(map[string]interface{})
		}
		for k, v := range vars {
			host.Vars[k] = v
		}
	}
	// group entries are keyed by host name; their Name field may be empty
	seen := make(map[*types.Host]bool)
	for name, host := range msl.hostMap {
		setHostVars(name, host)
		seen[host] = true
	}
	for _, group := range msl.groupMap {
		for name, host := range group.Hosts {
			if !seen[host] {
				setHostVars(name, host)
				seen[host] = true
			}
		}
	}
}
