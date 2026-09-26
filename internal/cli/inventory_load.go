package cli

import (
	"context"
	"fmt"
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
	cacheManager := cache.NewManager(5 * time.Minute)
	p := parser.NewEnhancedParser(template.NewEngine(), log)
	manager := inventory.NewManager(p, log, cacheManager)
	merged, err := inventory.NewMultiSourceLoader(p, log, cacheManager, 10*time.Minute).LoadFromMultipleSources(ctx, inventoryPaths)
	if err != nil {
		return nil, fmt.Errorf("failed to load inventory: %w", err)
	}
	if err := manager.SetInventory(merged); err != nil {
		return nil, fmt.Errorf("failed to load inventory: %w", err)
	}
	return manager, nil
}
