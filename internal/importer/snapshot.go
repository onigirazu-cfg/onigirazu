// Package importer turns running hosts into a playbook: it collects what
// makes each host differ from a fresh install (packages installed by hand,
// services, accounts, configuration files) and writes roles that recreate
// it.
package importer

import (
	"bufio"
	"context"
	_ "embed"
	"encoding/base64"
	"fmt"
	"sort"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
)

//go:embed collect.sh
var collectScript string

// Snapshot is what one host has beyond a fresh install
type Snapshot struct {
	Host     string
	OS       string // debian | redhat | unknown
	Distro   string // ubuntu 24.04
	Packages []string
	// Units: unit file state and vendor preset; Active: running now
	Units  map[string]Unit
	Active map[string]bool
	Users  []User
	Groups []Group
	Files  []File
	Mounts []Mount
	// Hostname, FQDN and IP as the host knows itself
	Hostname, FQDN, IP string
	// Systemd: systemd runs the host (not a container without it)
	Systemd bool
	// Timezone from /etc/localtime (Europe/Madrid)
	Timezone string
	// Changed are package configuration files edited on the host
	Changed map[string]bool
	// Skipped: what was left out and why
	Skipped []Skip
}

// Unit is a systemd service unit file
type Unit struct {
	State, Preset string
}

// User is an account from getent passwd
type User struct {
	Name          string
	UID, GID      int
	Comment, Home string
	Shell         string
	Groups        []string // supplementary
}

// Group is a group from getent group
type Group struct {
	Name    string
	GID     int
	Members []string
}

// File is a file, directory or link no package owns (or a changed
// package configuration file)
type File struct {
	Path, Kind           string // file | directory | link
	Mode, Owner, Group   string
	UID, GID             int
	Content              []byte
	Target               string // link
	Secret, SecretReason string
	// Secrets are values taken out of the file; Masked is the content with
	// a mark where each was (see secretMark)
	Secrets []SecretValue
	Masked  []byte
}

// SecretValue is a password, token or key found in a configuration file
type SecretValue struct {
	Var, Value string
	Line       int
}

// Mount is an /etc/fstab entry
type Mount struct {
	Src, Path, FSType, Opts string
}

// Skip is something the import leaves out
type Skip struct {
	Path, Reason string
}

// Collect runs the collector on the host through exec (which should have
// become set: configuration files are often root's only)
func Collect(ctx context.Context, host string, exec *executor.CommandExecutor) (*Snapshot, error) {
	res, err := exec.Run(ctx, "sh -c "+shellQuote(collectScript))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", host, err)
	}
	snap, err := Parse(host, res.Stdout)
	if err != nil {
		return nil, fmt.Errorf("%s: %w (stderr: %s)", host, err, strings.TrimSpace(res.Stderr))
	}
	return snap, nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// Parse reads the collector's records
