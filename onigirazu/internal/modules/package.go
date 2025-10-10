package modules

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// ============================================================================
// CORE STRUCTURES
// ============================================================================

// UnifiedPackageModule implements comprehensive package management
type UnifiedPackageModule struct {
	BaseModule
	packageManager UnifiedPackageManager
	stateCache     *PackageStateCache
	metrics        *PackageMetrics
}

// UnifiedPackageManager interface combining all package management capabilities
type UnifiedPackageManager interface {
	// Basic operations
	Install(ctx context.Context, name, version string) (*PackageOperation, error)
	Remove(ctx context.Context, name string) (*PackageOperation, error)
	Update(ctx context.Context, name string) (*PackageOperation, error)
	UpdateAll(ctx context.Context) (*PackageOperation, error)

	// State checking
	IsInstalled(ctx context.Context, name string) (*PackageState, error)
	GetPackageInfo(ctx context.Context, name string) (*PackageInfo, error)

	// Batch operations
	InstallMultiple(ctx context.Context, packages []PackageSpec) (*BatchOperation, error)
	RemoveMultiple(ctx context.Context, packages []string) (*BatchOperation, error)

	// Cache management
	RefreshCache(ctx context.Context) error
	ValidateState(ctx context.Context) error

	// Advanced features
	DryRun(ctx context.Context, operation string, args ...string) (*OperationPreview, error)
	GetDependencies(ctx context.Context, name string) ([]string, error)
	VerifyChecksum(ctx context.Context, name, version string) (bool, error)

	// Discovery and search
	Search(ctx context.Context, query string) ([]PackageInfo, error)
	ListInstalled(ctx context.Context) ([]PackageInfo, error)
	ListUpgradable(ctx context.Context) ([]PackageInfo, error)

	// Maintenance
	Clean(ctx context.Context) error
	AutoRemove(ctx context.Context) ([]string, error)
	VerifyIntegrity(ctx context.Context) error
}

// ============================================================================
// DATA STRUCTURES
// ============================================================================

// PackageSpec defines a package specification
type PackageSpec struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	State   string `json:"state"`
}

// PackageState represents the current state of a package
type PackageState struct {
	Name             string    `json:"name"`
	Installed        bool      `json:"installed"`
	Version          string    `json:"version"`
	AvailableVersion string    `json:"available_version"`
	Repository       string    `json:"repository"`
	LastChecked      time.Time `json:"last_checked"`
	Hash             string    `json:"hash"`
	Dependencies     []string  `json:"dependencies,omitempty"`
}

// PackageInfo represents detailed package information
type PackageInfo struct {
	Name         string    `json:"name"`
	Version      string    `json:"version"`
	Description  string    `json:"description"`
	Architecture string    `json:"architecture"`
	Repository   string    `json:"repository"`
	Size         string    `json:"size"`
	Installed    bool      `json:"installed"`
	Upgradable   bool      `json:"upgradable"`
	NewVersion   string    `json:"new_version,omitempty"`
	Dependencies []string  `json:"dependencies,omitempty"`
	ReverseDeps  []string  `json:"reverse_dependencies,omitempty"`
	InstallDate  time.Time `json:"install_date,omitempty"`
	LastUpdate   time.Time `json:"last_update,omitempty"`
	Checksum     string    `json:"checksum,omitempty"`
	Source       string    `json:"source,omitempty"`
}

// PackageOperation represents the result of a package operation
type PackageOperation struct {
	Package      string        `json:"package"`
	Operation    string        `json:"operation"`
	Success      bool          `json:"success"`
	Changed      bool          `json:"changed"`
	OldVersion   string        `json:"old_version,omitempty"`
	NewVersion   string        `json:"new_version,omitempty"`
	Duration     time.Duration `json:"duration"`
	Output       string        `json:"output"`
	Error        string        `json:"error,omitempty"`
	RetryCount   int           `json:"retry_count,omitempty"`
	Dependencies []string      `json:"dependencies,omitempty"`
}

