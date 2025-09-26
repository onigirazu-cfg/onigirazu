package modules

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// GitModule handles Git repository operations
type GitModule struct {
	*BaseModule
}

// NewGitModule creates a new git module
func NewGitModule() *GitModule {
	return &GitModule{
		BaseModule: &BaseModule{
			name:        "git",
			description: "Manage Git repositories",
		},
	}
}

func (m *GitModule) GetDescription() string {
	return "Manages Git repositories"
}

// Execute performs git operations
func (m *GitModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
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

	// Get required parameters
	repo, ok := args["repo"].(string)
	if !ok || repo == "" {
		result.Error = "repo parameter is required and must be a string"
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("%s", result.Error)
	}

	dest, ok := args["dest"].(string)
	if !ok || dest == "" {
		result.Error = "dest parameter is required and must be a string"
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("%s", result.Error)
	}

	// Get optional parameters
	version := getStringArg(args, "version", "HEAD")
	force := getBoolArg(args, "force", false)
	update := getBoolArg(args, "update", true)
	depth := getIntArg(args, "depth", 0)
	recursive := getBoolArg(args, "recursive", true)

	// Check if git is available
	if _, err := exec.LookPath("git"); err != nil {
		result.Error = "git command not found in PATH"
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("%s", result.Error)
	}

	// Check if destination exists and is a git repository
	isGitRepo := m.isGitRepository(dest)
	repoExists := m.directoryExists(dest)

	var changed bool
	var err error

	if !repoExists {
		// Clone repository
		changed, err = m.cloneRepository(ctx, repo, dest, version, depth, recursive)
		if err != nil {
			result.Error = fmt.Sprintf("failed to clone repository: %v", err)
			result.Duration = time.Since(startTime)
			return result, fmt.Errorf("%s", result.Error)
		}
		result.Output["operation"] = "clone"
	} else if !isGitRepo {
		if force {
			// Remove existing directory and clone
			if err := os.RemoveAll(dest); err != nil {
				result.Error = fmt.Sprintf("failed to remove existing directory: %v", err)
				result.Duration = time.Since(startTime)
				return result, fmt.Errorf("%s", result.Error)
			}
			changed, err = m.cloneRepository(ctx, repo, dest, version, depth, recursive)
			if err != nil {
				result.Error = fmt.Sprintf("failed to clone repository: %v", err)
				result.Duration = time.Since(startTime)
				return result, fmt.Errorf("%s", result.Error)
			}
			result.Output["operation"] = "force_clone"
			changed = true
		} else {
			result.Error = "destination exists but is not a git repository (use force=true to overwrite)"
			result.Duration = time.Since(startTime)
			return result, fmt.Errorf("%s", result.Error)
		}
	} else {
		// Repository exists, update if requested
		if update {
			changed, err = m.updateRepository(ctx, dest, version, force)
			if err != nil {
				result.Error = fmt.Sprintf("failed to update repository: %v", err)
				result.Duration = time.Since(startTime)
				return result, fmt.Errorf("%s", result.Error)
			}
			result.Output["operation"] = "update"
		} else {
			result.Output["operation"] = "none"
		}
	}

	// Get current commit info
	commitInfo, err := m.getCurrentCommit(dest)
	if err != nil {
		result.Output["commit_warning"] = fmt.Sprintf("failed to get commit info: %v", err)
	} else {
		result.Output["commit"] = commitInfo
	}

	// Get repository status
	status, err := m.getRepositoryStatus(dest)
	if err != nil {
		result.Output["status_warning"] = fmt.Sprintf("failed to get repository status: %v", err)
	} else {
		result.Output["status"] = status
	}

	result.Success = true
	result.Changed = changed
	result.Output["repo"] = repo
	result.Output["dest"] = dest
	result.Output["version"] = version
	result.Duration = time.Since(startTime)

	return result, nil
}

// Validate validates git module arguments
func (m *GitModule) Validate(args map[string]interface{}) error {
	// Check required arguments
	if _, ok := args["repo"]; !ok {
		return fmt.Errorf("repo parameter is required")
	}

	if _, ok := args["dest"]; !ok {
		return fmt.Errorf("dest parameter is required")
	}

	// Validate repo is a string
	if repo, ok := args["repo"].(string); !ok || repo == "" {
		return fmt.Errorf("repo must be a non-empty string")
	}

	// Validate dest is a string
	if dest, ok := args["dest"].(string); !ok || dest == "" {
		return fmt.Errorf("dest must be a non-empty string")
	}

	// Validate optional parameters
	if version, exists := args["version"]; exists {
		if _, ok := version.(string); !ok {
			return fmt.Errorf("version must be a string")
		}
	}

	if force, exists := args["force"]; exists {
		if _, ok := force.(bool); !ok {
			return fmt.Errorf("force must be a boolean")
		}
	}

	if update, exists := args["update"]; exists {
		if _, ok := update.(bool); !ok {
			return fmt.Errorf("update must be a boolean")
		}
	}

	if depth, exists := args["depth"]; exists {
		if _, ok := depth.(int); !ok {
			return fmt.Errorf("depth must be an integer")
		}
	}

	if recursive, exists := args["recursive"]; exists {
		if _, ok := recursive.(bool); !ok {
			return fmt.Errorf("recursive must be a boolean")
		}
	}

	return nil
}

