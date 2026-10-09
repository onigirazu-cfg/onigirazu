package modules

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// verify: goss-style checks of the host's state, all evaluated by one shell
// script in one round trip. A play's verify: section becomes this task after
// its tasks and handlers; `onigirazu verify` runs only these.
//
//	verify:
//	  - file: {path: /etc/nginx/nginx.conf, mode: "0644", owner: root, contains: ["worker_processes", "/^user +www-data;/"]}
//	  - file: {path: /var/log/old.log, exists: false}
//	  - package: {name: nginx, installed: true}
//	  - service: {name: nginx, running: true, enabled: true}
//	  - port: {port: 443, listening: true}            # proto tcp|udp, ip
//	  - process: {name: nginx, running: true}
//	  - user: {name: deploy, exists: true, groups: [docker], shell: /bin/bash}
//	  - group: {name: docker, exists: true}
//	  - command: {cmd: "nginx -t", exit_status: 0, stderr: ["syntax is ok"]}
//	  - http: {url: https://localhost/health, status: 200, body: ["ok"], insecure: true}
//	  - mount: {path: /data, exists: true, type: ext4, opts: [noexec]}
//	  - kernel_param: {name: net.ipv4.ip_forward, value: "1"}
//	  - dns: {name: db.internal, resolves: true}
//
// contains/body/stderr/stdout patterns: a plain string, or /regexp/.

type VerifyModule struct {
	*BaseModule
}

func NewVerifyModule() *VerifyModule {
	return &VerifyModule{BaseModule: NewBaseModule("verify")}
}

func (m *VerifyModule) GetDescription() string {
	return "Check the host's state (files, packages, services, ports, users, commands, http, mounts) in one round trip"
}

// verifyCheck is one check as written, with its number and a label
type verifyCheck struct {
	n     int
	kind  string
	label string
	args  map[string]interface{}
}

func (m *VerifyModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	start := time.Now()
	result := types.TaskResult{TaskName: taskName(args), Host: host.Name, Module: m.name, Timestamp: start, Success: true, Output: map[string]interface{}{}}
	checks, err := verifyChecks(args["checks"])
	if err != nil {
		result.Success, result.Failed, result.Error = false, true, err.Error()
		return result, nil
	}
	script, err := verifyScript(checks)
	if err != nil {
		result.Success, result.Failed, result.Error = false, true, err.Error()
		return result, nil
	}
	out, _ := runShellOnHost(ctx, host, args, script) // every check reports; the exit code is the last one's
	report, passed, failed := parseVerifyOutput(checks, out)
	result.Output["checks"], result.Output["passed"], result.Output["failed"] = report, passed, failed
	result.Output["msg"] = fmt.Sprintf("%d check(s) passed, %d failed", passed, failed)
	if failed > 0 {
		result.Success, result.Failed = false, true
		var bad []string
		for _, r := range report {
			if passed, _ := r["ok"].(bool); !passed {
				bad = append(bad, fmt.Sprintf("%s: %s", r["check"], r["detail"]))
			}
		}
		result.Error = fmt.Sprintf("%d check(s) failed: %s", failed, strings.Join(bad, "; "))
	}
	result.Duration = time.Since(start)
	return result, nil
}

func (m *VerifyModule) Validate(args map[string]interface{}) error {
	_, err := verifyChecks(args["checks"])
	return err
}

var verifyKinds = map[string]bool{"file": true, "package": true, "service": true, "port": true, "process": true,
	"user": true, "group": true, "command": true, "http": true, "mount": true, "kernel_param": true, "dns": true}

// verifyChecks reads the checks list: [{file: {...}}, {service: {...}}, ...]
func verifyChecks(v interface{}) ([]verifyCheck, error) {
	var list []interface{}
	switch l := v.(type) {
	case []interface{}:
		list = l
	case []map[string]interface{}: // a play's verify: section, typed by the parser
		for _, m := range l {
			list = append(list, m)
		}
	}
	if len(list) == 0 {
		return nil, fmt.Errorf("verify: checks is a list of {kind: {...}}")
	}
	var checks []verifyCheck
	for i, item := range list {
		m, ok := item.(map[string]interface{})
		if !ok || len(m) != 1 {
			return nil, fmt.Errorf("verify: check %d: one kind per entry (file, package, service, port, process, user, group, command, http, mount, kernel_param, dns)", i+1)
		}
		for kind, a := range m {
			if !verifyKinds[kind] {
				return nil, fmt.Errorf("verify: check %d: unknown kind %q", i+1, kind)
			}
			args, _ := a.(map[string]interface{})
			if args == nil {
				return nil, fmt.Errorf("verify: check %d: %s needs its arguments", i+1, kind)
			}
			key := []string{"path", "name", "port", "cmd", "url"}
			label := kind
			for _, k := range key {
				if s, ok := args[k]; ok {
					label = fmt.Sprintf("%s %v", kind, s)
					break
				}
			}
			checks = append(checks, verifyCheck{n: i + 1, kind: kind, label: label, args: args})
		}
	}
	return checks, nil
}