// BatchOperation represents the result of a batch operation
type BatchOperation struct {
	Operations   []PackageOperation `json:"operations"`
	Success      bool               `json:"success"`
	Changed      bool               `json:"changed"`
	Duration     time.Duration      `json:"duration"`
	Summary      string             `json:"summary"`
	TotalCount   int                `json:"total_count"`
	SuccessCount int                `json:"success_count"`
	FailedCount  int                `json:"failed_count"`
	ChangedCount int                `json:"changed_count"`
}

// OperationPreview represents a dry run result
type OperationPreview struct {
	WillChange    bool     `json:"will_change"`
	Actions       []string `json:"actions"`
	Dependencies  []string `json:"dependencies"`
	Conflicts     []string `json:"conflicts"`
	Size          string   `json:"size"`
	EstimatedTime string   `json:"estimated_time"`
}

// RollbackInfo stores information needed for rollback
type RollbackInfo struct {
	Timestamp   time.Time                `json:"timestamp"`
	Operations  []PackageOperation       `json:"operations"`
	PrevStates  map[string]*PackageState `json:"prev_states"`
	CanRollback bool                     `json:"can_rollback"`
}

// SystemSnapshot represents a snapshot of the package system state
type SystemSnapshot struct {
	ID          string                   `json:"id"`
	Timestamp   time.Time                `json:"timestamp"`
	Description string                   `json:"description"`
	Packages    map[string]*PackageState `json:"packages"`
	Checksum    string                   `json:"checksum"`
}

// PackageGroup represents a group of related packages
type PackageGroup struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Packages    []string `json:"packages"`
	Optional    []string `json:"optional,omitempty"`
}

// AuditEntry represents an audit log entry
type AuditEntry struct {
	Timestamp time.Time         `json:"timestamp"`
	User      string            `json:"user"`
	Host      string            `json:"host"`
	Operation string            `json:"operation"`
	Package   string            `json:"package"`
	Success   bool              `json:"success"`
	Details   map[string]string `json:"details,omitempty"`
}

// HealthCheckResult represents system health check result
type HealthCheckResult struct {
	Healthy         bool      `json:"healthy"`
	Issues          []string  `json:"issues,omitempty"`
	Warnings        []string  `json:"warnings,omitempty"`
	BrokenPackages  []string  `json:"broken_packages,omitempty"`
	OrphanPackages  []string  `json:"orphan_packages,omitempty"`
	Recommendations []string  `json:"recommendations,omitempty"`
	CheckedAt       time.Time `json:"checked_at"`
}

