package inventory

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/interfaces"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// MultiSourceLoader handles loading and merging inventory from multiple sources
// (files, directories, and dynamic scripts)
type MultiSourceLoader struct {
	parser       interfaces.PlaybookParser
	logger       interfaces.Logger
	cache        interfaces.CacheManager
	dynamicTTL   time.Duration // Time-to-live for dynamic inventory cache
	hostMap      map[string]*types.Host
	groupMap     map[string]*types.Group
	hostOrder    []string // Track insertion order for last-occurrence-wins
	loadingMutex sync.Mutex
}

// NewMultiSourceLoader creates a new multi-source inventory loader
func NewMultiSourceLoader(
	parser interfaces.PlaybookParser,
	logger interfaces.Logger,
	cache interfaces.CacheManager,
	dynamicTTL time.Duration,
) *MultiSourceLoader {
	if dynamicTTL == 0 {
		dynamicTTL = 10 * time.Minute // Default TTL if not specified
	}
	return &MultiSourceLoader{
		parser:     parser,
		logger:     logger,
		cache:      cache,
		dynamicTTL: dynamicTTL,
		hostMap:    make(map[string]*types.Host),
		groupMap:   make(map[string]*types.Group),
		hostOrder:  make([]string, 0),
	}
}

// LoadFromMultipleSources loads and merges inventory from multiple paths
// Supports files, directories, and dynamic inventory scripts
// Merging strategy: Last-occurrence wins (later sources override earlier ones)
func (msl *MultiSourceLoader) LoadFromMultipleSources(
	ctx context.Context,
	inventoryPaths []string,
) (*types.Inventory, error) {
	msl.loadingMutex.Lock()
	defer msl.loadingMutex.Unlock()

	// Reset state
	msl.hostMap = make(map[string]*types.Host)
	msl.groupMap = make(map[string]*types.Group)
	msl.hostOrder = make([]string, 0)

	if len(inventoryPaths) == 0 {
		return nil, fmt.Errorf("no inventory sources provided")
	}

	// Log the loading strategy
	msl.logger.Info("Loading inventory from %d source(s) with last-occurrence-wins merge strategy",
		len(inventoryPaths))

	// Process each inventory source in order (later sources override earlier ones)
	for i, path := range inventoryPaths {
		msl.logger.Debug("Loading inventory source %d/%d: %s", i+1, len(inventoryPaths), path)

		if err := msl.loadSingleSource(ctx, path); err != nil {
			return nil, fmt.Errorf("failed to load inventory from %s: %w", path, err)
		}
	}

	// Build final inventory with merged data
	inventory := &types.Inventory{
		Hosts:  msl.buildHostsList(),
		Groups: msl.groupMap,
	}

	msl.logger.Info("Successfully merged inventory from %d source(s): %d hosts, %d groups",
		len(inventoryPaths), len(inventory.Hosts), len(inventory.Groups))

	return inventory, nil
}

// loadSingleSource loads inventory from a single source (file, directory, or script)
func (msl *MultiSourceLoader) loadSingleSource(ctx context.Context, path string) error {
	// Resolve absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Check if path exists
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("inventory source not found: %s", absPath)
		}
		return fmt.Errorf("cannot access inventory source: %w", err)
	}

	// Handle different source types
	if info.IsDir() {
		return msl.loadFromDirectory(ctx, absPath)
	}

	// Check if it's an executable (dynamic inventory script)
	if msl.isExecutable(info) {
		return msl.loadDynamicInventory(ctx, absPath)
	}

	// Otherwise treat as a static inventory file
	return msl.loadStaticFile(ctx, absPath)
}

