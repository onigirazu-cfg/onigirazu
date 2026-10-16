package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/cache"
	"github.com/onigirazu-cfg/onigirazu/internal/interfaces"
	"github.com/onigirazu-cfg/onigirazu/internal/inventory"
	"github.com/onigirazu-cfg/onigirazu/internal/parser"
	"github.com/onigirazu-cfg/onigirazu/internal/template"
)

// loadInventory loads the -i sources as apply does, for commands that only
// need to reach hosts
func loadInventory(ctx context.Context, log interfaces.Logger) (*inventory.Manager, error) {
	return loadInventoryFrom(ctx, log, inventoryPaths)
}

// inventorySources is what apply uses for a playbook when no -i was given:
// the paths, else an inventory file next to the playbook, else localhost
func inventorySources(log interfaces.Logger, playbook string, paths []string) []string {
	if len(paths) > 0 {
		return paths
	}
	if playbook != "" {
		p := parser.NewEnhancedParser(template.NewEngine(), log)
		if found, err := p.FindInventoryFile(filepath.Dir(playbook)); err == nil {
			return []string{found}
		}
	}
	return []string{"localhost,"}
}

// loadInventoryFrom loads the given sources
func loadInventoryFrom(ctx context.Context, log interfaces.Logger, paths []string) (*inventory.Manager, error) {
	cacheManager := cache.NewManager(5 * time.Minute)
	p := parser.NewEnhancedParser(template.NewEngine(), log)
	manager := inventory.NewManager(p, log, cacheManager)
	merged, err := inventory.NewMultiSourceLoader(p, log, cacheManager, 10*time.Minute).LoadFromMultipleSources(ctx, paths)
	if err != nil {
		return nil, fmt.Errorf("failed to load inventory: %w", err)
	}
	if err := manager.SetInventory(merged); err != nil {
		return nil, fmt.Errorf("failed to load inventory: %w", err)
	}
	return manager, nil
}
