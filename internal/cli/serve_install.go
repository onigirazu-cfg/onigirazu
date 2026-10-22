package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// serveStopMargin is what TimeoutStopSec adds to shutdown_timeout
const serveStopMargin = time.Minute

type serveUnitOptions struct {
	Exec, GlobalConfig, ServeConfig, Addr          string
	User, Group, WorkingDirectory, EnvironmentFile string
	Env, ReadWritePaths                            []string
	ShutdownTimeout                                time.Duration // serve.yml's, with its default
}

func unitQuote(s string, dollars bool) string {
	quote := strings.ContainsAny(s, "\"'\\") || strings.ContainsFunc(s, unicode.IsSpace)
	s = strings.ReplaceAll(s, "%", "%%")
	if dollars {
		s = strings.ReplaceAll(s, "$", "$$")
	}
	if quote {
		s = strings.NewReplacer("\\", "\\\\", "\"", "\\\"").Replace(s)
		return "\"" + s + "\""
	}
	return s
}

func renderServeUnit(o serveUnitOptions) (string, error) {
	values := []string{o.Exec, o.GlobalConfig, o.ServeConfig, o.Addr, o.User, o.Group, o.WorkingDirectory, o.EnvironmentFile}
	values = append(values, o.Env...)
	values = append(values, o.ReadWritePaths...)
	for _, v := range values {
		if strings.ContainsAny(v, "\r\n") {
			return "", fmt.Errorf("unit values must not contain newlines")
		}
	}
	var b strings.Builder
	b.WriteString("[Unit]\nDescription=onigirazu serve: fleet server\nAfter=network-online.target\nWants=network-online.target\n\n[Service]\n")
	for _, field := range []struct{ key, value string }{
		{"User", o.User}, {"Group", o.Group}, {"WorkingDirectory", o.WorkingDirectory}, {"EnvironmentFile", o.EnvironmentFile},
	} {
		// systemd takes these values verbatim (no unquoting); only specifiers need escaping.
		if field.value != "" {
			fmt.Fprintf(&b, "%s=%s\n", field.key, strings.ReplaceAll(field.value, "%", "%%"))
		}
	}
	for _, env := range o.Env {
		if err := validateServeEnv(env); err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "Environment=%s\n", unitQuote(env, false))
	}
	args := []string{o.Exec}
	if o.GlobalConfig != "" {
		args = append(args, "-c", o.GlobalConfig)
	}
	args = append(args, "serve")
	if o.Addr != "" {
		args = append(args, "--addr", o.Addr)
	}
	args = append(args, "--config-file", o.ServeConfig)
	for i := range args {
		args[i] = unitQuote(args[i], true)
	}
	fmt.Fprintf(&b, "ExecStart=%s\nRestart=on-failure\nRestartSec=5s\nNoNewPrivileges=yes\nProtectSystem=strict\n", strings.Join(args, " "))
	// A stop lets running runs finish for shutdown_timeout: SIGTERM goes to
	// serve alone (its children keep working), and systemd waits past the
	// timeout for the cancel and the last results before it kills.
	fmt.Fprintf(&b, "KillMode=mixed\nTimeoutStopSec=%d\n", int((o.ShutdownTimeout + serveStopMargin).Seconds()))
	paths := make([]string, 0, len(o.ReadWritePaths))
	seen := map[string]bool{}
	for _, path := range o.ReadWritePaths {
		if path != "" && !seen[path] {
			seen[path] = true
			paths = append(paths, unitQuote(path, false))
		}
	}
	fmt.Fprintf(&b, "ReadWritePaths=%s\nPrivateTmp=yes\n\n[Install]\nWantedBy=multi-user.target\n", strings.Join(paths, " "))
	return b.String(), nil
}

var serveEnvKey = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func validateServeEnv(entry string) error {
	key, _, ok := strings.Cut(entry, "=")
	if !ok || !serveEnvKey.MatchString(key) || strings.ContainsAny(entry, "\r\n\x00") {
		return fmt.Errorf("environment entry must be KEY=VALUE with a valid key and no newlines or NUL")
	}
	return nil
}