// verifyScript renders the checks as one POSIX sh script. Every check prints
// "N\tok|fail\tdetail"; nothing a check does can stop the next one.
func verifyScript(checks []verifyCheck) (string, error) {
	var b strings.Builder
	b.WriteString(`#!/bin/sh
ok() { printf '%s\tok\t%s\n' "$1" "$2"; }
fail() { printf '%s\tfail\t%s\n' "$1" "$2"; }
# has FILE PATTERN: a /regexp/ is extended grep, anything else is fixed
has() { case "$2" in /*/) grep -Eq -- "$(printf '%s' "$2" | sed 's|^/||; s|/$||')" "$1" ;; *) grep -Fq -- "$2" "$1" ;; esac; }
`)
	for _, c := range checks {
		s, err := verifyCheckScript(c)
		if err != nil {
			return "", err
		}
		b.WriteString("( " + s + " ) 2>/dev/null\n")
	}
	return b.String(), nil
}

func vq(v interface{}) string { return shellQuote(fmt.Sprint(v)) }

// wantBool reads a boolean argument with its default
func wantBool(args map[string]interface{}, key string, def bool) bool {
	if v, ok := args[key]; ok {
		if b, ok := parseBool(v); ok {
			return b
		}
	}
	return def
}

func patterns(args map[string]interface{}, key string) []string {
	switch v := args[key].(type) {
	case string:
		return []string{v}
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, p := range v {
			out = append(out, fmt.Sprint(p))
		}
		return out
	}
	return nil
}

// verifyContains renders checks of file $f against the patterns of key
func verifyContains(n int, args map[string]interface{}, key, file string) string {
	var b strings.Builder
	for _, p := range patterns(args, key) {
		fmt.Fprintf(&b, "has %s %s || { fail %d %s; exit 0; }; ", file, vq(p), n, vq(key+" lacks "+p))
	}
	return b.String()
}

