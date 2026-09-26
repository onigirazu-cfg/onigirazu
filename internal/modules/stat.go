package modules

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// StatModule retrieves file or directory status
type StatModule struct {
	*BaseModule
}

func NewStatModule() *StatModule {
	return &StatModule{
		BaseModule: NewBaseModule("stat"),
	}
}

func (m *StatModule) GetDescription() string {
	return "Retrieves file or directory status"
}

func (m *StatModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()

	result := types.TaskResult{
		TaskName:  taskName(args),
		Host:      host.Name,
		Module:    m.name,
		Timestamp: startTime,
	}

	// Validate arguments
	if err := m.Validate(args); err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Duration = time.Since(startTime)
		return result, err
	}

	path, _ := args["path"].(string)
	stat, err := statOnHost(ctx, host, args, path)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to stat %s: %v", path, err)
		result.Duration = time.Since(startTime)
		return result, nil
	}
	// the fields also at the top level, as before
	output := map[string]interface{}{"stat": stat}
	for k, v := range stat {
		output[k] = v
	}
	result.Success = true
	result.Changed = false // stat never changes anything
	result.Output = output
	result.Duration = time.Since(startTime)
	return result, nil
}

// statScript prints key=value lines about a path: GNU stat and sha*sum
// first, BSD stat and shasum as fallbacks. A link is reported as a link
// (follow: false, as in Ansible).
const statScript = `p=%s; algo=%s
if [ ! -e "$p" ] && [ ! -L "$p" ]; then echo exists=false; exit 0; fi
echo exists=true
if [ -L "$p" ]; then echo type=link; echo lnk_source="$(readlink -f "$p" 2>/dev/null || readlink "$p")"; echo lnk_target="$(readlink "$p")"
elif [ -d "$p" ]; then echo type=directory; elif [ -f "$p" ]; then echo type=file; else echo type=other; fi
stat --printf 'size=%%s\nmode=%%a\nmtime=%%Y\natime=%%X\nctime=%%Z\nuid=%%u\ngid=%%g\npw_name=%%U\ngr_name=%%G\ninode=%%i\nnlink=%%h\n' "$p" 2>/dev/null ||
  stat -f 'size=%%z%%nmode=%%Lp%%nmtime=%%m%%natime=%%a%%nctime=%%c%%nuid=%%u%%ngid=%%g%%npw_name=%%Su%%ngr_name=%%Sg%%ninode=%%i%%nnlink=%%l' "$p"
if [ -n "$algo" ] && [ -f "$p" ] && [ ! -L "$p" ]; then
  s=$( ("${algo}sum" "$p" 2>/dev/null || shasum -a "${algo#sha}" "$p" 2>/dev/null || md5 -q "$p") | cut -d' ' -f1)
  echo checksum="$s"
fi`

// statOnHost reads what Ansible's stat returns about path on the host
func statOnHost(ctx context.Context, host types.Host, args map[string]interface{}, path string) (map[string]interface{}, error) {
	algo := ""
	if getBoolArg(args, "get_checksum", true) {
		algo = getStringArg(args, "checksum_algorithm", "sha1")
		switch algo {
		case "md5", "sha1", "sha224", "sha256", "sha384", "sha512":
		default:
			return nil, fmt.Errorf("unsupported checksum_algorithm %q", algo)
		}
	}
	out, err := runShellOnHost(ctx, host, args, fmt.Sprintf(statScript, shellQuote(path), shellQuote(algo)))
	if err != nil {
		return nil, err
	}
	data := map[string]string{}
	for _, line := range strings.Split(out, "\n") {
		if k, v, ok := strings.Cut(strings.TrimSpace(line), "="); ok {
			data[k] = v
		}
	}
	if data["exists"] != "true" {
		return map[string]interface{}{"exists": false, "path": path}, nil
	}
	stat := map[string]interface{}{
		"exists": true, "path": path,
		"isdir": data["type"] == "directory", "isreg": data["type"] == "file", "islnk": data["type"] == "link",
		"pw_name": data["pw_name"], "gr_name": data["gr_name"],
	}
	for _, k := range []string{"size", "mtime", "atime", "ctime", "uid", "gid", "inode", "nlink"} {
		if n, err := strconv.ParseInt(data[k], 10, 64); err == nil {
			stat[k] = n
		}
	}
	if m, err := strconv.ParseUint(data["mode"], 8, 32); err == nil {
		stat["mode"] = fmt.Sprintf("%04o", m)
		stat["readable"] = m&0o400 != 0
		stat["writable"] = m&0o200 != 0
		stat["executable"] = m&0o100 != 0
	}
	for _, k := range []string{"checksum", "lnk_source", "lnk_target"} {
		if v, ok := data[k]; ok {
			stat[k] = v
		}
	}
	return stat, nil
}

func (m *StatModule) Validate(args map[string]interface{}) error {
	if err := m.BaseModule.Validate(args); err != nil {
		return err
	}

	path, exists := args["path"]
	if !exists {
		return fmt.Errorf("argument 'path' is required")
	}

	if _, ok := path.(string); !ok {
		return fmt.Errorf("argument 'path' must be a string")
	}

	return nil
}