// ============================================================================
// CACHE IMPLEMENTATION
// ============================================================================

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
func (c *PackageStateCache) Get(name string) (*PackageState, bool) {
	if value, ok := c.cache.Load(name); ok {
		state := value.(*PackageState)
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
func (c *PackageStateCache) Set(name string, state *PackageState) {
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

// ============================================================================
// METRICS COLLECTION
// ============================================================================

// PackageMetrics collects metrics about package operations
type PackageMetrics struct {
	mu                sync.RWMutex
	TotalOperations   int64         `json:"total_operations"`
	SuccessfulOps     int64         `json:"successful_ops"`
	FailedOps         int64         `json:"failed_ops"`
	TotalDuration     time.Duration `json:"total_duration"`
	AverageDuration   time.Duration `json:"average_duration"`
	PackagesInstalled int64         `json:"packages_installed"`
	PackagesRemoved   int64         `json:"packages_removed"`
	PackagesUpdated   int64         `json:"packages_updated"`
	CacheHitRate      float64       `json:"cache_hit_rate"`
	RetryCount        int64         `json:"retry_count"`
}

// RecordOperation records a package operation in metrics
func (m *PackageMetrics) RecordOperation(op *PackageOperation) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalOperations++
	m.TotalDuration += op.Duration

	if op.Success {
		m.SuccessfulOps++
	} else {
		m.FailedOps++
	}

	if op.Changed {
		switch op.Operation {
		case "install":
			m.PackagesInstalled++
		case "remove":
			m.PackagesRemoved++
		case "update":
			m.PackagesUpdated++
		}
	}

	m.RetryCount += int64(op.RetryCount)

	if m.TotalOperations > 0 {
		m.AverageDuration = m.TotalDuration / time.Duration(m.TotalOperations)
	}
}

// GetMetrics returns a copy of current metrics
func (m *PackageMetrics) GetMetrics() PackageMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy without the mutex
	return PackageMetrics{
		TotalOperations:   m.TotalOperations,
		SuccessfulOps:     m.SuccessfulOps,
		FailedOps:         m.FailedOps,
		TotalDuration:     m.TotalDuration,
		AverageDuration:   m.AverageDuration,
		PackagesInstalled: m.PackagesInstalled,
		PackagesRemoved:   m.PackagesRemoved,
		PackagesUpdated:   m.PackagesUpdated,
		CacheHitRate:      m.CacheHitRate,
		RetryCount:        m.RetryCount,
	}
}

// ============================================================================
// LOCK FILE SUPPORT
// ============================================================================

// PackageLockFile represents a lock file for package versions
type PackageLockFile struct {
	Version  string                      `json:"version"`
	Created  time.Time                   `json:"created"`
	Updated  time.Time                   `json:"updated"`
	Packages map[string]PackageLockEntry `json:"packages"`
}

// PackageLockEntry represents a single package entry in lock file
type PackageLockEntry struct {
	Version      string   `json:"version"`
	Resolved     string   `json:"resolved"`
	Integrity    string   `json:"integrity"`
	Dependencies []string `json:"dependencies,omitempty"`
}

// LoadLockFile loads a package lock file
func LoadLockFile(path string) (*PackageLockFile, error) {
	// #nosec G304 - path is provided by user configuration and is intentional
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &PackageLockFile{
				Version:  "1.0",
				Created:  time.Now(),
				Packages: make(map[string]PackageLockEntry),
			}, nil
		}
		return nil, fmt.Errorf("failed to read lock file: %w", err)
	}

	var lockFile PackageLockFile
	if err := json.Unmarshal(data, &lockFile); err != nil {
		return nil, fmt.Errorf("failed to parse lock file: %w", err)
	}

	return &lockFile, nil
}

// SaveLockFile saves a package lock file
func (l *PackageLockFile) Save(path string) error {
	l.Updated = time.Now()

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal lock file: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write lock file: %w", err)
	}

	return nil
}

// UpdatePackage updates a package entry in lock file
func (l *PackageLockFile) UpdatePackage(name, version, checksum string, deps []string) {
	l.Packages[name] = PackageLockEntry{
		Version:      version,
		Resolved:     fmt.Sprintf("%s@%s", name, version),
		Integrity:    checksum,
		Dependencies: deps,
	}
}

// ============================================================================
// MODULE IMPLEMENTATION
// ============================================================================

// NewUnifiedPackageModule creates a new unified package module
func NewUnifiedPackageModule() *UnifiedPackageModule {
	return &UnifiedPackageModule{
		BaseModule: BaseModule{
			name:        "package",
			description: "Unified package management with advanced features",
		},
		stateCache: NewPackageStateCache(5 * time.Minute),
		metrics:    &PackageMetrics{},
	}
}

