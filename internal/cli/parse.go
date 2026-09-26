package cli

import (
	"context"
	"io"

	"github.com/onigirazu-cfg/onigirazu/internal/logger"
	"github.com/onigirazu-cfg/onigirazu/internal/modules"
	"github.com/onigirazu-cfg/onigirazu/internal/parser"
	"github.com/onigirazu-cfg/onigirazu/internal/template"
	"github.com/onigirazu-cfg/onigirazu/internal/validator"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// parsePlaybook reads a playbook the way apply does (includes, roles,
// short forms), for commands that only look at it
func parsePlaybook(ctx context.Context, path string) (*types.Playbook, error) {
	log := logger.NewEnhanced("error", logger.LogFormat("text"), io.Discard)
	p := parser.NewEnhancedParser(template.NewEngine(), log)
	// unknown modules are reported as apply reports them
	p.SetModuleSyntaxValidator(validator.NewModuleSyntaxValidator(modules.NewRegistry().ListModules()))
	return p.ParsePlaybook(ctx, path)
}
