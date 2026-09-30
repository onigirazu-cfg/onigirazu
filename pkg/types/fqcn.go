package types

import "strings"

// collectionModules maps fully qualified names of collection modules to the
// built-in modules that implement them
var collectionModules = map[string]string{
	"ansible.posix.sysctl":                 "sysctl",
	"ansible.posix.mount":                  "mount",
	"ansible.posix.authorized_key":         "authorized_key",
	"community.general.archive":            "archive",
	"community.docker.docker_container":    "docker_container",
	"community.docker.docker_image":        "docker_image",
	"community.docker.docker_compose":      "docker_compose",
	"community.docker.docker_compose_v2":   "docker_compose",
	"community.docker.docker_host_info":    "docker_host_info",
	"community.general.ufw":                "ufw",
	"community.general.ini_file":           "ini_file",
	"community.general.timezone":           "timezone",
	"community.mysql.mysql_db":             "mysql_db",
	"community.mysql.mysql_user":           "mysql_user",
	"community.postgresql.postgresql_db":   "postgresql_db",
	"community.postgresql.postgresql_user": "postgresql_user",
	"ansible.windows.win_ping":             "win_ping",
	"ansible.windows.win_command":          "win_command",
	"ansible.windows.win_shell":            "win_shell",
}

// ShortModuleName turns ansible.builtin.copy into copy; other names are
// returned unchanged
func ShortModuleName(name string) string {
	for _, prefix := range []string{"ansible.builtin.", "ansible.legacy."} {
		if short, ok := strings.CutPrefix(name, prefix); ok {
			name = short
			break
		}
	}
	// dnf is yum's successor with the same arguments; the yum module runs
	// whichever the host has
	if name == "dnf" || name == "dnf5" {
		return "yum"
	}
	// the v2 module with collections: [community.docker] or its bare name
	if name == "docker_compose_v2" {
		return "docker_compose"
	}
	if !strings.Contains(name, ".") {
		return name
	}
	if short, ok := collectionModules[name]; ok {
		return short
	}
	return name
}

// shortModuleKeys renames the fully qualified keys of a task (modules and
// keywords such as ansible.builtin.include_tasks) to their short names
func shortModuleKeys(taskMap map[string]interface{}) {
	for key, value := range taskMap {
		if short := ShortModuleName(key); short != key {
			if _, taken := taskMap[short]; !taken {
				taskMap[short] = value
				delete(taskMap, key)
			}
		}
	}
	if module, ok := taskMap["module"].(string); ok {
		taskMap["module"] = ShortModuleName(module)
	}
	if local, ok := taskMap["local_action"].(string); ok {
		module, rest, _ := strings.Cut(strings.TrimSpace(local), " ")
		taskMap["local_action"] = strings.TrimSpace(ShortModuleName(module) + " " + rest)
	}
	if local, ok := taskMap["local_action"].(map[string]interface{}); ok {
		if module, ok := local["module"].(string); ok {
			local["module"] = ShortModuleName(module)
		}
	}
}
