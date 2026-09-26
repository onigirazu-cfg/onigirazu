package parser

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// roleFileDirs is where a role's tasks find their sources, as in Ansible:
// template in templates/, copy, script and unarchive in files/
var roleFileDirs = map[string]string{"template": "templates", "copy": "files", "script": "files", "unarchive": "files"}

// resolveRoleFiles makes the relative sources of a role's tasks point into
// the role: roles/x/templates/src or roles/x/files/src, else roles/x/src.
// A templated source cannot be checked and goes to the role's directory for
// its module.
func resolveRoleFiles(tasks []types.Task, rolePath string) {
	for i := range tasks {
		t := &tasks[i]
		resolveRoleFiles(t.Block, rolePath)
		resolveRoleFiles(t.Rescue, rolePath)
		resolveRoleFiles(t.Always, rolePath)

		dir, ok := roleFileDirs[t.Module]
		if !ok || t.Args == nil {
			continue
		}
		if remote, _ := t.Args["remote_src"].(bool); remote {
			continue
		}
		key := "src"
		if t.Module == "script" {
			key = "script"
		}
		src, ok := t.Args[key].(string)
		if !ok || src == "" || filepath.IsAbs(src) {
			continue
		}
		if strings.Contains(src, "{{") {
			t.Args[key] = filepath.Join(rolePath, dir, src)
			continue
		}
		for _, candidate := range []string{filepath.Join(rolePath, dir, src), filepath.Join(rolePath, src)} {
			if _, err := os.Stat(candidate); err == nil {
				t.Args[key] = candidate
				break
			}
		}
	}
}