// Execute manages system packages with all features
func (m *UnifiedPackageModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	startTime := time.Now()
	result := types.TaskResult{
		TaskName:  "package",
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

	// Configure become (privilege escalation) if requested
	if become, ok := args["_become"].(bool); ok && become {
		becomeUser, _ := args["_become_user"].(string)
		becomeMethod, _ := args["_become_method"].(string)
		exec.SetBecome(true, becomeUser, becomeMethod)
	}

	// Create package manager with executor
	m.packageManager = createUnifiedPackageManager(ctx, exec, host.Name)

	// Parse arguments
	packageSpecs, err := m.parsePackageSpecs(args)
	if err != nil {
		return m.failResult(result, err.Error())
	}

	state := getStringArg(args, "state", "present")
	updateCache := getBoolArg(args, "update_cache", false)
	dryRun := getBoolArg(args, "dry_run", false)
	enableRollback := getBoolArg(args, "enable_rollback", false)
	parallel := getBoolArg(args, "parallel", false)
	maxRetries := getIntArg(args, "max_retries", 3)
	lockFile := getStringArg(args, "lock_file", "")

	// Update cache if requested
	if updateCache {
		if err := m.packageManager.RefreshCache(ctx); err != nil {
			result.Output["cache_update_warning"] = err.Error()
		}
	}

	// Load lock file if specified
	var lock *PackageLockFile
	if lockFile != "" {
		lock, err = LoadLockFile(lockFile)
		if err != nil {
			result.Output["lock_file_warning"] = err.Error()
		}
	}

	// Perform dry run if requested
	if dryRun {
		previews := make(map[string]*OperationPreview)
		for _, pkg := range packageSpecs {
			preview, err := m.packageManager.DryRun(ctx, state, pkg.Name, pkg.Version)
			if err != nil {
				result.Output[fmt.Sprintf("preview_error_%s", pkg.Name)] = err.Error()
				continue
			}
			previews[pkg.Name] = preview
		}
		result.Output["previews"] = previews
		result.Output["dry_run"] = true
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Prepare rollback info if enabled
	var rollback *RollbackInfo
	if enableRollback {
		rollback = &RollbackInfo{
			Timestamp:   time.Now(),
			Operations:  make([]PackageOperation, 0),
			PrevStates:  make(map[string]*PackageState),
			CanRollback: true,
		}
	}

	// Execute operations
	var batchOp *BatchOperation
	if parallel && len(packageSpecs) > 1 {
		batchOp, err = m.executeParallel(ctx, packageSpecs, state, maxRetries, rollback)
	} else {
		batchOp, err = m.executeSequential(ctx, packageSpecs, state, maxRetries, rollback)
	}

	if err != nil {
		// Attempt rollback if enabled and error occurred
		if enableRollback && rollback != nil && rollback.CanRollback {
			if rollbackErr := m.performRollback(ctx, rollback); rollbackErr != nil {
				result.Output["rollback_error"] = rollbackErr.Error()
			} else {
				result.Output["rollback"] = "successful"
			}
		}
		return m.failResult(result, fmt.Sprintf("operation failed: %v", err))
	}

	// Update lock file if specified
	if lock != nil && lockFile != "" {
		for _, op := range batchOp.Operations {
			if op.Success && op.Changed {
				deps, _ := m.packageManager.GetDependencies(ctx, op.Package)
				lock.UpdatePackage(op.Package, op.NewVersion, "", deps)
			}
		}
		if err := lock.Save(lockFile); err != nil {
			result.Output["lock_file_save_error"] = err.Error()
		}
	}

	// Update result
	result.Changed = batchOp.Changed
	result.Output["batch_operation"] = batchOp
	result.Output["summary"] = batchOp.Summary

	// Add cache statistics
	hits, misses := m.stateCache.Stats()
	result.Output["cache_stats"] = map[string]interface{}{
		"hits":   hits,
		"misses": misses,
		"rate":   float64(hits) / float64(hits+misses),
	}

	// Add metrics
	result.Output["metrics"] = m.metrics.GetMetrics()

	result.Duration = time.Since(startTime)
	return result, nil
}

// parsePackageSpecs parses package specifications from arguments
func (m *UnifiedPackageModule) parsePackageSpecs(args map[string]interface{}) ([]PackageSpec, error) {
	nameArg, exists := args["name"]
	if !exists {
		return nil, fmt.Errorf("name parameter is required")
	}

	globalVersion := getStringArg(args, "version", "")
	globalState := getStringArg(args, "state", "present")

	var packageSpecs []PackageSpec

	switch v := nameArg.(type) {
	case string:
		// Single package name
		packageSpecs = []PackageSpec{{Name: v, Version: globalVersion, State: globalState}}

	case []interface{}:
		for i, item := range v {
			switch itemVal := item.(type) {
			case string:
				// Simple string in list
				packageSpecs = append(packageSpecs, PackageSpec{
					Name:    itemVal,
					Version: globalVersion,
					State:   globalState,
				})

			case map[string]interface{}:
				// Object with name and optional version/state
				pkgName, ok := itemVal["name"].(string)
				if !ok {
					return nil, fmt.Errorf("name[%d].name must be a string", i)
				}
				pkgVersion := getStringArg(itemVal, "version", globalVersion)
				pkgState := getStringArg(itemVal, "state", globalState)
				packageSpecs = append(packageSpecs, PackageSpec{
					Name:    pkgName,
					Version: pkgVersion,
					State:   pkgState,
				})

			default:
				return nil, fmt.Errorf("name[%d] must be a string or object with 'name' field", i)
			}
		}

	case []string:
		// List of strings
		for _, name := range v {
			packageSpecs = append(packageSpecs, PackageSpec{
				Name:    name,
				Version: globalVersion,
				State:   globalState,
			})
		}

	default:
		return nil, fmt.Errorf("name parameter must be a string, list of strings, or list of objects")
	}

	if len(packageSpecs) == 0 {
		return nil, fmt.Errorf("at least one package name is required")
	}

	return packageSpecs, nil
}

// executeSequential executes package operations sequentially
func (m *UnifiedPackageModule) executeSequential(ctx context.Context, packages []PackageSpec, state string, maxRetries int, rollback *RollbackInfo) (*BatchOperation, error) {
	startTime := time.Now()
	batchOp := &BatchOperation{
		Operations: make([]PackageOperation, 0, len(packages)),
		TotalCount: len(packages),
	}

	for _, pkg := range packages {
		// Save current state for rollback
		if rollback != nil {
			currentState, _ := m.packageManager.IsInstalled(ctx, pkg.Name)
			if currentState != nil {
				rollback.PrevStates[pkg.Name] = currentState
			}
		}

		// Execute operation with retry
		var op *PackageOperation
		var err error
		for retry := 0; retry <= maxRetries; retry++ {
			op, err = m.executePackageOperation(ctx, pkg, state)
			if err == nil {
				break
			}
			if retry < maxRetries {
				time.Sleep(time.Second * time.Duration(retry+1))
				op.RetryCount = retry + 1
			}
		}

		if op != nil {
			batchOp.Operations = append(batchOp.Operations, *op)
			m.metrics.RecordOperation(op)

			if rollback != nil {
				rollback.Operations = append(rollback.Operations, *op)
			}

			if op.Success {
				batchOp.SuccessCount++
				if op.Changed {
					batchOp.ChangedCount++
				}
			} else {
				batchOp.FailedCount++
			}
		}
	}

	batchOp.Duration = time.Since(startTime)
	batchOp.Success = batchOp.FailedCount == 0
	batchOp.Changed = batchOp.ChangedCount > 0
	batchOp.Summary = fmt.Sprintf("Total: %d, Success: %d, Failed: %d, Changed: %d",
		batchOp.TotalCount, batchOp.SuccessCount, batchOp.FailedCount, batchOp.ChangedCount)

	return batchOp, nil
}

// executeParallel executes package operations in parallel
func (m *UnifiedPackageModule) executeParallel(ctx context.Context, packages []PackageSpec, state string, maxRetries int, rollback *RollbackInfo) (*BatchOperation, error) {
	startTime := time.Now()
	batchOp := &BatchOperation{
		Operations: make([]PackageOperation, len(packages)),
		TotalCount: len(packages),
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, pkg := range packages {
		wg.Add(1)
		go func(idx int, p PackageSpec) {
			defer wg.Done()

			// Save current state for rollback
			if rollback != nil {
				currentState, _ := m.packageManager.IsInstalled(ctx, p.Name)
				if currentState != nil {
					mu.Lock()
					rollback.PrevStates[p.Name] = currentState
					mu.Unlock()
				}
			}

			// Execute operation with retry
			var op *PackageOperation
			var err error
			for retry := 0; retry <= maxRetries; retry++ {
				op, err = m.executePackageOperation(ctx, p, state)
				if err == nil {
					break
				}
				if retry < maxRetries {
					time.Sleep(time.Second * time.Duration(retry+1))
					op.RetryCount = retry + 1
				}
			}

			if op != nil {
				mu.Lock()
				batchOp.Operations[idx] = *op
				m.metrics.RecordOperation(op)

				if rollback != nil {
					rollback.Operations = append(rollback.Operations, *op)
				}

				if op.Success {
					batchOp.SuccessCount++
					if op.Changed {
						batchOp.ChangedCount++
					}
				} else {
					batchOp.FailedCount++
				}
				mu.Unlock()
			}
		}(i, pkg)
	}

	wg.Wait()

	batchOp.Duration = time.Since(startTime)
	batchOp.Success = batchOp.FailedCount == 0
	batchOp.Changed = batchOp.ChangedCount > 0
	batchOp.Summary = fmt.Sprintf("Total: %d, Success: %d, Failed: %d, Changed: %d (parallel)",
		batchOp.TotalCount, batchOp.SuccessCount, batchOp.FailedCount, batchOp.ChangedCount)

	return batchOp, nil
}

// executePackageOperation executes a single package operation
func (m *UnifiedPackageModule) executePackageOperation(ctx context.Context, pkg PackageSpec, state string) (*PackageOperation, error) {
	// Get current state with caching
	currentState, err := m.packageManager.IsInstalled(ctx, pkg.Name)
	if err != nil {
		return &PackageOperation{
			Package:   pkg.Name,
			Operation: state,
			Success:   false,
			Error:     fmt.Sprintf("failed to check package state: %v", err),
		}, err
	}

	// Execute operation based on desired state
	switch state {
	case "present":
		return m.handlePresentState(ctx, pkg, currentState)
	case "absent":
		return m.handleAbsentState(ctx, pkg, currentState)
	case "latest":
		return m.handleLatestState(ctx, pkg, currentState)
	default:
		return &PackageOperation{
			Package:   pkg.Name,
			Operation: state,
			Success:   false,
			Error:     fmt.Sprintf("invalid state: %s", state),
		}, fmt.Errorf("invalid state: %s", state)
	}
}

// handlePresentState handles the "present" state
func (m *UnifiedPackageModule) handlePresentState(ctx context.Context, pkg PackageSpec, currentState *PackageState) (*PackageOperation, error) {
	if !currentState.Installed {
		// Package not installed, install it
		return m.packageManager.Install(ctx, pkg.Name, pkg.Version)
	}

	if pkg.Version != "" && currentState.Version != pkg.Version {
		// Specific version requested and current version differs
		return m.packageManager.Install(ctx, pkg.Name, pkg.Version)
	}

	// Package already in desired state
	return &PackageOperation{
		Package:   pkg.Name,
		Operation: "present",
		Success:   true,
		Changed:   false,
		Duration:  0,
		Output:    "Package already in desired state",
	}, nil
}

// handleAbsentState handles the "absent" state
func (m *UnifiedPackageModule) handleAbsentState(ctx context.Context, pkg PackageSpec, currentState *PackageState) (*PackageOperation, error) {
	if currentState.Installed {
		// Package installed, remove it
		return m.packageManager.Remove(ctx, pkg.Name)
	}

	// Package already absent
	return &PackageOperation{
		Package:   pkg.Name,
		Operation: "absent",
		Success:   true,
		Changed:   false,
		Duration:  0,
		Output:    "Package already absent",
	}, nil
}

// handleLatestState handles the "latest" state
func (m *UnifiedPackageModule) handleLatestState(ctx context.Context, pkg PackageSpec, currentState *PackageState) (*PackageOperation, error) {
	if !currentState.Installed {
		// Package not installed, install latest
		return m.packageManager.Install(ctx, pkg.Name, "")
	}

	if currentState.AvailableVersion != "" && currentState.Version != currentState.AvailableVersion {
		// Update available
		return m.packageManager.Update(ctx, pkg.Name)
	}

	// Package already at latest version
	return &PackageOperation{
		Package:   pkg.Name,
		Operation: "latest",
		Success:   true,
		Changed:   false,
		Duration:  0,
		Output:    "Package already at latest version",
	}, nil
}

// performRollback performs rollback of package operations
func (m *UnifiedPackageModule) performRollback(ctx context.Context, rollback *RollbackInfo) error {
	if !rollback.CanRollback {
		return fmt.Errorf("rollback not available")
	}

	// Reverse the operations
	for i := len(rollback.Operations) - 1; i >= 0; i-- {
		op := rollback.Operations[i]
		prevState, exists := rollback.PrevStates[op.Package]
		if !exists {
			continue
		}

		// Restore previous state
		if prevState.Installed && !op.Changed {
			continue // No change needed
		}

		if prevState.Installed {
			// Reinstall with previous version
			_, err := m.packageManager.Install(ctx, op.Package, prevState.Version)
			if err != nil {
				return fmt.Errorf("failed to rollback %s: %w", op.Package, err)
			}
		} else {
			// Remove package
			_, err := m.packageManager.Remove(ctx, op.Package)
			if err != nil {
				return fmt.Errorf("failed to rollback %s: %w", op.Package, err)
			}
		}
	}

	return nil
}

// Validate validates package module arguments
func (m *UnifiedPackageModule) Validate(args map[string]interface{}) error {
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
				return fmt.Errorf("invalid state: %s (must be one of: present, absent, latest)", stateStr)
			}
		}
	}

	return nil
}

