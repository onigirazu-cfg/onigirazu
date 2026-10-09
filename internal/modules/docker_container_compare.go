package modules

import (
	"fmt"
	"sort"
	"strings"
)

// containerDiffs lists the options of the task that the existing container
// does not have, as Ansible's docker_container compares them: only options
// the task sets; env as "these variables are set" (the image adds its own);
// the image by ID, so a newer pull of the same tag counts. imageID is the ID
// of the task's image on the host, "" when it is not there.
func containerDiffs(inspect map[string]interface{}, args map[string]interface{}, imageID string) []string {
	var diffs []string
	config, _ := inspect["Config"].(map[string]interface{})
	hostConfig, _ := inspect["HostConfig"].(map[string]interface{})

	if image := getStringArg(args, "image", ""); image != "" {
		current, _ := inspect["Image"].(string)
		if imageID == "" || imageID != current {
			diffs = append(diffs, "image")
		}
	}

	if env, ok := args["env"].(map[string]interface{}); ok {
		have := map[string]bool{}
		for _, e := range stringList(config["Env"]) {
			have[e] = true
		}
		for k, v := range env {
			if !have[fmt.Sprintf("%s=%v", k, v)] {
				diffs = append(diffs, "env")
				break
			}
		}
	}

	if ports, ok := args["ports"].([]interface{}); ok {
		want := map[string]bool{}
		for _, p := range ports {
			want[normalizePort(fmt.Sprint(p))] = true
		}
		have := map[string]bool{}
		bindings, _ := hostConfig["PortBindings"].(map[string]interface{})
		for containerPort, list := range bindings {
			binds, _ := list.([]interface{})
			for _, b := range binds {
				bm, _ := b.(map[string]interface{})
				ip, _ := bm["HostIp"].(string)
				hp, _ := bm["HostPort"].(string)
				have[normalizePort(strings.Trim(ip+":"+hp+":"+containerPort, ":"))] = true
			}
		}
		if !sameSet(want, have) {
			diffs = append(diffs, "ports")
		}
	}

	if volumes, ok := args["volumes"].([]interface{}); ok {
		want := map[string]bool{}
		for _, v := range volumes {
			want[fmt.Sprint(v)] = true
		}
		have := map[string]bool{}
		for _, b := range stringList(hostConfig["Binds"]) {
			have[b] = true
		}
		if !sameSet(want, have) {
			diffs = append(diffs, "volumes")
		}
	}

	var command []string
	switch c := args["command"].(type) {
	case string:
		if c != "" {
			command, _ = splitCommandLine(c)
		}
	case []interface{}:
		for _, w := range c {
			command = append(command, fmt.Sprint(w))
		}
	}
	if command != nil && strings.Join(command, "\x00") != strings.Join(stringList(config["Cmd"]), "\x00") {
		diffs = append(diffs, "command")
	}

	if restart := getStringArg(args, "restart_policy", ""); restart != "" {
		policy, _ := hostConfig["RestartPolicy"].(map[string]interface{})
		current, _ := policy["Name"].(string)
		if current == "" {
			current = "no"
		}
		if restart != current {
			diffs = append(diffs, "restart_policy")
		}
	}

	if networks, ok := args["networks"].([]interface{}); ok {
		settings, _ := inspect["NetworkSettings"].(map[string]interface{})
		attached, _ := settings["Networks"].(map[string]interface{})
		for _, n := range networks {
			if nm, ok := n.(map[string]interface{}); ok {
				n = nm["name"]
			}
			if _, ok := attached[fmt.Sprint(n)]; !ok {
				diffs = append(diffs, "networks")
				break
			}
		}
	}
	// comparisons: {option: ignore} (or '*': ignore) leaves options out
	if cmp, ok := args["comparisons"].(map[string]interface{}); ok {
		kept := diffs[:0]
		for _, d := range diffs {
			rule, ok := cmp[d]
			if !ok {
				rule = cmp["*"]
			}
			if fmt.Sprint(rule) != "ignore" {
				kept = append(kept, d)
			}
		}
		diffs = kept
	}
	sort.Strings(diffs)
	return diffs
}

// normalizePort turns a -p spec into ip:hostport:containerport/proto with
// the defaults filled in ("8080:80" -> "0.0.0.0:8080:80/tcp")
func normalizePort(spec string) string {
	proto := "tcp"
	if i := strings.LastIndex(spec, "/"); i >= 0 {
		spec, proto = spec[:i], spec[i+1:]
	}
	parts := strings.Split(spec, ":")
	ip, hostPort, containerPort := "0.0.0.0", "", parts[len(parts)-1]
	switch len(parts) {
	case 2:
		hostPort = parts[0]
	case 3:
		if parts[0] != "" {
			ip = parts[0]
		}
		hostPort = parts[1]
	}
	return ip + ":" + hostPort + ":" + containerPort + "/" + proto
}

func sameSet(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}