func parseServeEnvFile(path string) ([]string, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- the operator's environment file
	if err != nil {
		return nil, err
	}
	var entries []string
	for i, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if !ok || !serveEnvKey.MatchString(key) || strings.HasPrefix(line, "export ") {
			return nil, fmt.Errorf("environment file line %d: expected KEY=VALUE (export is unsupported)", i+1)
		}
		value, ok = unquoteServeEnvValue(value)
		if !ok {
			return nil, fmt.Errorf("environment file line %d: unmatched quotes", i+1)
		}
		entry := key + "=" + value
		if err := validateServeEnv(entry); err != nil {
			return nil, fmt.Errorf("environment file line %d: invalid entry", i+1)
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// unquoteServeEnvValue follows systemd's environment file rules: single quotes
// are literal, double quotes unescape \" \\ \` \$, unquoted values unescape any
// character.
func unquoteServeEnvValue(value string) (string, bool) {
	if value == "" {
		return value, true
	}
	q := value[0]
	if q != '"' && q != '\'' {
		var b strings.Builder
		for i := 0; i < len(value); i++ {
			if value[i] == '\\' && i+1 < len(value) {
				i++
			}
			b.WriteByte(value[i])
		}
		return b.String(), true
	}
	if len(value) < 2 || value[len(value)-1] != q {
		return "", false
	}
	inner := value[1 : len(value)-1]
	if q == '\'' {
		return inner, !strings.Contains(inner, "'")
	}
	var b strings.Builder
	for i := 0; i < len(inner); i++ {
		c := inner[i]
		if c == '\\' && i+1 < len(inner) && strings.IndexByte("\"\\`$", inner[i+1]) >= 0 {
			i++
			c = inner[i]
		} else if c == '"' {
			return "", false
		}
		b.WriteByte(c)
	}
	return b.String(), true
}

// applyServeEnv returns a restore function for command reuse and tests.
func applyServeEnv(entries []string) (func(), error) {
	previous := map[string]*string{}
	restore := func() {
		for key, value := range previous {
			if value == nil {
				_ = os.Unsetenv(key)
			} else {
				_ = os.Setenv(key, *value)
			}
		}
	}
	for _, entry := range entries {
		if err := validateServeEnv(entry); err != nil {
			restore()
			return nil, err
		}
		key, value, _ := strings.Cut(entry, "=")
		if _, saved := previous[key]; !saved {
			if old, ok := os.LookupEnv(key); ok {
				previous[key] = &old
			} else {
				previous[key] = nil
			}
		}
		if err := os.Setenv(key, value); err != nil {
			restore()
			return nil, fmt.Errorf("set service environment: invalid entry")
		}
	}
	return restore, nil
}

func installServeService(o *serveOptions, out, errOut io.Writer) error {
	name := o.user
	if name == "" {
		name = "root"
	}
	account, err := user.Lookup(name)
	if err != nil {
		return fmt.Errorf("service user: %w", err)
	}
	groupName, gid := o.group, account.Gid
	if groupName == "" && o.user != "" {
		group, err := user.LookupGroupId(gid)
		if err != nil {
			return fmt.Errorf("service group: %w", err)
		}
		groupName = group.Name
	} else if groupName != "" {
		group, err := user.LookupGroup(groupName)
		if err != nil {
			return fmt.Errorf("service group: %w", err)
		}
		gid = group.Gid
	}
	config, err := filepath.Abs(o.config)
	if err != nil {
		return err
	}
	working := o.workingDirectory
	if working == "" {
		working = filepath.Dir(config)
	}
	working, err = filepath.Abs(working)
	if err != nil {
		return err
	}
	info, err := os.Stat(working)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("working directory must be a directory")
	}
	global := configPath
	if global != "" {
		global, err = filepath.Abs(global)
		if err != nil {
			return err
		}
	}
	envFile := o.environmentFile
	var entries []string
	if envFile != "" {
		envFile, err = filepath.Abs(envFile)
		if err != nil {
			return err
		}
		entries, err = parseServeEnvFile(envFile)
		if err != nil {
			return err
		}
	}
	// systemd applies EnvironmentFile after Environment, regardless of line order.
	entries = append(append([]string{}, o.env...), entries...)
	// Validate with the service account's home: data_dir defaults under it.
	entries = append([]string{"HOME=" + account.HomeDir}, entries...)
	restore, err := applyServeEnv(entries)
	if err != nil {
		return err
	}
	defer restore()
	cfg, err := loadServeConfig(config)
	restore()
	if err != nil {
		return err
	}
	cfg.printWarnings(errOut)
	dataDir := cfg.DataDir
	if !filepath.IsAbs(dataDir) {
		dataDir = filepath.Join(working, dataDir)
	}
	paths := []string{dataDir, working}
	// A missing home (system users with /nonexistent) would stop the unit from starting.
	// Install creates it when data_dir is under it.
	if info, err := os.Stat(account.HomeDir); (err == nil && info.IsDir()) || strings.HasPrefix(dataDir, filepath.Clean(account.HomeDir)+string(filepath.Separator)) {
		paths = append(paths, account.HomeDir)
	}
	for _, path := range o.readWritePaths {
		abs, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		paths = append(paths, abs)
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	unit, err := renderServeUnit(serveUnitOptions{
		Exec: self, GlobalConfig: global, ServeConfig: config, Addr: o.addr,
		User: o.user, Group: groupName, WorkingDirectory: working,
		EnvironmentFile: envFile, Env: o.env, ReadWritePaths: paths,
		ShutdownTimeout: cfg.shutdownTimeout(),
	})
	if err != nil {
		return err
	}
	if o.print {
		_, err := io.WriteString(out, unit)
		return err
	}
	if _, err := os.Stat("/run/systemd/system"); err != nil {
		return fmt.Errorf("serve install needs systemd")
	}
	var createdParents []string
	if o.user != "" {
		for dir := filepath.Dir(dataDir); ; dir = filepath.Dir(dir) {
			if _, err := os.Stat(dir); err == nil {
				break
			} else if !os.IsNotExist(err) {
				return err
			}
			createdParents = append(createdParents, dir)
		}
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return err
	}
	if o.user != "" {
		uidNum, err := strconv.Atoi(account.Uid)
		if err != nil {
			return err
		}
		gidNum, err := strconv.Atoi(gid)
		if err != nil {
			return err
		}
		// New private parents must also be traversable by the service user.
		for _, dir := range append(createdParents, dataDir) {
			// Never follow a symlink planted in the path to chown its target.
			if info, err := os.Lstat(dir); err != nil {
				return err
			} else if !info.IsDir() {
				return fmt.Errorf("%s is not a directory", dir)
			}
			if err := os.Lchown(dir, uidNum, gidNum); err != nil {
				return err
			}
		}
	}
	const unitPath = "/etc/systemd/system/onigirazu-serve.service"
	if err := os.WriteFile(unitPath, []byte(unit), 0o644); err != nil { // #nosec G306 -- unit files are world-readable
		return err
	}
	for _, c := range [][]string{{"systemctl", "daemon-reload"}, {"systemctl", "enable", "--now", "onigirazu-serve.service"}} {
		if output, err := exec.Command(c[0], c[1:]...).CombinedOutput(); err != nil { // #nosec G204 -- fixed systemctl arguments
			return fmt.Errorf("%s: %v: %s", strings.Join(c, " "), err, strings.TrimSpace(string(output)))
		}
	}
	fmt.Fprintf(out, "%s installed and started\n", unitPath)
	for _, line := range strings.Split(unit, "\n") {
		if strings.HasPrefix(line, "ExecStart=") {
			fmt.Fprintln(out, line)
		}
	}
	return nil
}