// failResult creates a failed result
func (m *UnifiedPackageModule) failResult(result types.TaskResult, message string) (types.TaskResult, error) {
	result.Success = false
	result.Failed = true
	result.Error = message
	result.Duration = time.Since(result.Timestamp)
	return result, fmt.Errorf("%s", message)
}

// ============================================================================
// SNAPSHOT MANAGEMENT
// ============================================================================

// CreateSnapshot creates a snapshot of current package system state
func (m *UnifiedPackageModule) CreateSnapshot(ctx context.Context, description string) (*SystemSnapshot, error) {
	packages, err := m.packageManager.ListInstalled(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list installed packages: %w", err)
	}

	snapshot := &SystemSnapshot{
		ID:          fmt.Sprintf("snapshot-%d", time.Now().Unix()),
		Timestamp:   time.Now(),
		Description: description,
		Packages:    make(map[string]*PackageState),
	}

	// Convert PackageInfo to PackageState
	for _, pkg := range packages {
		snapshot.Packages[pkg.Name] = &PackageState{
			Name:        pkg.Name,
			Installed:   pkg.Installed,
			Version:     pkg.Version,
			Repository:  pkg.Repository,
			LastChecked: time.Now(),
		}
	}

	// Calculate checksum
	data, _ := json.Marshal(snapshot.Packages)
	hash := sha256.Sum256(data)
	snapshot.Checksum = hex.EncodeToString(hash[:])

	return snapshot, nil
}

