package core // import "github.com/onigirazu-cfg/onigirazu/internal/core"


TYPES

type CoreEngine struct {
	// Has unexported fields.
}
    CoreEngine represents the core execution engine for Onigirazu

func NewCoreEngine(logger *logger.Logger) *CoreEngine
    NewCoreEngine creates a new instance of the core engine

func (e *CoreEngine) Run(playbookPath, inventoryPath string, checkMode bool, stateFile string) error
    Run executes playbook

