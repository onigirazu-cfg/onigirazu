package modules

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// ArchiveModule handles file/directory archiving operations
type ArchiveModule struct {
	BaseModule
}

// NewArchiveModule creates a new archive module instance
func NewArchiveModule() *ArchiveModule {
	return &ArchiveModule{
		BaseModule: BaseModule{
			name:        "archive",
			description: "Creates a compressed archive of one or more files or directories",
		},
	}
}

// GetDescription returns the module description
func (m *ArchiveModule) GetDescription() string {
	return m.description
}

// Execute runs the archive module
func (m *ArchiveModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	result := types.TaskResult{
		Host:      host.Name,
		Module:    m.name,
		Timestamp: time.Now(),
		Output:    make(map[string]interface{}),
	}

	startTime := time.Now()
	defer func() {
		result.Duration = time.Since(startTime)
	}()

	// Validate arguments
	if err := m.Validate(args); err != nil {
		result.Failed = true
		result.Error = err.Error()
		return result, err
	}

	// Get arguments
	paths := m.getPaths(args)
	dest := getStringArg(args, "dest", "")
	format := getStringArg(args, "format", "gz")
	removeSources := getBoolArg(args, "remove", false)
	excludePaths := m.getExcludePaths(args)
	_ = getBoolArg(args, "force_archive", false) // force_archive reserved for future use

	flag, ok := map[string]string{"gz": "z", "tgz": "z", "bz2": "j", "xz": "J", "tar": ""}[format]
	if !ok && format != "zip" {
		result.Failed = true
		result.Error = fmt.Sprintf("unsupported format: %s", format)
		return result, fmt.Errorf("%s", result.Error)
	}

	// tar/zip run on the target host; paths are stored relative to /
	var srcs, rels, excludes []string
	for _, p := range paths {
		srcs = append(srcs, shellPathOrGlob(p))
		rels = append(rels, shellPathOrGlob(strings.TrimPrefix(p, "/")))
	}
	var create string
	if format == "zip" {
		for _, e := range excludePaths {
			excludes = append(excludes, "-x "+shellQuote(strings.TrimPrefix(e, "/")))
		}
		create = fmt.Sprintf(`cd / && zip -qr "$dest" %s %s`, strings.Join(rels, " "), strings.Join(excludes, " "))
	} else {
		for _, e := range excludePaths {
			excludes = append(excludes, "--exclude="+shellQuote(strings.TrimPrefix(e, "/")))
		}
		create = fmt.Sprintf(`tar -c%sf "$dest" %s -C / %s`, flag, strings.Join(excludes, " "), strings.Join(rels, " "))
	}
	remove := ""
	if removeSources {
		remove = "rm -rf " + strings.Join(srcs, " ")
	}
	script := fmt.Sprintf(`set -e
dest=%s
srcs=$(ls -d %s 2>/dev/null || true)
if [ -e "$dest" ]; then
  # Up to date: every source is gone (removed after archiving) or older
  if [ -z "$srcs" ] || [ -z "$(find $srcs -newer "$dest" -print -quit 2>/dev/null)" ]; then
    echo unchanged; exit 0
  fi
fi
[ -n "$srcs" ] || { echo "no files match the given paths" >&2; exit 3; }
mkdir -p "$(dirname "$dest")"
%s
%s
echo changed`, shellQuote(dest), strings.Join(srcs, " "), create, remove)

	out, err := runShellOnHost(ctx, host, args, script)
	if err != nil {
		result.Failed = true
		result.Error = fmt.Sprintf("failed to create archive: %v", err)
		return result, err
	}
	result.Changed = strings.TrimSpace(out) == "changed"
	result.Output["dest"] = dest
	result.Output["format"] = format
	result.Success = true
	return result, nil
}

// shellPathOrGlob quotes a path, but leaves simple glob patterns unquoted so
// the shell on the host expands them
func shellPathOrGlob(p string) string {
	if strings.ContainsAny(p, "*?[") && safeGlob.MatchString(p) {
		return p
	}
	return shellQuote(p)
}

var safeGlob = regexp.MustCompile(`^[A-Za-z0-9_./*?\[\]-]+$`)

func (m *ArchiveModule) Validate(args map[string]interface{}) error {
	if err := m.BaseModule.Validate(args); err != nil {
		return err
	}

	// Check required arguments
	paths := m.getPaths(args)
	if len(paths) == 0 {
		return fmt.Errorf("'path' is required (can be string or list)")
	}

	if dest, ok := args["dest"]; !ok || dest == "" {
		return fmt.Errorf("'dest' is required")
	}

	// Validate format
	format := getStringArg(args, "format", "gz")
	validFormats := map[string]bool{"tar": true, "gz": true, "bz2": true, "xz": true, "zip": true}
	if !validFormats[format] {
		return fmt.Errorf("invalid format '%s', must be one of: tar, gz, bz2, xz, zip", format)
	}

	return nil
}

// getPaths extracts path argument (can be string or list)
func (m *ArchiveModule) getPaths(args map[string]interface{}) []string {
	var paths []string

	if pathArg, ok := args["path"]; ok {
		switch v := pathArg.(type) {
		case string:
			if v != "" {
				paths = append(paths, v)
			}
		case []interface{}:
			for _, p := range v {
				if str, ok := p.(string); ok && str != "" {
					paths = append(paths, str)
				}
			}
		}
	}

	return paths
}

// getExcludePaths extracts exclude_path argument
func (m *ArchiveModule) getExcludePaths(args map[string]interface{}) []string {
	var excludePaths []string

	if excludeArg, ok := args["exclude_path"]; ok {
		switch v := excludeArg.(type) {
		case string:
			if v != "" {
				excludePaths = append(excludePaths, v)
			}
		case []interface{}:
			for _, p := range v {
				if str, ok := p.(string); ok && str != "" {
					excludePaths = append(excludePaths, str)
				}
			}
		}
	}

	return excludePaths
}

// collectFiles collects all files matching the paths (with glob support)
