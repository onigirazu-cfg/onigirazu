package types

import "strings"

// collectionModules maps fully qualified names of collection modules to the
// built-in modules that implement them
var collectionModules = map[string]string{
	"ansible.posix.sysctl":                  "sysctl",
	"ansible.posix.mount":                   "mount",
	"ansible.posix.authorized_key":          "authorized_key",
	"community.general.archive":             "archive",
	"community.docker.docker_container":     "docker_container",
	"community.sops.load_vars":              "include_vars",
	"community.routeros.api":                "routeros_api",
	"community.routeros.api_modify":         "routeros_api",
	"community.routeros.command":            "routeros_command",
	"community.routeros.facts":              "routeros_facts",
	"community.docker.docker_image":         "docker_image",
	"community.docker.docker_compose":       "docker_compose",
	"community.docker.docker_compose_v2":    "docker_compose",
	"community.docker.docker_host_info":     "docker_host_info",
	"community.general.ufw":                 "ufw",
	"community.general.ini_file":            "ini_file",
	"community.general.timezone":            "timezone",
	"community.mysql.mysql_db":              "mysql_db",
	"community.mysql.mysql_user":            "mysql_user",
	"community.postgresql.postgresql_db":    "postgresql_db",
	"community.postgresql.postgresql_user":  "postgresql_user",
	"ansible.windows.win_ping":              "win_ping",
	"ansible.windows.win_command":           "win_command",
	"ansible.windows.win_shell":             "win_shell",
	"ansible.windows.win_powershell":        "win_powershell",
	"ansible.windows.win_regedit":           "win_regedit",
	"ansible.windows.win_file":              "win_file",
	"ansible.windows.win_copy":              "win_copy",
	"ansible.windows.win_service":           "win_service",
	"community.windows.win_timezone":        "win_timezone",
	"community.windows.win_firewall_rule":   "win_firewall_rule",
	"ansible.windows.win_firewall":          "win_firewall",
	"community.windows.win_firewall":        "win_firewall",
	"ansible.windows.win_group_membership":  "win_group_membership",
	"ansible.windows.win_feature":           "win_feature",
	"ansible.windows.win_reboot":            "win_reboot",
	"community.windows.win_scheduled_task":  "win_scheduled_task",
	"chocolatey.chocolatey.win_chocolatey":  "win_chocolatey",
	"ansible.windows.win_optional_feature":  "win_optional_feature",
	"community.windows.win_disk_facts":      "win_disk_facts",
	"community.windows.win_initialize_disk": "win_initialize_disk",
	"community.windows.win_partition":       "win_partition",
	"community.windows.win_format":          "win_format",
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