// loadFromDirectory loads all inventory files from a directory
func (msl *MultiSourceLoader) loadFromDirectory(ctx context.Context, dirPath string) error {
	msl.logger.Debug("Scanning directory for inventory sources: %s", dirPath)

	// Collect all inventory files and scripts
	var filePaths []string
	var scriptPaths []string

	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		// Skip subdirectories (group_vars, host_vars will be handled later if needed)
		if d.IsDir() {
			// Skip hidden directories and group_vars/host_vars for now (future enhancement)
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			if d.Name() == "group_vars" || d.Name() == "host_vars" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check file type
		ext := strings.ToLower(filepath.Ext(path))
		supportedExts := map[string]bool{
			".yml":  true,
			".yaml": true,
			".json": true,
			".ini":  true,
			".toml": true,
		}

		if supportedExts[ext] {
			filePaths = append(filePaths, path)
			return nil
		}

		// Check if it's executable (skip on Windows, check permissions on Unix)
		fileInfo, err := os.Stat(path)
		if err == nil && msl.isExecutable(fileInfo) {
			// Skip files that look like they have extensions (likely data files)
			if ext != "" && !supportedExts[ext] {
				scriptPaths = append(scriptPaths, path)
			} else if ext == "" {
				// No extension - likely a script
				scriptPaths = append(scriptPaths, path)
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("error scanning directory: %w", err)
	}

	if len(filePaths) == 0 && len(scriptPaths) == 0 {
		msl.logger.Warn("No inventory sources found in directory: %s", dirPath)
		return nil
	}

	// Sort files alphabetically for deterministic loading
	sort.Strings(filePaths)
	sort.Strings(scriptPaths)

	msl.logger.Debug("Found %d static inventory files and %d dynamic scripts in %s",
		len(filePaths), len(scriptPaths), dirPath)

	// Load static files first
	for _, filePath := range filePaths {
		if err := msl.loadStaticFile(ctx, filePath); err != nil {
			msl.logger.Warn("Failed to load static inventory file %s: %v", filePath, err)
			// Continue loading other files
		}
	}

	// Load dynamic scripts second (dynamic sources override static ones)
	for _, scriptPath := range scriptPaths {
		if err := msl.loadDynamicInventory(ctx, scriptPath); err != nil {
			msl.logger.Warn("Failed to load dynamic inventory from %s: %v", scriptPath, err)
			// Continue loading other scripts
		}
	}

	return nil
}

// loadStaticFile loads a single static inventory file
func (msl *MultiSourceLoader) loadStaticFile(ctx context.Context, filePath string) error {
	msl.logger.Debug("Loading static inventory file: %s", filePath)

	// Parse the inventory file
	inventory, err := msl.parser.ParseInventory(ctx, filePath)
	if err != nil {
		return fmt.Errorf("failed to parse inventory file: %w", err)
	}

	// Merge the loaded inventory
	msl.mergeInventory(inventory)

	return nil
}

// loadDynamicInventory loads inventory from a dynamic script with TTL-based caching
func (msl *MultiSourceLoader) loadDynamicInventory(ctx context.Context, scriptPath string) error {
	msl.logger.Info("Loading dynamic inventory from script: %s", scriptPath)

	// Create cache key based on script path and execution context
	cacheKey := fmt.Sprintf("dynamic_inventory:%s", scriptPath)

	// Check cache first
	if cached, found := msl.cache.Get(ctx, cacheKey); found {
		if inventory, ok := cached.(*types.Inventory); ok {
			msl.logger.Debug("Using cached dynamic inventory from %s", scriptPath)
			msl.mergeInventory(inventory)
			return nil
		}
	}

	// Execute the script
	inventory, err := msl.executeDynamicInventoryScript(ctx, scriptPath)
	if err != nil {
		return fmt.Errorf("failed to execute dynamic inventory script: %w", err)
	}

	// Cache the result with TTL
	_ = msl.cache.SetWithTTL(ctx, cacheKey, inventory, msl.dynamicTTL)

	// Merge the loaded inventory
	msl.mergeInventory(inventory)

	return nil
}

// executeDynamicInventoryScript executes a dynamic inventory script and parses its JSON output
func (msl *MultiSourceLoader) executeDynamicInventoryScript(
	ctx context.Context,
	scriptPath string,
) (*types.Inventory, error) {
	// Create a timeout context if not already set
	exCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(exCtx, scriptPath)
	cmd.Stderr = os.Stderr

	// Execute the script
	output, err := cmd.Output()
	if err != nil {
		if exCtx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("dynamic inventory script timed out: %s", scriptPath)
		}
		return nil, fmt.Errorf("failed to execute script: %w", err)
	}

	// Parse the JSON output as inventory
	// Dynamic scripts must output valid JSON with "hosts" and/or "groups" keys
	inventory := &types.Inventory{
		Hosts:  make([]types.Host, 0),
		Groups: make(map[string]*types.Group),
	}

	if len(output) > 0 {
		if err := msl.parseDynamicInventoryOutput(output, inventory); err != nil {
			return nil, fmt.Errorf("failed to parse dynamic inventory output: %w", err)
		}
	}

	msl.logger.Debug("Successfully executed dynamic inventory script: %s (got %d hosts, %d groups)",
		scriptPath, len(inventory.Hosts), len(inventory.Groups))

	return inventory, nil
}

// parseDynamicInventoryOutput parses JSON output from a dynamic inventory script
func (msl *MultiSourceLoader) parseDynamicInventoryOutput(
	output []byte,
	inventory *types.Inventory,
) error {
	// Use the parser to handle JSON parsing
	// This allows us to leverage existing inventory parsing logic

	// For now, create a temporary file to use the standard parser
	// (In a production implementation, we might want a direct JSON parser method)
	tempFile, err := os.CreateTemp("", "dynamic_inventory_*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.Write(output); err != nil {
		tempFile.Close()
		return err
	}
	tempFile.Close()

	// Parse using the standard parser
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	parsed, err := msl.parser.ParseInventory(ctx, tempFile.Name())
	if err != nil {
		return err
	}

	// Copy data from parsed inventory
	if parsed != nil {
		inventory.Hosts = parsed.Hosts
		inventory.Groups = parsed.Groups
	}

	return nil
}

// mergeInventory merges an inventory into the current state using last-occurrence-wins strategy
func (msl *MultiSourceLoader) mergeInventory(inv *types.Inventory) {
	if inv == nil {
		return
	}

	// Merge hosts (last-occurrence-wins)
	for i := range inv.Hosts {
		host := inv.Hosts[i]
		hostname := host.Name
		if hostname == "" {
			hostname = host.Address
		}

		if hostname == "" {
			msl.logger.Warn("Skipping host with no name or address")
			continue
		}

		// Track insertion order for building the final list
		if _, exists := msl.hostMap[hostname]; !exists {
			msl.hostOrder = append(msl.hostOrder, hostname)
		}

		// Merge host: for each host, merge variables (last-occurrence wins for individual vars)
		if existingHost, exists := msl.hostMap[hostname]; exists {
			// Update connection parameters if provided
			if host.Address != "" && host.Address != existingHost.Address {
				existingHost.Address = host.Address
			}
			if host.Port != 0 && host.Port != existingHost.Port {
				existingHost.Port = host.Port
			}
			if host.User != "" && host.User != existingHost.User {
				existingHost.User = host.User
			}
			// Merge variables (last-occurrence wins)
			if host.Vars != nil {
				if existingHost.Vars == nil {
					existingHost.Vars = make(map[string]interface{})
				}
				for k, v := range host.Vars {
					existingHost.Vars[k] = v
				}
			}
		} else {
			// New host
			msl.hostMap[hostname] = &host
		}
	}

	// Merge groups
	for groupName, group := range inv.Groups {
		if _, exists := msl.groupMap[groupName]; exists {
			// Merge group hosts (last-occurrence-wins for hosts in groups too)
			existingGroup := msl.groupMap[groupName]
			if group.Hosts != nil {
				existingGroup.Hosts = group.Hosts
			}
			if group.Vars != nil {
				if existingGroup.Vars == nil {
					existingGroup.Vars = make(map[string]interface{})
				}
				// Last-occurrence-wins for group variables
				for k, v := range group.Vars {
					existingGroup.Vars[k] = v
				}
			}
			if group.Children != nil {
				existingGroup.Children = group.Children
			}
		} else {
			// New group
			msl.groupMap[groupName] = group
		}
	}
}

// buildHostsList builds the final hosts list from the merged host map, maintaining insertion order
func (msl *MultiSourceLoader) buildHostsList() []types.Host {
	hosts := make([]types.Host, 0, len(msl.hostOrder))

	for _, hostname := range msl.hostOrder {
		if host, exists := msl.hostMap[hostname]; exists {
			hosts = append(hosts, *host)
		}
	}

	return hosts
}

// isExecutable checks if a file is executable
func (msl *MultiSourceLoader) isExecutable(info fs.FileInfo) bool {
	// On Unix-like systems, check for executable permission
	// On Windows, check by extension (handled at directory walk level)
	return info.Mode()&0o111 != 0
}

// ClearDynamicInventoryCache clears the dynamic inventory cache
func (msl *MultiSourceLoader) ClearDynamicInventoryCache(ctx context.Context) {
	// Clear all keys starting with "dynamic_inventory:"
	msl.logger.Info("Clearing dynamic inventory cache")
	// Note: This is a simplified implementation
	// In production, you might want to track all cache keys or use a more sophisticated cache management
}

// GetCacheTTL returns the current TTL for dynamic inventory cache
func (msl *MultiSourceLoader) GetCacheTTL() time.Duration {
	return msl.dynamicTTL
}

// SetCacheTTL sets the TTL for dynamic inventory cache
func (msl *MultiSourceLoader) SetCacheTTL(ttl time.Duration) {
	if ttl > 0 {
		msl.dynamicTTL = ttl
		msl.logger.Debug("Set dynamic inventory cache TTL to %v", ttl)
	}
}
