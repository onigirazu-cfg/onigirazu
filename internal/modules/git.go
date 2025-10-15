package modules

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// GitModuleFixed handles Git repository operations using remote executor
type GitModuleFixed struct {
	*BaseModule
}

// NewGitModuleFixed creates a new git module
func NewGitModuleFixed() *GitModuleFixed {
	return &GitModuleFixed{
		BaseModule: &BaseModule{
			name:        "git",
			description: "Manage Git repositories",
		},
	}
}

// NewGitModule creates a new git module (compatibility wrapper)
func NewGitModule() *GitModuleFixed {
	return NewGitModuleFixed()
}

func (m *GitModuleFixed) GetDescription() string {
	return "Manages Git repositories"
}

// Execute performs git operations
func (m *GitModuleFixed) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()

	result := types.TaskResult{
		TaskName:  "git",
		Host:      host.Name,
		Module:    m.name,
		Success:   false,
		Changed:   false,
		Output:    make(map[string]interface{}),
		Timestamp: startTime,
	}

	exec, err := executor.NewCommandExecutor(host)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create executor: %v", err)
		result.Duration = time.Since(startTime)
		return result, nil
	}
	defer exec.Close()

	// Get required parameters
	repo, ok := args["repo"].(string)
	if !ok || repo == "" {
		result.Error = "repo parameter is required"
		result.Duration = time.Since(startTime)
		return result, nil
	}

	dest, ok := args["dest"].(string)
	if !ok || dest == "" {
		result.Error = "dest parameter is required"
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Get optional parameters
	version := "HEAD"
	if v, ok := args["version"].(string); ok && v != "" {
		version = v
	}

	force := false
	if f, ok := args["force"].(bool); ok {
		force = f
	}

	update := true
	if u, ok := args["update"].(bool); ok {
		update = u
	}

	// Check if destination exists and is a git repository
	isGitRepo := m.isGitRepository(exec, dest)
	destExists := m.pathExists(exec, dest)

	if destExists && !isGitRepo && !force {
		result.Error = fmt.Sprintf("destination %s exists but is not a git repository. Use force=true to overwrite", dest)
		result.Duration = time.Since(startTime)
		return result, nil
	}

	if !destExists || (destExists && !isGitRepo && force) {
		// Clone repository
		return m.cloneRepository(exec, repo, dest, version, result, startTime)
	} else if isGitRepo && update {
		// Update existing repository
		return m.updateRepository(exec, dest, version, result, startTime)
	} else {
		// Repository exists and update is false
		result.Success = true
		result.Changed = false
		result.Output["message"] = "Repository already exists and update is disabled"
		result.Duration = time.Since(startTime)
		return result, nil
	}
}

func (m *GitModuleFixed) Validate(args map[string]interface{}) error {
	if err := m.BaseModule.Validate(args); err != nil {
		return err
	}

	repo, exists := args["repo"]
	if !exists {
		return fmt.Errorf("argument 'repo' is required")
	}
	if _, ok := repo.(string); !ok {
		return fmt.Errorf("argument 'repo' must be a string")
	}

	dest, exists := args["dest"]
	if !exists {
		return fmt.Errorf("argument 'dest' is required")
	}
	if _, ok := dest.(string); !ok {
		return fmt.Errorf("argument 'dest' must be a string")
	}

	// Validate optional parameters
	if version, exists := args["version"]; exists {
		if _, ok := version.(string); !ok {
			return fmt.Errorf("argument 'version' must be a string")
		}
	}

	if force, exists := args["force"]; exists {
		if _, ok := force.(bool); !ok {
			return fmt.Errorf("argument 'force' must be a boolean")
		}
	}

	if update, exists := args["update"]; exists {
		if _, ok := update.(bool); !ok {
			return fmt.Errorf("argument 'update' must be a boolean")
		}
	}

	return nil
}

