package modules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// AuthorizedKeyModule manages SSH authorized keys
type AuthorizedKeyModule struct {
	*BaseModule
}

// NewAuthorizedKeyModule creates a new authorized_key module
func NewAuthorizedKeyModule() *AuthorizedKeyModule {
	return &AuthorizedKeyModule{
		BaseModule: NewBaseModule("authorized_key"),
	}
}

func (m *AuthorizedKeyModule) GetDescription() string {
	return "Manage SSH authorized keys for user accounts"
}

func (m *AuthorizedKeyModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()

	result := types.TaskResult{
		TaskName:  taskName(args),
		Host:      host.Name,
		Module:    m.name,
		Timestamp: startTime,
		Success:   true,
		Changed:   false,
		Output:    make(map[string]interface{}),
	}

	// Get parameters
	username := ""
	if userVal, exists := args["user"]; exists {
		if userStr, ok := userVal.(string); ok {
			username = userStr
		}
	}

	if username == "" {
		result.Success = false
		result.Error = "'user' parameter is required"
		result.Duration = time.Since(startTime)
		return result, nil
	}

	key := ""
	if keyVal, exists := args["key"]; exists {
		if keyStr, ok := keyVal.(string); ok {
			key = strings.TrimSpace(keyStr)
		}
	}

	if key == "" {
		result.Success = false
		result.Error = "'key' parameter is required"
		result.Duration = time.Since(startTime)
		return result, nil
	}

	state := "present"
	if stateVal, exists := args["state"]; exists {
		if stateStr, ok := stateVal.(string); ok {
			state = stateStr
		}
	}

	exclusive := false
	if exclusiveVal, exists := args["exclusive"]; exists {
		if exclusiveBool, ok := exclusiveVal.(bool); ok {
			exclusive = exclusiveBool
		}
	}

	// Resolve the account on the target host, not on the control machine
	passwd, err := runOnHost(ctx, host, args, "getent", "passwd", username)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("user %q not found on %s", username, host.Name)
		result.Duration = time.Since(startTime)
		return result, nil
	}
	fields := strings.Split(strings.TrimSpace(passwd), ":")
	if len(fields) < 6 || fields[5] == "" {
		result.Success = false
		result.Error = fmt.Sprintf("no home directory for user %q", username)
		result.Duration = time.Since(startTime)
		return result, nil
	}
	homeDir := fields[5]
	primaryGroup, err := runOnHost(ctx, host, args, "id", "-gn", username)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to read the group of %q: %v", username, err)
		result.Duration = time.Since(startTime)
		return result, nil
	}
	owner := username + ":" + strings.TrimSpace(primaryGroup)
	sshDir := homeDir + "/.ssh"
	authKeysPath := sshDir + "/authorized_keys"

	data, _, err := readHostFile(ctx, host, args, authKeysPath)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Keep every existing line (comments, options) and compare keys by type
	// and key material, not by their trailing comment
	var lines []string
	if len(data) > 0 {
		lines = strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	}
	wantID := authorizedKeyID(key)
	kept := make([]string, 0, len(lines)+1)
	found := false
	for _, line := range lines {
		id := authorizedKeyID(line)
		switch {
		case id != "" && id == wantID:
			if state == "present" && !found {
				kept = append(kept, line)
			}
			found = true
		case exclusive && state == "present" && id != "":
			// another key: dropped in exclusive mode
		default:
			kept = append(kept, line)
		}
	}
	if state == "present" && !found {
		kept = append(kept, key)
	}
	newKeys := kept
	content := strings.Join(kept, "\n")
	if content != "" {
		content += "\n"
	}
	result.Changed = content != string(data)

	if result.Changed {
		if _, err := runShellOnHost(ctx, host, args, fmt.Sprintf("install -d -m 700 -o %s -g %s %s",
			shellQuote(username), shellQuote(strings.TrimSpace(primaryGroup)), shellQuote(sshDir))); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to create %s: %v", sshDir, err)
			result.Duration = time.Since(startTime)
			return result, nil
		}
		if err := writeHostFile(ctx, host, args, authKeysPath, []byte(content), 0600); err != nil {
			result.Success = false
			result.Error = err.Error()
			result.Duration = time.Since(startTime)
			return result, nil
		}
		if _, err := runOnHost(ctx, host, args, "chown", owner, authKeysPath); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to set the owner of %s: %v", authKeysPath, err)
			result.Duration = time.Since(startTime)
			return result, nil
		}
	}

	result.Output["user"] = username
	result.Output["state"] = state
	result.Output["key_count"] = len(newKeys)
	result.Output["msg"] = fmt.Sprintf("Key %s", state)

	result.Duration = time.Since(startTime)
	return result, nil
}

func (m *AuthorizedKeyModule) Validate(args map[string]interface{}) error {
	if err := m.BaseModule.Validate(args); err != nil {
		return err
	}

	// Check required parameters
	if _, exists := args["user"]; !exists {
		return fmt.Errorf("authorized_key module requires 'user' parameter")
	}

	if _, exists := args["key"]; !exists {
		return fmt.Errorf("authorized_key module requires 'key' parameter")
	}

	return nil
}

// authorizedKeyID returns "type key" of an authorized_keys line, skipping
// leading options, or "" for comments and blank lines
func authorizedKeyID(line string) string {
	fields := strings.Fields(line)
	if len(fields) == 0 || strings.HasPrefix(fields[0], "#") {
		return ""
	}
	for i := 0; i+1 < len(fields); i++ {
		t := fields[i]
		if strings.HasPrefix(t, "ssh-") || strings.HasPrefix(t, "ecdsa-") || strings.HasPrefix(t, "sk-") {
			return t + " " + fields[i+1]
		}
	}
	return ""
}
