package modules

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
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
		TaskName:  args["name"].(string),
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

	// Get user info
	u, err := user.Lookup(username)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("user '%s' not found: %v", username, err)
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Build authorized_keys path
	authKeysPath := filepath.Join(u.HomeDir, ".ssh", "authorized_keys")

	// Read existing keys
	existingKeys := []string{}
	if data, err := os.ReadFile(authKeysPath); err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" && !strings.HasPrefix(line, "#") {
				existingKeys = append(existingKeys, line)
			}
		}
	} else if !os.IsNotExist(err) {
		result.Success = false
		result.Error = fmt.Sprintf("failed to read authorized_keys: %v", err)
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Determine if key exists
	keyExists := false
	keyIndex := -1
	for i, existingKey := range existingKeys {
		if strings.TrimSpace(existingKey) == key {
			keyExists = true
			keyIndex = i
			break
		}
	}

	// Handle state
	newKeys := existingKeys
	if state == "present" {
		if !keyExists {
			newKeys = append(newKeys, key)
			result.Changed = true
		}
		if exclusive {
			// In exclusive mode, keep only this key
			if len(newKeys) > 1 || (len(newKeys) == 1 && newKeys[0] != key) {
				newKeys = []string{key}
				result.Changed = true
			}
		}
	} else if state == "absent" {
		if keyExists {
			// Remove the key
			newKeys = append(existingKeys[:keyIndex], existingKeys[keyIndex+1:]...)
			result.Changed = true
		}
	}

	// Write back if changed
	if result.Changed {
		// Create .ssh directory if needed
		sshDir := filepath.Dir(authKeysPath)
		if err := os.MkdirAll(sshDir, 0700); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to create .ssh directory: %v", err)
			result.Duration = time.Since(startTime)
			return result, nil
		}

		// Write authorized_keys
		content := strings.Join(newKeys, "\n")
		if content != "" {
			content += "\n"
		}

		if err := os.WriteFile(authKeysPath, []byte(content), 0600); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to write authorized_keys: %v", err)
			result.Duration = time.Since(startTime)
			return result, nil
		}

		// Set proper permissions
		if err := os.Chown(authKeysPath, os.Getuid(), os.Getgid()); err != nil {
			// Continue anyway - may fail if not running as root
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