func Parse(host, out string) (*Snapshot, error) {
	s := &Snapshot{Host: host, Units: map[string]Unit{}, Active: map[string]bool{}, Changed: map[string]bool{}}
	sc := bufio.NewScanner(strings.NewReader(out))
	sc.Buffer(make([]byte, 1<<20), 16<<20)
	ended := false
	var last *File
	for sc.Scan() {
		f := strings.Split(sc.Text(), "\t")
		switch f[0] {
		case "OS":
			s.OS = field(f, 1)
			s.Distro = strings.TrimSpace(field(f, 2) + " " + field(f, 3))
		case "PKG":
			if p := strings.TrimSpace(field(f, 1)); p != "" {
				s.Packages = append(s.Packages, p)
			}
		case "UNIT":
			s.Units[field(f, 1)] = Unit{State: field(f, 2), Preset: field(f, 3)}
		case "HOST":
			s.Hostname, s.FQDN, s.IP = field(f, 1), field(f, 2), field(f, 3)
		case "SYSTEMD":
			s.Systemd = true
		case "ACTIVE":
			s.Active[field(f, 1)] = true
		case "PASSWD":
			// name:x:uid:gid:gecos:home:shell
			p := strings.Split(field(f, 1), ":")
			if len(p) == 7 {
				s.Users = append(s.Users, User{Name: p[0], UID: atoi(p[2]), GID: atoi(p[3]), Comment: p[4], Home: p[5], Shell: p[6]})
			}
		case "GROUP":
			p := strings.Split(field(f, 1), ":")
			if len(p) == 4 {
				g := Group{Name: p[0], GID: atoi(p[2])}
				if p[3] != "" {
					g.Members = strings.Split(p[3], ",")
				}
				s.Groups = append(s.Groups, g)
			}
		case "FSTAB":
			m := strings.Fields(field(f, 1))
			if len(m) >= 4 {
				s.Mounts = append(s.Mounts, Mount{Src: m[0], Path: m[1], FSType: m[2], Opts: m[3]})
			}
		case "TZ":
			if i := strings.Index(field(f, 1), "zoneinfo/"); i >= 0 {
				s.Timezone = field(f, 1)[i+len("zoneinfo/"):]
			}
		case "CHANGED":
			s.Changed[field(f, 1)] = true
		case "TREE":
			s.Skipped = append(s.Skipped, Skip{field(f, 1), "application tree with many files (not imported)"})
		case "BIG":
			s.Skipped = append(s.Skipped, Skip{field(f, 1), "larger than 1 MiB (" + field(f, 2) + " bytes)"})
		case "LINK":
			s.Files = append(s.Files, File{Path: field(f, 1), Kind: "link", Target: field(f, 2)})
		case "DIR", "FILE":
			kind := map[string]string{"DIR": "directory", "FILE": "file"}[f[0]]
			s.Files = append(s.Files, File{Path: field(f, 1), Kind: kind, Mode: field(f, 2),
				Owner: field(f, 3), Group: field(f, 4), UID: atoi(field(f, 5)), GID: atoi(field(f, 6))})
			last = &s.Files[len(s.Files)-1]
		case "DATA":
			if last == nil || last.Kind != "file" {
				return nil, fmt.Errorf("file content without a file")
			}
			data, err := base64.StdEncoding.DecodeString(field(f, 1))
			if err != nil {
				return nil, fmt.Errorf("content of %s: %w", last.Path, err)
			}
			last.Content = data
		case "END":
			ended = true
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if !ended {
		return nil, fmt.Errorf("the collector did not finish")
	}
	sort.Strings(s.Packages)
	sort.SliceStable(s.Users, func(i, j int) bool { return s.Users[i].UID < s.Users[j].UID })
	sort.SliceStable(s.Groups, func(i, j int) bool { return s.Groups[i].GID < s.Groups[j].GID })
	sort.Slice(s.Files, func(i, j int) bool { return s.Files[i].Path < s.Files[j].Path })
	for i := range s.Files {
		markSecret(&s.Files[i])
	}
	return s, nil
}

func field(f []string, i int) string {
	if i < len(f) {
		return f[i]
	}
	return ""
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// markSecret flags files whose content must not be written into the
// playbook (private keys: the whole file), and takes the values of
// password, token and key settings out of the others
func markSecret(f *File) {
	if f.Kind != "file" {
		return
	}
	c := string(f.Content)
	if strings.Contains(c, "PRIVATE KEY-----") {
		f.Secret, f.SecretReason = "private key", "contains a private key"
		return
	}
	if !utf8.ValidString(c) || strings.ContainsRune(c, 0) || noSecretFiles[f.Path] {
		return
	}
	lines := strings.SplitAfter(c, "\n")
	used := map[string]int{}
	for i, line := range lines {
		m := secretLine.FindStringSubmatchIndex(line)
		if m == nil {
			continue
		}
		key, sep, value := line[m[4]:m[5]], line[m[6]:m[7]], line[m[8]:m[9]]
		if !strings.ContainsAny(sep, "=:") && !spaceKeys[strings.ToLower(key)] {
			continue
		}
		if len(value) < 4 || placeholderValue.MatchString(value) {
			continue
		}
		name := "secret_" + safeName(strings.TrimPrefix(f.Path, "/")) + "_" + safeName(strings.ToLower(key))
		used[name]++
		if n := used[name]; n > 1 {
			name = fmt.Sprintf("%s_%d", name, n)
		}
		f.Secrets = append(f.Secrets, SecretValue{Var: name, Value: value, Line: i + 1})
		lines[i] = line[:m[8]] + secretMark + name + "\x00" + line[m[9]:]
	}
	if len(f.Secrets) == 0 {
		return
	}
	if jinjaSyntax.MatchString(c) {
		// a template of it would render its own braces
		f.Secrets, f.Secret, f.SecretReason = nil, "credentials", "holds secrets and Jinja-like braces"
		return
	}
	f.Masked = []byte(strings.Join(lines, ""))
}

// noSecretFiles name password databases, not passwords ("passwd: files")
var noSecretFiles = map[string]bool{"/etc/nsswitch.conf": true, "/etc/login.defs": true}

// secretMark starts the mark of a secret in File.Masked; a NUL ends it
const secretMark = "\x00secret:"

// secretLine finds "password = x", "api_key: x", "DB_PASS=x", "token x"
var secretLine = regexp.MustCompile(`(?i)^(\s*(?:export\s+)?["']?)([\w.\-]*(?:password|passwd|secret|token|api[_-]?key|access[_-]?key|auth[_-]?key|credentials?|requirepass|masterauth)[\w.\-]*)(["']?\s*[=:]?\s*["']?)([^\s"'#;,]+)`)

// spaceKeys are settings written "key value" (redis)
var spaceKeys = map[string]bool{"requirepass": true, "masterauth": true}

// placeholderValue: references and empty-ish values, not secrets
var placeholderValue = regexp.MustCompile(`^(\$\{.*|\$\w+|%\(.*|<.*>|\*+|none|null|true|false|yes|no|changeme|required|optional|file:.*|/.*)$`)

// unmark turns the secret marks into template variables
func unmark(b []byte) []byte {
	return secretMarks.ReplaceAll(b, []byte("{{ $1 }}"))
}

var secretMarks = regexp.MustCompile("\x00secret:([A-Za-z0-9_]+)\x00")
