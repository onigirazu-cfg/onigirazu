package modules

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// FindModule searches for files matching patterns
type FindModule struct {
	*BaseModule
}

// NewFindModule creates a new find module instance
func NewFindModule() *FindModule {
	return &FindModule{
		BaseModule: NewBaseModule("find"),
	}
}

// GetDescription returns the module description
func (m *FindModule) GetDescription() string {
	return "Search for files matching patterns"
}

// Execute runs the find module
func (m *FindModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()

	result := types.TaskResult{
		Host:      host.Name,
		Module:    m.name,
		Timestamp: startTime,
		Output:    make(map[string]interface{}),
	}

	// Validate arguments
	if err := m.Validate(args); err != nil {
		result.Failed = true
		result.Error = err.Error()
		result.Duration = time.Since(startTime)
		return result, err
	}

	// Ansible spells them paths/patterns
	// a list or a comma separated string, as in Ansible
	paths := listArg(args, "paths", "path")
	if len(paths) == 0 {
		paths = []string{"."}
	}
	patterns := listArg(args, "patterns", "pattern")
	if len(patterns) == 0 {
		patterns = []string{"*"}
	}
	recurse := getBoolArg(args, "recurse", false)
	// Ansible calls it file_type
	fileType := getStringArg(args, "file_type", getStringArg(args, "type", ""))
	limit := getIntArg(args, "limit", 0)

	// Initialize executor for remote execution
	exec, err := executor.NewCommandExecutor(host)
	if err != nil {
		result.Failed = true
		result.Error = fmt.Sprintf("failed to create executor: %v", err)
		result.Duration = time.Since(startTime)
		return result, err
	}
	defer exec.Close()

	sel, err := newFindSelection(args, patterns)
	if err != nil {
		result.Failed = true
		result.Error = err.Error()
		result.Duration = time.Since(startTime)
		return result, err
	}
	findPatterns := patterns
	if sel.useRegex {
		findPatterns = []string{"*"} // regexes are matched here, not by find
	}

	// Find files matching pattern
	found, err := m.findFiles(exec, paths, findPatterns, fileType, sel.depth, recurse)
	if err != nil {
		result.Failed = true
		result.Error = fmt.Sprintf("find failed: %v", err)
		result.Duration = time.Since(startTime)
		return result, err
	}
	files := make([]map[string]interface{}, 0, len(found))
	now := float64(time.Now().UnixNano()) / 1e9
	for _, f := range found {
		if sel.keep(f, now) {
			files = append(files, f)
			if limit > 0 && len(files) == limit {
				break
			}
		}
	}

	result.Success = true
	result.Changed = false // find never changes anything
	result.Output["files"] = files
	result.Output["file_count"] = len(files)
	result.Output["matched"] = len(files)
	result.Duration = time.Since(startTime)

	return result, nil
}

// findFiles searches for files matching the pattern
func (m *FindModule) findFiles(exec *executor.CommandExecutor, paths, patterns []string, fileType string, maxDepth int, recurse bool) ([]map[string]interface{}, error) {
	var files []map[string]interface{}

	// Like Ansible, only the given directory unless recurse is set (then down
	// to depth levels); never the given directory itself
	depth := "-mindepth 1 -maxdepth 1 "
	if recurse {
		depth = "-mindepth 1 "
		if maxDepth > 0 {
			depth += fmt.Sprintf("-maxdepth %d ", maxDepth)
		}
	}
	quoted := make([]string, len(paths))
	for i, p := range paths {
		quoted[i] = shellQuote(p)
	}
	names := make([]string, len(patterns))
	for i, p := range patterns {
		names[i] = "-name " + shellQuote(p)
	}
	// one command for the list and the stats, a line per file: "type size
	// mode mtime path"; GNU find prints them itself, elsewhere one stat runs
	// for the whole list
	typeFlag := "-type " + m.getTypeFlag(fileType) + " "
	if fileType == "any" {
		typeFlag = ""
	}
	sel := fmt.Sprintf("%s %s%s\\( %s \\)", strings.Join(quoted, " "), depth, typeFlag, strings.Join(names, " -o "))
	cmd := fmt.Sprintf(`if find /dev/null -maxdepth 0 -printf '' 2>/dev/null; then
  find %s -printf '%%y %%s %%m %%T@ %%p\n' 2>/dev/null
else
  find %s -print 2>/dev/null | tr '\n' '\0' | xargs -0 stat -f '%%p %%z %%Lp %%m %%N' 2>/dev/null || true
fi`, sel, sel)

	output, err := exec.Execute(cmd)
	if err != nil {
		// If path doesn't exist, return empty list
		if strings.Contains(err.Error(), "No such file") || strings.Contains(err.Error(), "no such file") {
			return files, nil
		}
		return nil, fmt.Errorf("find command failed: %v", err)
	}
	for _, rec := range strings.Split(output, "\n") {
		if info, ok := parseFindRecord(rec); ok {
			files = append(files, info)
		}
	}
	return files, nil
}

