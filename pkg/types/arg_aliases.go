package types

// argAliases are the other names Ansible accepts for module arguments, by
// module: alias -> argument. A playbook that says apt pkg: or file dest:
// gets the argument the module reads.
var argAliases = map[string]map[string]string{
	"apt":         {"pkg": "name", "package": "name", "update-cache": "update_cache", "default-release": "default_release", "install-recommends": "install_recommends"},
	"yum":         {"pkg": "name"},
	"dnf":         {"pkg": "name"},
	"package":     {"pkg": "name"},
	"pip":         {"pkg": "name"},
	"file":        {"dest": "path", "name": "path"},
	"stat":        {"dest": "path", "name": "path"},
	"lineinfile":  {"dest": "path", "destfile": "path", "name": "path"},
	"blockinfile": {"dest": "path", "destfile": "path", "name": "path"},
	"replace":     {"dest": "path", "destfile": "path", "name": "path"},
	"ini_file":    {"dest": "path"},
	"mount":       {"name": "path"},
	"find":        {"name": "path"},
	"user":        {"user": "name"},
	"service":     {"service": "name"},
	"systemd":     {"service": "name", "unit": "name", "daemon-reload": "daemon_reload", "daemon-reexec": "daemon_reexec"},
	"sysctl":      {"key": "name", "val": "value"},
	"git":         {"name": "repo"},
}

// CanonicalArgs renames the aliases of a module's arguments; an argument
// given under both names keeps its own
func CanonicalArgs(module string, args map[string]interface{}) {
	aliases := argAliases[module]
	for alias, name := range aliases {
		value, ok := args[alias]
		if !ok {
			continue
		}
		if _, set := args[name]; !set {
			args[name] = value
		}
		delete(args, alias)
	}
}
