package modules

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// EnhancedPackageModule implements improved package management with better idempotency
type EnhancedPackageModule struct {
	BaseModule
	packageManager EnhancedPackageManager
	stateCache     *PackageStateCache
	operationLock  sync.RWMutex
}

// EnhancedPackageManager interface with improved methods for idempotency
type EnhancedPackageManager interface {
	// Core operations
	Install(ctx context.Context, name, version string) (*PackageOperation, error)
	Remove(ctx context.Context, name string) (*PackageOperation, error)
	Update(ctx context.Context, name string) (*PackageOperation, error)
	UpdateAll(ctx context.Context) (*PackageOperation, error)

	// State checking with caching
	IsInstalled(ctx context.Context, name string) (*EnhancedPackageState, error)
	GetPackageInfo(ctx context.Context, name string) (*EnhancedPackageInfo, error)

	// Batch operations for better performance
	InstallMultiple(ctx context.Context, packages []PackageSpec) (*BatchOperation, error)
	RemoveMultiple(ctx context.Context, packages []string) (*BatchOperation, error)

	// Cache and state management
	RefreshCache(ctx context.Context) error
	ValidateState(ctx context.Context) error

	// Dry run support
	DryRun(ctx context.Context, operation string, args ...string) (*OperationPreview, error)
}

// EnhancedPackageState represents the current state of a package
type EnhancedPackageState struct {
	Name             string    `json:"name"`
	Installed        bool      `json:"installed"`
	Version          string    `json:"version"`
	AvailableVersion string    `json:"available_version"`
	Repository       string    `json:"repository"`
	LastChecked      time.Time `json:"last_checked"`
	Hash             string    `json:"hash"` // For change detection
}

// EnhancedPackageInfo extends PackageInfo with additional metadata
type EnhancedPackageInfo struct {
	PackageInfo
	Dependencies []string  `json:"dependencies"`
	ReverseDeps  []string  `json:"reverse_dependencies"`
	InstallDate  time.Time `json:"install_date"`
	LastUpdate   time.Time `json:"last_update"`
	Checksum     string    `json:"checksum"`
	Source       string    `json:"source"`
}

// PackageSpec defines a package specification for batch operations
type PackageSpec struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	State   string `json:"state"`
}

// PackageOperation represents the result of a package operation
type PackageOperation struct {
	Package    string        `json:"package"`
	Operation  string        `json:"operation"`
	Success    bool          `json:"success"`
	Changed    bool          `json:"changed"`
	OldVersion string        `json:"old_version,omitempty"`
	NewVersion string        `json:"new_version,omitempty"`
	Duration   time.Duration `json:"duration"`
	Output     string        `json:"output"`
	Error      string        `json:"error,omitempty"`
}

// BatchOperation represents the result of a batch operation
type BatchOperation struct {
	Operations []PackageOperation `json:"operations"`
	Success    bool               `json:"success"`
	Changed    bool               `json:"changed"`
	Duration   time.Duration      `json:"duration"`
	Summary    string             `json:"summary"`
}

// OperationPreview represents a dry run result
type OperationPreview struct {
	WillChange   bool     `json:"will_change"`
	Actions      []string `json:"actions"`
	Dependencies []string `json:"dependencies"`
	Conflicts    []string `json:"conflicts"`
	Size         string   `json:"size"`
}

// PackageStateCache provides thread-safe caching of package states
type PackageStateCache struct {
	cache  sync.Map
	ttl    time.Duration
	hits   int64
	misses int64
}

// NewPackageStateCache creates a new package state cache
func NewPackageStateCache(ttl time.Duration) *PackageStateCache {
	return &PackageStateCache{
		ttl: ttl,
	}
}

// Get retrieves a package state from cache
func (c *PackageStateCache) Get(name string) (*EnhancedPackageState, bool) {
	if value, ok := c.cache.Load(name); ok {
		state := value.(*EnhancedPackageState)
		if time.Since(state.LastChecked) < c.ttl {
			atomic.AddInt64(&c.hits, 1)
			return state, true
		}
		// Expired, remove from cache
		c.cache.Delete(name)
	}
	atomic.AddInt64(&c.misses, 1)
	return nil, false
}