// parseFindRecord reads "type size mode mtime path" into Ansible's fields;
// the type is a find -printf %y letter or a BSD stat %p octal st_mode
func parseFindRecord(rec string) (map[string]interface{}, bool) {
	f := strings.SplitN(rec, " ", 5)
	if len(f) != 5 || f[4] == "" {
		return nil, false
	}
	kind := "other"
	switch f[0] {
	case "d":
		kind = "directory"
	case "f":
		kind = "file"
	case "l":
		kind = "link"
	default:
		if st, err := strconv.ParseUint(f[0], 8, 32); err == nil {
			switch st >> 12 {
			case 0o04:
				kind = "directory"
			case 0o10:
				kind = "file"
			case 0o12:
				kind = "link"
			}
		}
	}
	path := f[4]
	mode := f[2]
	for len(mode) < 4 {
		mode = "0" + mode
	}
	info := map[string]interface{}{
		"path": path, "name": filepath.Base(path), "type": kind,
		"isdir": kind == "directory", "isreg": kind == "file", "isfile": kind == "file", "islnk": kind == "link", "islink": kind == "link",
		"mode": mode,
	}
	if size, err := strconv.ParseInt(f[1], 10, 64); err == nil {
		info["size"] = size
	}
	if mtime, err := strconv.ParseFloat(f[3], 64); err == nil {
		info["mtime"] = mtime
	}
	return info, true
}

// getTypeFlag returns the find -type flag value
func (m *FindModule) getTypeFlag(fileType string) string {
	switch fileType {
	case "file":
		return "f"
	case "directory":
		return "d"
	case "link":
		return "l"
	case "socket":
		return "s"
	case "pipe":
		return "p"
	case "block":
		return "b"
	case "char":
		return "c"
	default:
		return "f" // Default to files
	}
}

// escapeSingleQuotes escapes single quotes in pattern
func escapeSingleQuotes(pattern string) string {
	return strings.ReplaceAll(pattern, "'", "'\\''")
}

// Validate checks if the provided arguments are valid
func (m *FindModule) Validate(args map[string]interface{}) error {
	// path is optional, defaults to "."
	if path, ok := args["path"].(string); ok && path == "" {
		return fmt.Errorf("'path' must not be empty")
	}

	// pattern is optional, defaults to "*"
	if pattern, ok := args["pattern"].(string); ok && pattern == "" {
		return fmt.Errorf("'pattern' must not be empty")
	}

	// type is optional
	fileType, ok := args["file_type"].(string)
	if !ok {
		fileType, ok = args["type"].(string)
	}
	if ok {
		validTypes := []string{"any", "file", "directory", "link", "socket", "pipe", "block", "char", ""}
		validType := false
		for _, t := range validTypes {
			if fileType == t {
				validType = true
				break
			}
		}
		if !validType {
			return fmt.Errorf("invalid type '%s': must be one of any, file, directory, link, socket, pipe, block, char", fileType)
		}
	}

	// limit is optional, must be non-negative
	if limit, ok := toInt(args["limit"]); ok && limit < 0 {
		return fmt.Errorf("'limit' must be non-negative")
	}

	return nil
}

// listArg reads a list argument given as a list or a comma separated string,
// from the first of the keys that is set
func listArg(args map[string]interface{}, keys ...string) []string {
	for _, k := range keys {
		switch v := args[k].(type) {
		case string:
			var out []string
			for _, part := range strings.Split(v, ",") {
				if part = strings.TrimSpace(part); part != "" {
					out = append(out, part)
				}
			}
			return out
		case []interface{}:
			out := make([]string, 0, len(v))
			for _, item := range v {
				out = append(out, fmt.Sprint(item))
			}
			return out
		}
	}
	return nil
}