// isGitRepository checks if a directory is a git repository
func (m *GitModule) isGitRepository(path string) bool {
	gitDir := filepath.Join(path, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// directoryExists checks if a directory exists
func (m *GitModule) directoryExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// cloneRepository clones a git repository
func (m *GitModule) cloneRepository(ctx context.Context, repo, dest, version string, depth int, recursive bool) (bool, error) {
	args := []string{"clone"}

	if depth > 0 {
		args = append(args, "--depth", fmt.Sprintf("%d", depth))
	}

	if !recursive {
		args = append(args, "--no-recurse-submodules")
	}

	if version != "HEAD" {
		args = append(args, "--branch", version)
	}

	args = append(args, repo, dest)

	cmd := exec.CommandContext(ctx, "git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("git clone failed: %v, output: %s", err, string(output))
	}

	return true, nil
}

// updateRepository updates an existing git repository
func (m *GitModule) updateRepository(ctx context.Context, dest, version string, force bool) (bool, error) {
	// Change to repository directory
	originalDir, err := os.Getwd()
	if err != nil {
		return false, err
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(dest); err != nil {
		return false, fmt.Errorf("failed to change to repository directory: %v", err)
	}

	// Get current commit
	currentCommit, err := m.getCurrentCommitHash()
	if err != nil {
		return false, fmt.Errorf("failed to get current commit: %v", err)
	}

	// Fetch latest changes
	cmd := exec.CommandContext(ctx, "git", "fetch", "origin")
	if output, err := cmd.CombinedOutput(); err != nil {
		return false, fmt.Errorf("git fetch failed: %v, output: %s", err, string(output))
	}

	// Checkout specified version
	checkoutArgs := []string{"checkout"}
	if force {
		checkoutArgs = append(checkoutArgs, "--force")
	}
	checkoutArgs = append(checkoutArgs, version)

	cmd = exec.CommandContext(ctx, "git", checkoutArgs...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return false, fmt.Errorf("git checkout failed: %v, output: %s", err, string(output))
	}

	// Get new commit
	newCommit, err := m.getCurrentCommitHash()
	if err != nil {
		return false, fmt.Errorf("failed to get new commit: %v", err)
	}

	return currentCommit != newCommit, nil
}

// getCurrentCommit gets current commit information
func (m *GitModule) getCurrentCommit(repoPath string) (map[string]interface{}, error) {
	originalDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(repoPath); err != nil {
		return nil, err
	}

	// Get commit hash
	cmd := exec.Command("git", "rev-parse", "HEAD")
	hashOutput, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	// Get commit message
	cmd = exec.Command("git", "log", "-1", "--pretty=format:%s")
	msgOutput, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	// Get commit author
	cmd = exec.Command("git", "log", "-1", "--pretty=format:%an")
	authorOutput, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	// Get commit date
	cmd = exec.Command("git", "log", "-1", "--pretty=format:%ci")
	dateOutput, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"hash":    strings.TrimSpace(string(hashOutput)),
		"message": strings.TrimSpace(string(msgOutput)),
		"author":  strings.TrimSpace(string(authorOutput)),
		"date":    strings.TrimSpace(string(dateOutput)),
	}, nil
}

// getCurrentCommitHash gets just the current commit hash
func (m *GitModule) getCurrentCommitHash() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// getRepositoryStatus gets repository status information
func (m *GitModule) getRepositoryStatus(repoPath string) (map[string]interface{}, error) {
	originalDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(repoPath); err != nil {
		return nil, err
	}

	// Get current branch
	cmd := exec.Command("git", "branch", "--show-current")
	branchOutput, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	// Get status
	cmd = exec.Command("git", "status", "--porcelain")
	statusOutput, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	// Check if there are uncommitted changes
	hasChanges := len(strings.TrimSpace(string(statusOutput))) > 0

	return map[string]interface{}{
		"branch":      strings.TrimSpace(string(branchOutput)),
		"clean":       !hasChanges,
		"has_changes": hasChanges,
	}, nil
}