// Set stores a package state in cache
func (c *PackageStateCache) Set(name string, state *EnhancedPackageState) {
	state.LastChecked = time.Now()
	c.cache.Store(name, state)
}

// Delete removes a specific entry from cache
func (c *PackageStateCache) Delete(name string) {
	c.cache.Delete(name)
}

// Clear removes all entries from cache
func (c *PackageStateCache) Clear() {
	c.cache.Range(func(key, value interface{}) bool {
		c.cache.Delete(key)
		return true
	})
}

// Stats returns cache statistics
func (c *PackageStateCache) Stats() (hits, misses int64) {
	return atomic.LoadInt64(&c.hits), atomic.LoadInt64(&c.misses)
}

// NewEnhancedPackageModule creates a new enhanced package module
func NewEnhancedPackageModule() *EnhancedPackageModule {
	return &EnhancedPackageModule{
		BaseModule: BaseModule{
			name:        "package_enhanced",
			description: "Enhanced package management with improved idempotency",
		},
		stateCache: NewPackageStateCache(5 * time.Minute),
	}
}

// Execute manages system packages with enhanced idempotency
func (m *EnhancedPackageModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()
	result := types.TaskResult{
		TaskName:  "package_enhanced",
		Host:      host.Name,
		Module:    m.GetName(),
		Success:   true,
		Changed:   false,
		Output:    make(map[string]interface{}),
		Timestamp: startTime,
	}

	// Create executor for this host
	exec, err := executor.NewCommandExecutor(host)
	if err != nil {
		return m.failResult(result, fmt.Sprintf("failed to create executor: %v", err))
	}
	defer exec.Close()

	// Create enhanced package manager with executor
	m.packageManager = createEnhancedPackageManager(ctx, exec)

	// Parse arguments
	name, ok := args["name"].(string)
	if !ok {
		return m.failResult(result, "name parameter is required")
	}

	state := getStringArg(args, "state", "present")
	version := getStringArg(args, "version", "")
	updateCache := getBoolArg(args, "update_cache", false)
	dryRun := getBoolArg(args, "dry_run", false)

	// Update cache if requested
	if updateCache {
		if err := m.packageManager.RefreshCache(ctx); err != nil {
			result.Output["cache_update_warning"] = err.Error()
		}
	}

	// Perform dry run if requested
	if dryRun {
		preview, err := m.packageManager.DryRun(ctx, state, name, version)
		if err != nil {
			return m.failResult(result, fmt.Sprintf("dry run failed: %v", err))
		}
		result.Output["preview"] = preview
		result.Output["dry_run"] = true
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Lock for thread safety
	m.operationLock.Lock()
	defer m.operationLock.Unlock()

	// Get current package state with caching
	currentState, err := m.packageManager.IsInstalled(ctx, name)
	if err != nil {
		return m.failResult(result, fmt.Sprintf("failed to check package state: %v", err))
	}

	result.Output["package_name"] = name
	result.Output["current_state"] = currentState

	// Execute operation based on desired state
	var operation *PackageOperation
	switch state {
	case "present":
		operation, err = m.handlePresentState(ctx, name, version, currentState)
	case "absent":
		operation, err = m.handleAbsentState(ctx, name, currentState)
	case "latest":
		operation, err = m.handleLatestState(ctx, name, currentState)
	default:
		return m.failResult(result, fmt.Sprintf("invalid state: %s", state))
	}

	if err != nil {
		return m.failResult(result, fmt.Sprintf("operation failed: %v", err))
	}

	// Update result with operation details
	result.Changed = operation.Changed
	result.Output["operation"] = operation

	// Get final package info
	if finalInfo, err := m.packageManager.GetPackageInfo(ctx, name); err == nil {
		result.Output["final_info"] = finalInfo
	}

	// Add cache statistics
	hits, misses := m.stateCache.Stats()
	result.Output["cache_stats"] = map[string]interface{}{
		"hits":   hits,
		"misses": misses,
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// handlePresentState handles the "present" state
func (m *EnhancedPackageModule) handlePresentState(ctx context.Context, name, version string, currentState *EnhancedPackageState) (*PackageOperation, error) {
	if !currentState.Installed {
		// Package not installed, install it
		return m.packageManager.Install(ctx, name, version)
	}

	if version != "" && currentState.Version != version {
		// Specific version requested and current version differs
		return m.packageManager.Install(ctx, name, version)
	}

	// Package already in desired state
	return &PackageOperation{
		Package:   name,
		Operation: "present",
		Success:   true,
		Changed:   false,
		Duration:  0,
		Output:    "Package already in desired state",
	}, nil
}

// handleAbsentState handles the "absent" state
func (m *EnhancedPackageModule) handleAbsentState(ctx context.Context, name string, currentState *EnhancedPackageState) (*PackageOperation, error) {
	if currentState.Installed {
		// Package installed, remove it
		return m.packageManager.Remove(ctx, name)
	}

	// Package already absent
	return &PackageOperation{
		Package:   name,
		Operation: "absent",
		Success:   true,
		Changed:   false,
		Duration:  0,
		Output:    "Package already absent",
	}, nil
}

// handleLatestState handles the "latest" state
func (m *EnhancedPackageModule) handleLatestState(ctx context.Context, name string, currentState *EnhancedPackageState) (*PackageOperation, error) {
	if !currentState.Installed {
		// Package not installed, install latest
		return m.packageManager.Install(ctx, name, "")
	}

	if currentState.AvailableVersion != "" && currentState.Version != currentState.AvailableVersion {
		// Update available
		return m.packageManager.Update(ctx, name)
	}

	// Package already at latest version
	return &PackageOperation{
		Package:   name,
		Operation: "latest",
		Success:   true,
		Changed:   false,
		Duration:  0,
		Output:    "Package already at latest version",
	}, nil
}

// Validate validates enhanced package module arguments
func (m *EnhancedPackageModule) Validate(args map[string]interface{}) error {
	if _, exists := args["name"]; !exists {
		return fmt.Errorf("name parameter is required")
	}

	if state, exists := args["state"]; exists {
		if stateStr, ok := state.(string); ok {
			validStates := []string{"present", "absent", "latest"}
			valid := false
			for _, validState := range validStates {
				if stateStr == validState {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("invalid state: %s", stateStr)
			}
		}
	}

	return nil
}

// createEnhancedPackageManager creates appropriate enhanced package manager
func createEnhancedPackageManager(ctx context.Context, exec *executor.CommandExecutor) EnhancedPackageManager {
	switch runtime.GOOS {
	case "linux":
		if hasCommand("apt") {
			return NewEnhancedAptManager(exec)
		} else if hasCommand("yum") {
			return NewEnhancedYumManager(exec)
		} else if hasCommand("dnf") {
			return NewEnhancedDnfManager(exec)
		}
	case "darwin":
		if hasCommand("brew") {
			return NewEnhancedBrewManager(exec)
		}
	case "windows":
		if hasCommand("choco") {
			return NewEnhancedChocolateyManager(exec)
		}
	}
	return NewEnhancedGenericManager(exec)
}

// generateStateHash generates a hash for package state to detect changes
func generateStateHash(name, version, repository string) string {
	data := fmt.Sprintf("%s:%s:%s", name, version, repository)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// failResult creates a failed result
func (m *EnhancedPackageModule) failResult(result types.TaskResult, message string) (types.TaskResult, error) {
	result.Success = false
	result.Failed = true
	result.Error = message
	result.Duration = time.Since(result.Timestamp)
	return result, fmt.Errorf("%s", message)
}
