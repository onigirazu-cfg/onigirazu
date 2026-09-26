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

	if inCheckMode(args) {
		return m.checkRepository(exec, repo, dest, version, !destExists || !isGitRepo, update, result, startTime)
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
	_, err := exec.Execute("test -e " + shellQuote(path))
	return err == nil
}

func (m *GitModuleFixed) executeInDirectory(exec *executor.CommandExecutor, dir string, command string, args ...string) (string, error) {
	// Change to directory and execute command; every part is quoted
	fullCommand := "cd " + shellQuote(dir) + " && " + shellJoin(append([]string{command}, args...)...)
	return exec.Execute(fullCommand)
}

func (m *GitModuleFixed) IsIdempotent() bool {
	return true
}

// checkRepository is check mode: a clone would change the host; an update
// would when the requested version resolves to another commit than HEAD
func (m *GitModuleFixed) checkRepository(exec *executor.CommandExecutor, repo, dest, version string, clone, update bool, result types.TaskResult, startTime time.Time) (types.TaskResult, error) {
	result.Success = true
	result.Duration = time.Since(startTime)
	switch {
	case clone:
		result.Changed = true
		result.Output["message"] = "repository would be cloned"
		return result, nil
	case !update:
		result.Output["message"] = "repository exists and update is disabled"
		return result, nil
	}
	current, err := m.getCurrentCommit(exec, dest)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to get current commit: %v", err)
		return result, nil
	}
	want := version
	if !isCommitHash(version) {
		// the peeled ^{} line of an annotated tag names the commit; it comes last
		out, err := exec.Execute(shellJoin("git", "ls-remote", repo, version, version+"^{}"))
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to resolve %s: %v", version, err)
			return result, nil
		}
		lines := strings.Fields(strings.TrimSpace(out))
		if len(lines) < 2 {
			result.Success = false
			result.Error = fmt.Sprintf("version %s not found in %s", version, repo)
			return result, nil
		}
		want = lines[len(lines)-2]
	}
	result.Changed = !strings.HasPrefix(current, want) && !strings.HasPrefix(want, current)
	result.Output["current_commit"] = current
	result.Output["target_commit"] = want
	if result.Changed {
		result.Output["message"] = "repository would be updated"
	} else {
		result.Output["message"] = "repository is at the requested version"
	}
	return result, nil
}

func isCommitHash(v string) bool {
	if len(v) < 7 || len(v) > 40 {
		return false
	}
	for _, r := range v {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}