// RestoreSnapshot restores system to a previous snapshot
func (m *UnifiedPackageModule) RestoreSnapshot(ctx context.Context, snapshot *SystemSnapshot) error {
	currentPackages, err := m.packageManager.ListInstalled(ctx)
	if err != nil {
		return fmt.Errorf("failed to list current packages: %w", err)
	}

	// Build current state map
	currentMap := make(map[string]bool)
	for _, pkg := range currentPackages {
		currentMap[pkg.Name] = true
	}

	// Remove packages not in snapshot
	for name := range currentMap {
		if _, exists := snapshot.Packages[name]; !exists {
			if _, err := m.packageManager.Remove(ctx, name); err != nil {
				return fmt.Errorf("failed to remove %s: %w", name, err)
			}
		}
	}

	// Install/update packages from snapshot
	for name, state := range snapshot.Packages {
		if _, err := m.packageManager.Install(ctx, name, state.Version); err != nil {
			return fmt.Errorf("failed to install %s: %w", name, err)
		}
	}

	return nil
}

// ============================================================================
// PACKAGE GROUP MANAGEMENT
// ============================================================================

// InstallGroup installs a package group
func (m *UnifiedPackageModule) InstallGroup(ctx context.Context, group *PackageGroup, includeOptional bool) (*BatchOperation, error) {
	packages := make([]PackageSpec, 0, len(group.Packages))

	for _, name := range group.Packages {
		packages = append(packages, PackageSpec{
			Name:  name,
			State: "present",
		})
	}

	if includeOptional {
		for _, name := range group.Optional {
			packages = append(packages, PackageSpec{
				Name:  name,
				State: "present",
			})
		}
	}

	return m.packageManager.InstallMultiple(ctx, packages)
}

