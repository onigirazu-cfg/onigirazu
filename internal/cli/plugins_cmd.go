package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// Command plugins: "onigirazu NAME args..." runs the executable
// onigirazu-NAME, found in $ONIGIRAZU_PLUGIN_PATH, ~/.onigirazu/plugins,
// next to onigirazu, then in PATH. It gets ONIGIRAZU_BIN, the path of this
// onigirazu, to run playbooks with.

const pluginPrefix = "onigirazu-"

// pluginDirs lists the directories searched before PATH
func pluginDirs() []string {
	var dirs []string
	for _, d := range filepath.SplitList(os.Getenv("ONIGIRAZU_PLUGIN_PATH")) {
		if d != "" {
			dirs = append(dirs, d)
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".onigirazu", "plugins"))
	}
	if self, err := os.Executable(); err == nil {
		dirs = append(dirs, filepath.Dir(self))
	}
	return dirs
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	return runtime.GOOS == "windows" || info.Mode()&0o111 != 0
}

// findPlugin returns the path of the plugin executable for name, or ""
func findPlugin(name string) string {
	if name == "" || strings.ContainsAny(name, `/\`) || strings.HasPrefix(name, "-") {
		return ""
	}
	file := pluginPrefix + name
	if runtime.GOOS == "windows" {
		file += ".exe"
	}
	for _, d := range pluginDirs() {
		if p := filepath.Join(d, file); isExecutable(p) {
			return p
		}
	}
	if p, err := exec.LookPath(file); err == nil {
		return p
	}
	return ""
}

// listPlugins returns name -> path of every plugin found, the first one of a
// name winning
func listPlugins() map[string]string {
	found := map[string]string{}
	dirs := append(pluginDirs(), filepath.SplitList(os.Getenv("PATH"))...)
	for _, d := range dirs {
		entries, err := os.ReadDir(d)
		if err != nil {
			continue
		}
		for _, e := range entries {
			name, ok := strings.CutPrefix(e.Name(), pluginPrefix)
			name = strings.TrimSuffix(name, ".exe")
			if !ok || name == "" {
				continue
			}
			if _, seen := found[name]; seen {
				continue
			}
			if p := filepath.Join(d, e.Name()); isExecutable(p) {
				found[name] = p
			}
		}
	}
	return found
}

// runPlugin runs a plugin with the terminal and returns its exit status as
// an ExitError
func runPlugin(path string, args []string) error {
	cmd := exec.Command(path, args...) // #nosec G204 G702 -- a plugin the user installed, with the user's arguments
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Env = os.Environ()
	if self, err := os.Executable(); err == nil {
		cmd.Env = append(cmd.Env, "ONIGIRAZU_BIN="+self)
	}
	// Ctrl-C reaches the plugin, which decides what to clean up
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	defer signal.Stop(sig)
	err := cmd.Run()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return &ExitError{Code: exitErr.ExitCode()}
	}
	if err != nil {
		return fmt.Errorf("plugin %s: %w", filepath.Base(path), err)
	}
	return nil
}

// pluginFor returns the plugin that handles args, when args[0] is not a
// command of onigirazu
func pluginFor(root *cobra.Command, args []string) string {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return ""
	}
	if c, _, err := root.Find(args); err == nil && c != root {
		return ""
	}
	return findPlugin(args[0])
}

func newPluginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugin",
		Short: "Command plugins: onigirazu NAME runs the executable onigirazu-NAME",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List the command plugins found",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			found := listPlugins()
			names := make([]string, 0, len(found))
			for n := range found {
				names = append(names, n)
			}
			sort.Strings(names)
			out := cmd.OutOrStdout()
			if len(names) == 0 {
				fmt.Fprintf(out, "No plugins found (searched %s and PATH)\n", strings.Join(pluginDirs(), ", "))
				return nil
			}
			for _, n := range names {
				fmt.Fprintf(out, "%-16s %s\n", n, found[n])
			}
			return nil
		},
	})
	return cmd
}