func verifyCheckScript(c verifyCheck) (string, error) {
	n, a := c.n, c.args
	switch c.kind {
	case "file":
		p := vq(a["path"])
		if a["path"] == nil {
			return "", fmt.Errorf("verify: file: path is required")
		}
		var b strings.Builder
		if !wantBool(a, "exists", true) {
			fmt.Fprintf(&b, "[ ! -e %s ] && ok %d exists=false || fail %d 'exists'", p, n, n)
			return b.String(), nil
		}
		fmt.Fprintf(&b, "[ -e %s ] || [ -L %s ] || { fail %d 'missing'; exit 0; }; ", p, p, n)
		switch t, _ := a["type"].(string); t {
		case "file":
			fmt.Fprintf(&b, "[ -f %s ] || { fail %d 'not a file'; exit 0; }; ", p, n)
		case "directory":
			fmt.Fprintf(&b, "[ -d %s ] || { fail %d 'not a directory'; exit 0; }; ", p, n)
		case "link", "symlink":
			fmt.Fprintf(&b, "[ -L %s ] || { fail %d 'not a symlink'; exit 0; }; ", p, n)
		}
		if t, ok := a["target"]; ok {
			fmt.Fprintf(&b, "[ \"$(readlink %s)\" = %s ] || { fail %d \"target $(readlink %s)\"; exit 0; }; ", p, vq(t), n, p)
		}
		if mode, ok := a["mode"]; ok {
			want := strings.TrimLeft(fmt.Sprint(mode), "0")
			if want == "" {
				want = "0"
			}
			fmt.Fprintf(&b, "m=$(stat -c %%a %s 2>/dev/null || stat -f %%Lp %s); [ \"${m#0}\" = %s ] || [ \"$m\" = %s ] || { fail %d \"mode $m\"; exit 0; }; ", p, p, vq(want), vq(want), n)
		}
		if o, ok := a["owner"]; ok {
			fmt.Fprintf(&b, "o=$(stat -c %%U %s 2>/dev/null || stat -f %%Su %s); [ \"$o\" = %s ] || { fail %d \"owner $o\"; exit 0; }; ", p, p, vq(o), n)
		}
		if g, ok := a["group"]; ok {
			fmt.Fprintf(&b, "g=$(stat -c %%G %s 2>/dev/null || stat -f %%Sg %s); [ \"$g\" = %s ] || { fail %d \"group $g\"; exit 0; }; ", p, p, vq(g), n)
		}
		b.WriteString(verifyContains(n, a, "contains", p))
		fmt.Fprintf(&b, "ok %d ''", n)
		return b.String(), nil
	case "package":
		name := vq(a["name"])
		installed := wantBool(a, "installed", true)
		s := fmt.Sprintf("if dpkg-query -W -f='${Status}' %s 2>/dev/null | grep -q 'install ok installed' || rpm -q %s >/dev/null 2>&1 || pacman -Q %s >/dev/null 2>&1; then i=1; else i=0; fi; ", name, name, name)
		if !installed {
			return s + fmt.Sprintf("[ $i = 0 ] && ok %d absent || fail %d installed", n, n), nil
		}
		s += fmt.Sprintf("[ $i = 1 ] || { fail %d 'not installed'; exit 0; }; ", n)
		if v, ok := a["version"]; ok {
			s += fmt.Sprintf("ver=$(dpkg-query -W -f='${Version}' %s 2>/dev/null || rpm -q --qf '%%{VERSION}-%%{RELEASE}' %s 2>/dev/null); case \"$ver\" in %s*) ;; *) fail %d \"version $ver\"; exit 0 ;; esac; ", name, name, fmt.Sprint(v), n)
		}
		return s + fmt.Sprintf("ok %d ''", n), nil
	case "service":
		name := vq(a["name"])
		var b strings.Builder
		if v, ok := a["running"]; ok {
			want, _ := parseBool(v)
			fmt.Fprintf(&b, "st=$(systemctl is-active %s 2>/dev/null) || { service %s status >/dev/null 2>&1 && st=active || st=inactive; }; ", name, name)
			if want {
				fmt.Fprintf(&b, "[ \"$st\" = active ] || { fail %d \"$st\"; exit 0; }; ", n)
			} else {
				fmt.Fprintf(&b, "[ \"$st\" != active ] || { fail %d running; exit 0; }; ", n)
			}
		}
		if v, ok := a["enabled"]; ok {
			want, _ := parseBool(v)
			// a socket-activated service is enabled through its socket
			fmt.Fprintf(&b, "en=$(systemctl is-enabled %s 2>/dev/null); case \"$en\" in enabled|static|alias|indirect) ;; *) s=$(systemctl is-enabled %s.socket 2>/dev/null); [ \"$s\" = enabled ] && en=enabled ;; esac; ", name, name)
			if want {
				fmt.Fprintf(&b, "case \"$en\" in enabled|static|alias|indirect) ;; *) fail %d \"${en:-not enabled}\"; exit 0 ;; esac; ", n)
			} else {
				fmt.Fprintf(&b, "case \"$en\" in enabled) fail %d enabled; exit 0 ;; esac; ", n)
			}
		}
		fmt.Fprintf(&b, "ok %d ''", n)
		return b.String(), nil
	case "port":
		port := fmt.Sprint(a["port"])
		if _, err := strconv.Atoi(port); err != nil {
			return "", fmt.Errorf("verify: port: port %q", port)
		}
		flag := "-t"
		if strings.EqualFold(fmt.Sprint(a["proto"]), "udp") {
			flag = "-u"
		}
		ip := ""
		if v, ok := a["ip"]; ok {
			ip = fmt.Sprint(v)
		}
		match := fmt.Sprintf("[:.]%s$", port)
		if ip != "" {
			match = fmt.Sprintf("^%s:%s$", strings.ReplaceAll(ip, ".", `\.`), port)
		}
		// ss, netstat, and /proc/net (containers without either): hex port, state 0A = LISTEN
		proc := "/proc/net/tcp /proc/net/tcp6"
		if flag == "-u" {
			proc = "/proc/net/udp /proc/net/udp6"
		}
		portHex, _ := strconv.Atoi(port)
		s := fmt.Sprintf("if ss -Hln %s 2>/dev/null | awk '{print $4}' | grep -Eq %s || netstat -ln 2>/dev/null | awk '{print $4}' | grep -Eq %s || grep -qiE '^ *[0-9]+: [0-9A-F]*:%04X [0-9A-F]+:[0-9A-F]+ 0A' %s 2>/dev/null; then l=1; else l=0; fi; ", flag, shellQuote(match), shellQuote(match), portHex, proc)
		if wantBool(a, "listening", true) {
			return s + fmt.Sprintf("[ $l = 1 ] && ok %d '' || fail %d 'not listening'", n, n), nil
		}
		return s + fmt.Sprintf("[ $l = 0 ] && ok %d '' || fail %d listening", n, n), nil
	case "process":
		name := vq(a["name"])
		if wantBool(a, "running", true) {
			return fmt.Sprintf("pgrep -x %s >/dev/null 2>&1 && ok %d '' || fail %d 'not running'", name, n, n), nil
		}
		return fmt.Sprintf("pgrep -x %s >/dev/null 2>&1 && fail %d running || ok %d ''", name, n, n), nil
	case "user":
		name := vq(a["name"])
		if !wantBool(a, "exists", true) {
			return fmt.Sprintf("getent passwd %s >/dev/null 2>&1 && fail %d exists || ok %d ''", name, n, n), nil
		}
		var b strings.Builder
		fmt.Fprintf(&b, "e=$(getent passwd %s) || { fail %d 'no such user'; exit 0; }; ", name, n)
		for key, field := range map[string]int{"uid": 3, "gid": 4, "home": 6, "shell": 7} {
			if v, ok := a[key]; ok {
				fmt.Fprintf(&b, "f=$(printf '%%s' \"$e\" | cut -d: -f%d); [ \"$f\" = %s ] || { fail %d \"%s $f\"; exit 0; }; ", field, vq(v), n, key)
			}
		}
		for _, g := range patterns(a, "groups") {
			fmt.Fprintf(&b, "id -Gn %s 2>/dev/null | tr ' ' '\\n' | grep -qx %s || { fail %d %s; exit 0; }; ", name, vq(g), n, vq("not in group "+g))
		}
		fmt.Fprintf(&b, "ok %d ''", n)
		return b.String(), nil
	case "group":
		name := vq(a["name"])
		if !wantBool(a, "exists", true) {
			return fmt.Sprintf("getent group %s >/dev/null 2>&1 && fail %d exists || ok %d ''", name, n, n), nil
		}
		s := fmt.Sprintf("e=$(getent group %s) || { fail %d 'no such group'; exit 0; }; ", name, n)
		if v, ok := a["gid"]; ok {
			s += fmt.Sprintf("f=$(printf '%%s' \"$e\" | cut -d: -f3); [ \"$f\" = %s ] || { fail %d \"gid $f\"; exit 0; }; ", vq(v), n)
		}
		return s + fmt.Sprintf("ok %d ''", n), nil
	case "command":
		cmd, _ := a["cmd"].(string)
		if cmd == "" {
			return "", fmt.Errorf("verify: command: cmd is required")
		}
		timeout := 30
		if v, ok := toInt(a["timeout"]); ok && v > 0 {
			timeout = v
		}
		want := 0
		if v, ok := toInt(a["exit_status"]); ok {
			want = v
		}
		var b strings.Builder
		fmt.Fprintf(&b, "o=$(mktemp); r=$(mktemp); timeout %d sh -c %s >\"$o\" 2>\"$r\"; rc=$?; ", timeout, vq(cmd))
		fmt.Fprintf(&b, "[ $rc = %d ] || { fail %d \"exit $rc: $(head -c 200 \"$r\" | tr '\\n' ' ')\"; rm -f \"$o\" \"$r\"; exit 0; }; ", want, n)
		b.WriteString(verifyContains(n, a, "stdout", `"$o"`))
		b.WriteString(verifyContains(n, a, "stderr", `"$r"`))
		fmt.Fprintf(&b, "rm -f \"$o\" \"$r\"; ok %d ''", n)
		return b.String(), nil
	case "http":
		u, _ := a["url"].(string)
		if u == "" {
			return "", fmt.Errorf("verify: http: url is required")
		}
		status := 200
		if v, ok := toInt(a["status"]); ok {
			status = v
		}
		insecure := ""
		if wantBool(a, "insecure", false) {
			insecure = "-k "
		}
		timeout := 10
		if v, ok := toInt(a["timeout"]); ok && v > 0 {
			timeout = v
		}
		var b strings.Builder
		fmt.Fprintf(&b, "o=$(mktemp); code=$(curl -sS %s-m %d -o \"$o\" -w '%%{http_code}' %s 2>/dev/null) || { fail %d 'no answer'; rm -f \"$o\"; exit 0; }; ", insecure, timeout, vq(u), n)
		fmt.Fprintf(&b, "[ \"$code\" = %d ] || { fail %d \"status $code\"; rm -f \"$o\"; exit 0; }; ", status, n)
		b.WriteString(verifyContains(n, a, "body", `"$o"`))
		fmt.Fprintf(&b, "rm -f \"$o\"; ok %d ''", n)
		return b.String(), nil
	case "mount":
		p := vq(a["path"])
		if !wantBool(a, "exists", true) {
			return fmt.Sprintf("findmnt -n %s >/dev/null 2>&1 && fail %d mounted || ok %d ''", p, n, n), nil
		}
		var b strings.Builder
		fmt.Fprintf(&b, "fm=$(findmnt -n -o FSTYPE,OPTIONS %s 2>/dev/null) || { fail %d 'not mounted'; exit 0; }; ", p, n)
		if t, ok := a["type"]; ok {
			fmt.Fprintf(&b, "[ \"${fm%%%% *}\" = %s ] || { fail %d \"type ${fm%%%% *}\"; exit 0; }; ", vq(t), n)
		}
		for _, o := range patterns(a, "opts") {
			fmt.Fprintf(&b, "printf '%%s' \"${fm#* }\" | tr ',' '\\n' | grep -qx %s || { fail %d %s; exit 0; }; ", vq(o), n, vq("option "+o+" missing"))
		}
		fmt.Fprintf(&b, "ok %d ''", n)
		return b.String(), nil
	case "kernel_param":
		return fmt.Sprintf("v=$(sysctl -n %s 2>/dev/null); [ \"$v\" = %s ] && ok %d '' || fail %d \"value ${v:-unset}\"", vq(a["name"]), vq(a["value"]), n, n), nil
	case "dns":
		name := vq(a["name"])
		if !wantBool(a, "resolves", true) {
			return fmt.Sprintf("getent hosts %s >/dev/null 2>&1 && fail %d resolves || ok %d ''", name, n, n), nil
		}
		s := fmt.Sprintf("r=$(getent hosts %s) || { fail %d 'does not resolve'; exit 0; }; ", name, n)
		for _, addr := range patterns(a, "addrs") {
			s += fmt.Sprintf("printf '%%s\\n' \"$r\" | awk '{print $1}' | grep -qx %s || { fail %d %s; exit 0; }; ", vq(addr), n, vq("not "+addr))
		}
		return s + fmt.Sprintf("ok %d ''", n), nil
	}
	return "", fmt.Errorf("verify: unknown kind %s", c.kind)
}

// parseVerifyOutput matches the script's lines back to the checks; a check
// that printed nothing (the shell died) counts as failed
func parseVerifyOutput(checks []verifyCheck, out string) (report []map[string]interface{}, passed, failed int) {
	status := map[int][2]string{}
	for _, line := range strings.Split(out, "\n") {
		f := strings.SplitN(line, "\t", 3)
		if len(f) < 2 {
			continue
		}
		n, err := strconv.Atoi(f[0])
		if err != nil {
			continue
		}
		detail := ""
		if len(f) == 3 {
			detail = f[2]
		}
		status[n] = [2]string{f[1], detail}
	}
	for _, c := range checks {
		st, seen := status[c.n]
		ok := seen && st[0] == "ok"
		detail := st[1]
		if !seen {
			detail = "no result"
		}
		if ok {
			passed++
		} else {
			failed++
		}
		report = append(report, map[string]interface{}{"check": c.label, "kind": c.kind, "ok": ok, "detail": detail})
	}
	return report, passed, failed
}
