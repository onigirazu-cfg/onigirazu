package modules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// GetURLModule implements file downloading from URLs
type GetURLModule struct {
	BaseModule
}

// NewGetURLModule creates a new get_url module instance
func NewGetURLModule() *GetURLModule {
	return &GetURLModule{
		BaseModule: BaseModule{
			name:        "get_url",
			description: "Download files from HTTP, HTTPS, or FTP URLs",
		},
	}
}

// GetDescription returns the module description
func (m *GetURLModule) GetDescription() string {
	return m.description
}

// Validate validates the module arguments
func (m *GetURLModule) Validate(args map[string]interface{}) error {
	// Required: url
	if _, ok := args["url"]; !ok {
		return fmt.Errorf("url parameter is required")
	}

	// Required: dest
	if _, ok := args["dest"]; !ok {
		return fmt.Errorf("dest parameter is required")
	}

	return nil
}

// Execute executes the get_url module
func (m *GetURLModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
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

	// Get parameters
	url, _ := args["url"].(string)
	dest, _ := args["dest"].(string)
	force := getBoolArg(args, "force", false)
	backup := getBoolArg(args, "backup", false)
	checksum := getStringArg(args, "checksum", "")
	mode := getStringArg(args, "mode", "0644")
	owner := getStringArg(args, "owner", "")
	group := getStringArg(args, "group", "")
	timeout := getIntArg(args, "timeout", 30)
	headers := make(map[string]string)
	if headersVal, ok := args["headers"].(map[string]interface{}); ok {
		for k, v := range headersVal {
			if strVal, ok := v.(string); ok {
				headers[k] = strVal
			}
		}
	}

	// Initialize executor
	exec, err := executor.NewCommandExecutor(host)
	if err != nil {
		result.Failed = true
		result.Error = fmt.Sprintf("Failed to create executor: %v", err)
		return result, err
	}
	defer exec.Close()

	// Check if destination exists
	checkCmd := fmt.Sprintf("test -f %s && echo 'exists' || echo 'not_exists'", shellQuote(dest))
	checkOutput, _ := exec.ExecuteWithContext(ctx, "sh", "-c", checkCmd)
	destExists := strings.TrimSpace(checkOutput) == "exists"

	// If file exists and force is false, check if we need to download
	if destExists && !force {
		// Check if checksum matches (if provided)
		if checksum != "" {
			destChecksum, err := m.getRemoteFileChecksum(ctx, exec, dest, checksum)
			if err == nil && m.checksumMatches(checksum, destChecksum) {
				result.Output["msg"] = "File already exists with matching checksum"
				result.Output["dest"] = dest
				result.Output["url"] = url
				result.Success = true
				return result, nil
			}
		} else {
			// File exists and no checksum provided, skip download
			result.Output["msg"] = "File already exists (use force=true to overwrite)"
			result.Output["dest"] = dest
			result.Output["url"] = url
			result.Success = true
			return result, nil
		}
	}

	if inCheckMode(args) {
		result.Success = true
		result.Changed = true
		result.Output["msg"] = map[bool]string{true: "file would be downloaded again", false: "file would be downloaded"}[destExists]
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Create backup if requested and file exists
	if backup && destExists {
		backupCmd := fmt.Sprintf("cp %s %s", shellQuote(dest), shellQuote(fmt.Sprintf("%s.%d.backup", dest, time.Now().Unix())))
		_, err := exec.ExecuteWithContext(ctx, "sh", "-c", backupCmd)
		if err != nil {
			result.Output["warning"] = fmt.Sprintf("Failed to create backup: %v", err)
		} else {
			result.Output["backup_file"] = fmt.Sprintf("%s.%d.backup", dest, time.Now().Unix())
		}
	}

	// Download on the host, so URLs only the host can reach work and files of
	// any size go straight to disk: curl (or wget) into a temporary file, the
	// checksum check with the algorithm asked for, then install(1).
	headerFile := ""
	if len(headers) > 0 {
		headerFile = fmt.Sprintf("/tmp/.onigirazu-headers-%d", time.Now().UnixNano())
		var cfg strings.Builder
		for k, v := range headers {
			fmt.Fprintf(&cfg, "header = %q\n", k+": "+v)
		}
		if err := putPrivateFile(ctx, host, headerFile, []byte(cfg.String())); err != nil {
			result.Failed = true
			result.Error = fmt.Sprintf("failed to pass headers: %v", err)
			return result, err
		}
	}
	algo, want := "", ""
	if checksum != "" {
		algo, want, _ = strings.Cut(checksum, ":")
	}
	hashCmd := map[string]string{"md5": "md5sum", "sha1": "sha1sum", "sha256": "sha256sum", "sha512": "sha512sum"}[algo]
	if checksum != "" && hashCmd == "" {
		result.Failed = true
		result.Error = fmt.Sprintf("unsupported checksum algorithm %q", algo)
		return result, fmt.Errorf("%s", result.Error)
	}

	curlConfig := ""
	if headerFile != "" {
		curlConfig = " -K " + shellQuote(headerFile)
	}
	// values reach the script only as quoted variable assignments
	script := fmt.Sprintf(`set -e
t=$(mktemp)
h=%[1]s
trap 'rm -f "$t" ${h:+"$h"}' EXIT
if command -v curl >/dev/null 2>&1; then
  curl -fsSL --max-time %[2]d%[3]s -o "$t" %[4]s
else
  wget -q -T %[2]d -O "$t" %[4]s
fi
`, shellQuote(headerFile), timeout, curlConfig, shellQuote(url))
	if hashCmd != "" {
		script += fmt.Sprintf(`want=%s
got=$(%s "$t" | cut -d' ' -f1)
[ "$got" = "$want" ] || { printf 'checksum mismatch: expected %%s, got %%s\n' "$want" "$got" >&2; exit 3; }
`, shellQuote(strings.ToLower(want)), hashCmd)
	}
	// unchanged content: report it and keep the file as it is
	script += fmt.Sprintf(`if [ -f %[1]s ] && cmp -s "$t" %[1]s; then echo unchanged; exit 0; fi
install -D -m %[2]s "$t" %[1]s
wc -c < %[1]s
`, shellQuote(dest), shellQuote(mode))

	out, err := runShellOnHost(ctx, host, args, script)
	if err != nil {
		result.Failed = true
		result.Error = fmt.Sprintf("failed to download %s: %v", url, err)
		return result, err
	}
	out = strings.TrimSpace(out)
	changed := out != "unchanged"

	ownerChanged, err := ensureOwnership(ctx, host, args, dest, owner, group)
	if err != nil {
		result.Failed = true
		result.Error = err.Error()
		return result, err
	}

	result.Changed = changed || ownerChanged
	result.Success = true
	result.Output["url"] = url
	result.Output["dest"] = dest
	if changed {
		result.Output["msg"] = "File downloaded"
		result.Output["size"] = out
	} else {
		result.Output["msg"] = "File already has this content"
	}
	if checksum != "" {
		result.Output["checksum"] = checksum
	}
	return result, nil
}

// getRemoteFileChecksum calculates checksum of a file on remote host
func (m *GetURLModule) getRemoteFileChecksum(ctx context.Context, exec *executor.CommandExecutor, path string, checksumType string) (string, error) {
	var cmd string
	checksumAlgo := strings.Split(checksumType, ":")[0]

	switch checksumAlgo {
	case "md5":
		cmd = fmt.Sprintf("md5sum %[1]s 2>/dev/null || md5 -q %[1]s 2>/dev/null", shellQuote(path))
	case "sha1":
		cmd = fmt.Sprintf("sha1sum %[1]s 2>/dev/null || shasum -a 1 %[1]s 2>/dev/null", shellQuote(path))
	case "sha256":
		cmd = fmt.Sprintf("sha256sum %[1]s 2>/dev/null || shasum -a 256 %[1]s 2>/dev/null", shellQuote(path))
	default:
		return "", fmt.Errorf("unsupported checksum type: %s", checksumAlgo)
	}

	output, err := exec.ExecuteWithContext(ctx, "sh", "-c", cmd)
	if err != nil {
		return "", err
	}

	// Extract checksum from output
	parts := strings.Fields(output)
	if len(parts) > 0 {
		return parts[0], nil
	}

	return "", fmt.Errorf("failed to extract checksum from output")
}

// checksumMatches checks if two checksums match
func (m *GetURLModule) checksumMatches(expected, actual string) bool {
	// Expected format: "algorithm:checksum" or just "checksum"
	expectedChecksum := expected
	if strings.Contains(expected, ":") {
		parts := strings.Split(expected, ":")
		if len(parts) == 2 {
			expectedChecksum = parts[1]
		}
	}

	return strings.EqualFold(expectedChecksum, actual)
}