// RemoveGroup removes a package group
func (m *UnifiedPackageModule) RemoveGroup(ctx context.Context, group *PackageGroup) (*BatchOperation, error) {
	packageNames := append([]string{}, group.Packages...)
	packageNames = append(packageNames, group.Optional...)

	return m.packageManager.RemoveMultiple(ctx, packageNames)
}

// ============================================================================
// HEALTH CHECK
// ============================================================================

// PerformHealthCheck performs a comprehensive health check
func (m *UnifiedPackageModule) PerformHealthCheck(ctx context.Context) (*HealthCheckResult, error) {
	result := &HealthCheckResult{
		Healthy:   true,
		CheckedAt: time.Now(),
	}

	// Verify package system integrity
	if err := m.packageManager.VerifyIntegrity(ctx); err != nil {
		result.Healthy = false
		result.Issues = append(result.Issues, fmt.Sprintf("Integrity check failed: %v", err))
	}

	// Check for broken packages
	if _, err := m.packageManager.ListInstalled(ctx); err != nil {
		result.Healthy = false
		result.Issues = append(result.Issues, fmt.Sprintf("Failed to list packages: %v", err))
		return result, nil
	}

	// Check for upgradable packages
	upgradable, err := m.packageManager.ListUpgradable(ctx)
	if err == nil && len(upgradable) > 0 {
		result.Warnings = append(result.Warnings, fmt.Sprintf("%d packages can be upgraded", len(upgradable)))
		result.Recommendations = append(result.Recommendations, "Consider upgrading packages to latest versions")
	}

	// Check cache statistics
	hits, misses := m.stateCache.Stats()
	if hits+misses > 0 {
		hitRate := float64(hits) / float64(hits+misses) * 100
		if hitRate < 50 {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Low cache hit rate: %.1f%%", hitRate))
			result.Recommendations = append(result.Recommendations, "Consider increasing cache TTL")
		}
	}

	// Check for orphan packages (packages with no dependencies)
	orphans, err := m.packageManager.AutoRemove(ctx)
	if err == nil && len(orphans) > 0 {
		result.OrphanPackages = orphans
		result.Recommendations = append(result.Recommendations, fmt.Sprintf("Remove %d orphan packages to free space", len(orphans)))
	}

	return result, nil
}

// ============================================================================
// AUDIT LOGGING
// ============================================================================

// LogAuditEntry logs an audit entry (to be implemented with actual logging backend)
func (m *UnifiedPackageModule) LogAuditEntry(entry *AuditEntry) error {
	// This is a placeholder - in production, this would write to a proper audit log
	// For now, we'll just format it as JSON
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal audit entry: %w", err)
	}

	// In production, write to audit log file or send to logging service
	_ = data // Placeholder to avoid unused variable error

	return nil
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// generateStateHash generates a hash for package state to detect changes
func generateStateHash(name, version, repository string) string {
	data := fmt.Sprintf("%s:%s:%s", name, version, repository)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}