func (m *GitModuleFixed) cloneRepository(exec *executor.CommandExecutor, repo, dest, version string, result types.TaskResult, startTime time.Time) (types.TaskResult, error) {
	// Create parent directory if it doesn't exist
	parentDir := filepath.Dir(dest)
	_, err := exec.Execute("mkdir", "-p", parentDir)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create parent directory: %v", err)
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Clone the repository
	output, err := exec.Execute("git", "clone", repo, dest)
	if err != nil {
		result.Error = fmt.Sprintf("failed to clone repository: %v", err)
		result.Output["stdout"] = output
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Checkout specific version if not HEAD
	if version != "HEAD" {
		err = m.checkoutVersion(exec, dest, version)
		if err != nil {
			result.Error = fmt.Sprintf("failed to checkout version %s: %v", version, err)
			result.Duration = time.Since(startTime)
			return result, nil
		}
	}

	// Get repository information
	repoInfo, err := m.getRepositoryInfo(exec, dest)
	if err != nil {
		result.Error = fmt.Sprintf("failed to get repository info: %v", err)
		result.Duration = time.Since(startTime)
		return result, nil
	}

	result.Success = true
	result.Changed = true
	result.Output["message"] = "Repository cloned successfully"
	result.Output["repo"] = repo
	result.Output["dest"] = dest
	result.Output["version"] = version
	result.Output["info"] = repoInfo
	result.Duration = time.Since(startTime)

	return result, nil
}

func (m *GitModuleFixed) updateRepository(exec *executor.CommandExecutor, dest, version string, result types.TaskResult, startTime time.Time) (types.TaskResult, error) {
	// Get current commit before update
	currentCommit, err := m.getCurrentCommit(exec, dest)
	if err != nil {
		result.Error = fmt.Sprintf("failed to get current commit: %v", err)
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Fetch latest changes
	_, err = m.executeInDirectory(exec, dest, "git", "fetch", "origin")
	if err != nil {
		result.Error = fmt.Sprintf("failed to fetch changes: %v", err)
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Checkout the specified version
	err = m.checkoutVersion(exec, dest, version)
	if err != nil {
		result.Error = fmt.Sprintf("failed to checkout version %s: %v", version, err)
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Get new commit after update
	newCommit, err := m.getCurrentCommit(exec, dest)
	if err != nil {
		result.Error = fmt.Sprintf("failed to get new commit: %v", err)
		result.Duration = time.Since(startTime)
		return result, nil
	}

	changed := currentCommit != newCommit

	// Get repository information
	repoInfo, err := m.getRepositoryInfo(exec, dest)
	if err != nil {
		result.Error = fmt.Sprintf("failed to get repository info: %v", err)
		result.Duration = time.Since(startTime)
		return result, nil
	}

	result.Success = true
	result.Changed = changed
	if changed {
		result.Output["message"] = "Repository updated successfully"
	} else {
		result.Output["message"] = "Repository is already up to date"
	}
	result.Output["dest"] = dest
	result.Output["version"] = version
	result.Output["before"] = currentCommit
	result.Output["after"] = newCommit
	result.Output["info"] = repoInfo
	result.Duration = time.Since(startTime)

	return result, nil
}

func (m *GitModuleFixed) checkoutVersion(exec *executor.CommandExecutor, dest, version string) error {
	checkoutArgs := []string{"checkout"}

	// Handle different version formats
	if strings.HasPrefix(version, "origin/") {
		checkoutArgs = append(checkoutArgs, "-B", strings.TrimPrefix(version, "origin/"), version)
	} else {
		checkoutArgs = append(checkoutArgs, version)
	}

	_, err := m.executeInDirectory(exec, dest, "git", checkoutArgs...)
	return err
}

func (m *GitModuleFixed) getCurrentCommit(exec *executor.CommandExecutor, dest string) (string, error) {
	output, err := m.executeInDirectory(exec, dest, "git", "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

func (m *GitModuleFixed) getRepositoryInfo(exec *executor.CommandExecutor, dest string) (map[string]interface{}, error) {
	info := make(map[string]interface{})

	// Get current commit
	commit, err := m.executeInDirectory(exec, dest, "git", "rev-parse", "HEAD")
	if err == nil {
		info["commit"] = strings.TrimSpace(commit)
	}

	// Get commit message
	message, err := m.executeInDirectory(exec, dest, "git", "log", "-1", "--pretty=format:%s")
	if err == nil {
		info["message"] = strings.TrimSpace(message)
	}

	// Get author
	author, err := m.executeInDirectory(exec, dest, "git", "log", "-1", "--pretty=format:%an")
	if err == nil {
		info["author"] = strings.TrimSpace(author)
	}

	// Get commit date
	date, err := m.executeInDirectory(exec, dest, "git", "log", "-1", "--pretty=format:%ci")
	if err == nil {
		info["date"] = strings.TrimSpace(date)
	}

	return info, nil
}

func (m *GitModuleFixed) isGitRepository(exec *executor.CommandExecutor, path string) bool {
	_, err := m.executeInDirectory(exec, path, "git", "rev-parse", "--git-dir")
	return err == nil
}

func (m *GitModuleFixed) pathExists(exec *executor.CommandExecutor, path string) bool {
	_, err := exec.Execute(fmt.Sprintf("test -e %s", path))
	return err == nil
}

func (m *GitModuleFixed) executeInDirectory(exec *executor.CommandExecutor, dir string, command string, args ...string) (string, error) {
	// Change to directory and execute command
	fullCommand := fmt.Sprintf("cd %s && %s", dir, command)
	for _, arg := range args {
		fullCommand += " " + arg
	}
	return exec.Execute(fullCommand)
}

func (m *GitModuleFixed) IsIdempotent() bool {
	return true
}
