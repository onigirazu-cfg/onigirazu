package parser // import "github.com/onigirazu-cfg/onigirazu/internal/parser"


TYPES

type EnhancedParser struct {
	// Has unexported fields.
}
    EnhancedParser provides advanced playbook parsing capabilities

func NewEnhancedParser(templateEngine interfaces.TemplateEngine, logger interfaces.Logger) *EnhancedParser
    NewEnhancedParser creates a new enhanced parser

func (p *EnhancedParser) AddVariable(key string, value interface{})
    AddVariable adds a single variable

func (p *EnhancedParser) GetSupportedFormats() []string
    GetSupportedFormats returns list of supported file formats

func (p *EnhancedParser) ParseInventory(ctx context.Context, filePath string) (*types.Inventory, error)
    ParseInventory parses an inventory file

func (p *EnhancedParser) ParsePlaybook(ctx context.Context, filePath string) (*types.Playbook, error)
    ParsePlaybook parses a playbook file with template rendering

func (p *EnhancedParser) SetVariables(variables map[string]interface{})
    SetVariables sets global variables for template rendering

func (p *EnhancedParser) ValidateFile(filePath string) error
    ValidateFile validates file format and basic structure

func (p *EnhancedParser) ValidatePlaybook(playbook *types.Playbook) error
    ValidatePlaybook validates playbook structure (public interface method)

type Parser struct {
	// Has unexported fields.
}

func New() *Parser

func (p *Parser) AddVariable(key string, value interface{})
    AddVariable adds a single variable

func (p *Parser) ParseInventory(ctx context.Context, filePath string) (*types.Inventory, error)
    ParseInventory parses inventory from YAML file

func (p *Parser) ParsePlaybook(ctx context.Context, filePath string) (*types.Playbook, error)
    ParsePlaybook parses playbook from YAML file

func (p *Parser) SetVariables(variables map[string]interface{})
    SetVariables sets global variables for template rendering

func (p *Parser) ValidatePlay(play *types.Play, index int) error
    ValidatePlay validates play correctness

func (p *Parser) ValidatePlaybook(playbook *types.Playbook) error
    ValidatePlaybook validates playbook correctness

func (p *Parser) ValidateTask(task *types.Task, playIndex, taskIndex int) error
    ValidateTask validates task correctness

