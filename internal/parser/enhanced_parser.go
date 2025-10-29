package parser

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/onigirazu-cfg/onigirazu/internal/interfaces"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// EnhancedParser provides advanced playbook parsing capabilities
type EnhancedParser struct {
	templateEngine  interfaces.TemplateEngine
	logger          interfaces.Logger
	variables       map[string]interface{}
	inventoryParser *InventoryParser
	roleLoader      *RoleLoader // NEW: For loading roles
	lenient         bool        // NEW: Lenient mode for inventory parsing
}

// NewEnhancedParser creates a new enhanced parser
func NewEnhancedParser(templateEngine interfaces.TemplateEngine, logger interfaces.Logger) *EnhancedParser {
	parser := &EnhancedParser{
		templateEngine: templateEngine,
		logger:         logger,
		variables:      make(map[string]interface{}),
		lenient:        false, // Default to strict mode
	}
	// Create inventory parser
	parser.inventoryParser = NewInventoryParser(logger)
	// Create role loader (default roles path: playbooks/roles)
	parser.roleLoader = NewRoleLoader(logger, "playbooks/roles")
	return parser
}

// SetLenient sets lenient mode for inventory parsing
func (p *EnhancedParser) SetLenient(lenient bool) {
	p.lenient = lenient
	if lenient {
		p.logger.Info("Lenient mode enabled: inventory validation is relaxed")
	}
}

// ParsePlaybook parses a playbook file with template rendering
func (p *EnhancedParser) ParsePlaybook(ctx context.Context, filePath string) (*types.Playbook, error) {
	p.logger.Debug("Parsing playbook: %s", filePath)

	// Read file content
	content, err := os.ReadFile(filePath) // #nosec G304 -- filePath is provided by user as playbook file
	if err != nil {
		return nil, fmt.Errorf("failed to read playbook file %s: %w", filePath, err)
	}

	// NOTE: We do NOT render templates at parse time!
	// Template rendering happens during task execution when variables are available.
	// Rendering here would replace {{ variable }} with <no value> before execution.

	// Parse YAML directly without template rendering
	var playbook types.Playbook
	if err := yaml.Unmarshal(content, &playbook); err != nil {
		return nil, fmt.Errorf("failed to parse YAML in playbook %s: %w", filePath, err)
	}

	// Set playbook metadata
	playbook.FilePath = filePath
	playbook.Name = filepath.Base(filePath)

	// Validate and process playbook
	if err := p.validatePlaybook(&playbook); err != nil {
		return nil, fmt.Errorf("playbook validation failed for %s: %w", filePath, err)
	}

	// Process includes and imports
	if err := p.processIncludes(ctx, &playbook, filepath.Dir(filePath)); err != nil {
		return nil, fmt.Errorf("failed to process includes in playbook %s: %w", filePath, err)
	}

	// Load roles for each play (NEW v1.42.0)
	if err := p.processRoles(ctx, &playbook, filepath.Dir(filePath)); err != nil {
		return nil, fmt.Errorf("failed to process roles in playbook %s: %w", filePath, err)
	}

	p.logger.Info("Successfully parsed playbook: %s (%d plays)", filePath, len(playbook.Plays))
	return &playbook, nil
}

// ParseInventory parses an inventory file or inline specification
// Supports file formats: YAML, TOML, JSON, INI, and simple list formats
// Also supports inline host specifications like "192.168.1.1," or "host1,host2"
func (p *EnhancedParser) ParseInventory(ctx context.Context, filePath string) (*types.Inventory, error) {
	p.logger.Debug("Parsing inventory: %s", filePath)

	// Use the new inventory parser that handles both file-based and inline inventory
	inventory, err := p.inventoryParser.ParseInventoryOrInline(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse inventory %s: %w", filePath, err)
	}

	// Validate inventory (unless in lenient mode)
	if !p.lenient {
		if err := p.validateInventory(inventory); err != nil {
			return nil, fmt.Errorf("inventory validation failed for %s: %w", filePath, err)
		}
	} else {
		// In lenient mode, just warn about issues instead of failing
		if err := p.validateInventoryLenient(inventory); err != nil {
			p.logger.Warn("Inventory validation issues (lenient mode): %v", err)
		}
	}

	p.logger.Info("Successfully parsed inventory: %s (%d groups, %d hosts)",
		filePath, len(inventory.Groups), p.countHosts(inventory))
	return inventory, nil
}

// FindInventoryFile searches for inventory file in common locations
func (p *EnhancedParser) FindInventoryFile(baseDir string) (string, error) {
	return p.inventoryParser.FindInventoryFile(baseDir)
}

// SetVariables sets global variables for template rendering
func (p *EnhancedParser) SetVariables(variables map[string]interface{}) {
	p.variables = variables
}

// AddVariable adds a single variable
func (p *EnhancedParser) AddVariable(key string, value interface{}) {
	if p.variables == nil {
		p.variables = make(map[string]interface{})
	}
	p.variables[key] = value
}

// ValidatePlaybook validates playbook structure (public interface method)
func (p *EnhancedParser) ValidatePlaybook(playbook *types.Playbook) error {
	return p.validatePlaybook(playbook)
}

// validatePlaybook validates playbook structure
func (p *EnhancedParser) validatePlaybook(playbook *types.Playbook) error {
	if len(playbook.Plays) == 0 {
		return fmt.Errorf("playbook must contain at least one play")
	}

	for i, play := range playbook.Plays {
		if err := p.validatePlay(&play, i); err != nil {
			return fmt.Errorf("play %d validation failed: %w", i, err)
		}
	}

	return nil
}

// validatePlay validates a single play
func (p *EnhancedParser) validatePlay(play *types.Play, index int) error {
	if play.Name == "" {
		play.Name = fmt.Sprintf("Play %d", index+1)
	}

	if len(play.Hosts) == 0 {
		return fmt.Errorf("play '%s' must specify hosts", play.Name)
	}

	if len(play.Tasks) == 0 && len(play.PreTasks) == 0 && len(play.PostTasks) == 0 {
		return fmt.Errorf("play '%s' must contain at least one task", play.Name)
	}

	// Validate tasks
	for i, task := range play.Tasks {
		if err := p.validateTask(&task, fmt.Sprintf("%s.task[%d]", play.Name, i)); err != nil {
			return err
		}
	}

	// Validate pre-tasks
	for i, task := range play.PreTasks {
		if err := p.validateTask(&task, fmt.Sprintf("%s.pre_task[%d]", play.Name, i)); err != nil {
			return err
		}
	}

	// Validate post-tasks
	for i, task := range play.PostTasks {
		if err := p.validateTask(&task, fmt.Sprintf("%s.post_task[%d]", play.Name, i)); err != nil {
			return err
		}
	}

	return nil
}

// validateTask validates a single task
func (p *EnhancedParser) validateTask(task *types.Task, context string) error {
	if task.Module == "" {
		return fmt.Errorf("task in %s must specify a module", context)
	}

	if task.Name == "" {
		task.Name = fmt.Sprintf("%s task", task.Module)
	}

	// Validate loop syntax
	if task.Loop != nil {
		if err := p.validateLoop(task.Loop, context); err != nil {
			return err
		}
	}

	// Validate conditional syntax
	if task.When != "" {
		if err := p.validateCondition(task.When, context); err != nil {
			return err
		}
	}

	return nil
}

// validateLoop validates loop syntax
func (p *EnhancedParser) validateLoop(loop *types.Loop, context string) error {
	if loop.Items == nil && loop.Range == "" {
		return fmt.Errorf("loop in %s must specify either 'items' or 'range'", context)
	}

	if loop.Items != nil && loop.Range != "" {
		return fmt.Errorf("loop in %s cannot specify both 'items' and 'range'", context)
	}

	return nil
}

// validateCondition validates conditional syntax
func (p *EnhancedParser) validateCondition(condition, context string) error {
	// Basic validation - check for common syntax errors
	if strings.Contains(condition, "{{") && !strings.Contains(condition, "}}") {
		return fmt.Errorf("unclosed template expression in condition for %s", context)
	}

	return nil
}

// validateInventory validates inventory structure (strict mode)
func (p *EnhancedParser) validateInventory(inventory *types.Inventory) error {
	if len(inventory.Groups) == 0 {
		return fmt.Errorf("inventory must contain at least one group")
	}

	// Validate groups
	for groupName, group := range inventory.Groups {
		if err := p.validateGroup(group, groupName); err != nil {
			return fmt.Errorf("group '%s' validation failed: %w", groupName, err)
		}
	}

	return nil
}

// validateInventoryLenient validates inventory structure in lenient mode
// Issues are logged as warnings instead of errors
func (p *EnhancedParser) validateInventoryLenient(inventory *types.Inventory) error {
	var issues []string

	if len(inventory.Groups) == 0 {
		p.logger.Warn("Inventory contains no groups")
		if len(inventory.Hosts) == 0 {
			issues = append(issues, "inventory is empty (no hosts and no groups)")
		}
	}

	// Check groups in lenient mode - just warn about empty ones
	emptyGroups := []string{}
	for groupName, group := range inventory.Groups {
		if len(group.Hosts) == 0 && len(group.Children) == 0 {
			emptyGroups = append(emptyGroups, groupName)
		}

		// Validate hosts don't crash on nil
		for hostName, host := range group.Hosts {
			if err := p.validateHost(host, hostName); err != nil {
				p.logger.Warn("Host '%s' in group '%s': %v", hostName, groupName, err)
			}
		}
	}

	if len(emptyGroups) > 0 {
		p.logger.Warn("Found %d empty groups (no hosts or children): %v", len(emptyGroups), emptyGroups)
		issues = append(issues, fmt.Sprintf("empty groups: %v", emptyGroups))
	}

	if len(issues) > 0 {
		return fmt.Errorf("inventory has issues in lenient mode: %s", strings.Join(issues, "; "))
	}

	return nil
}

// validateGroup validates a single inventory group
func (p *EnhancedParser) validateGroup(group *types.Group, name string) error {
	if len(group.Hosts) == 0 && len(group.Children) == 0 {
		return fmt.Errorf("group '%s' must contain either hosts or children", name)
	}

	// Validate hosts
	for hostName, host := range group.Hosts {
		if err := p.validateHost(host, hostName); err != nil {
			return fmt.Errorf("host '%s' in group '%s' validation failed: %w", hostName, name, err)
		}
	}

	return nil
}

// validateHost validates a single host
func (p *EnhancedParser) validateHost(host *types.Host, name string) error {
	// Host can be nil if only name is specified in inventory
	if host == nil {
		return nil
	}

	if host.Address == "" {
		host.Address = name // Use hostname as address if not specified
	}

	// Set default SSH port if not specified
	if host.Port == 0 {
		host.Port = 22
	}

	return nil
}

// processIncludes processes include and import statements
func (p *EnhancedParser) processIncludes(ctx context.Context, playbook *types.Playbook, baseDir string) error {
	// Process includes in each play
	for i := range playbook.Plays {
		if err := p.processPlayIncludes(ctx, &playbook.Plays[i], baseDir); err != nil {
			return fmt.Errorf("failed to process includes in play %d: %w", i, err)
		}
	}

	return nil
}

// processPlayIncludes processes includes within a play
func (p *EnhancedParser) processPlayIncludes(ctx context.Context, play *types.Play, baseDir string) error {
	// Process task includes
	var expandedTasks []types.Task
	for _, task := range play.Tasks {
		if task.Include != "" {
			// Load included tasks
			includedTasks, err := p.loadIncludedTasks(ctx, task.Include, baseDir)
			if err != nil {
				return fmt.Errorf("failed to load included tasks from %s: %w", task.Include, err)
			}
			expandedTasks = append(expandedTasks, includedTasks...)
		} else {
			expandedTasks = append(expandedTasks, task)
		}
	}
	play.Tasks = expandedTasks

	return nil
}

// loadIncludedTasks loads tasks from an included file
func (p *EnhancedParser) loadIncludedTasks(ctx context.Context, includePath, baseDir string) ([]types.Task, error) {
	fullPath := filepath.Join(baseDir, includePath)

	content, err := os.ReadFile(fullPath) // #nosec G304 -- fullPath is constructed from baseDir and include path
	if err != nil {
		return nil, fmt.Errorf("failed to read included file %s: %w", fullPath, err)
	}

	// Render templates
	renderedContent, err := p.templateEngine.Render(ctx, string(content), p.variables)
	if err != nil {
		return nil, fmt.Errorf("failed to render templates in included file %s: %w", fullPath, err)
	}

	var tasks []types.Task
	if err := yaml.Unmarshal([]byte(renderedContent), &tasks); err != nil {
		return nil, fmt.Errorf("failed to parse YAML in included file %s: %w", fullPath, err)
	}

	// Validate included tasks
	for i, task := range tasks {
		if err := p.validateTask(&task, fmt.Sprintf("included[%s].task[%d]", includePath, i)); err != nil {
			return nil, err
		}
	}

	p.logger.Debug("Loaded %d tasks from included file: %s", len(tasks), includePath)
	return tasks, nil
}

// countHosts counts total number of hosts in inventory
func (p *EnhancedParser) countHosts(inventory *types.Inventory) int {
	count := 0
	for _, group := range inventory.Groups {
		count += len(group.Hosts)
	}
	return count
}

// GetSupportedFormats returns list of supported file formats
func (p *EnhancedParser) GetSupportedFormats() []string {
	return []string{"yaml", "yml"}
}

// ValidateFile validates file format and basic structure
func (p *EnhancedParser) ValidateFile(filePath string) error {
	// Check file extension
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext != ".yaml" && ext != ".yml" {
		return fmt.Errorf("unsupported file format: %s (supported: .yaml, .yml)", ext)
	}

	// Check if file exists and is readable
	content, err := os.ReadFile(filePath) // #nosec G304 -- filePath is provided by user for validation
	if err != nil {
		return fmt.Errorf("cannot read file %s: %w", filePath, err)
	}

	// Basic YAML syntax validation
	var temp interface{}
	if err := yaml.Unmarshal(content, &temp); err != nil {
		return fmt.Errorf("invalid YAML syntax in %s: %w", filePath, err)
	}

	return nil
}

// processRoles loads roles for each play in the playbook (NEW v1.42.0)
func (p *EnhancedParser) processRoles(ctx context.Context, playbook *types.Playbook, baseDir string) error {
	for i, play := range playbook.Plays {
		if len(play.Roles) == 0 {
			continue // No roles to process
		}

		p.logger.Debug("Loading %d roles for play %d: %s", len(play.Roles), i, play.Name)

		// Update role loader's path if playbook has custom path
		p.roleLoader.rolesPath = filepath.Join(baseDir, "roles")

		// Load each role
		for _, roleRef := range play.Roles {
			role, err := p.roleLoader.LoadRole(ctx, roleRef)
			if err != nil {
				return fmt.Errorf("failed to load role %s: %w", roleRef.Name, err)
			}
			playbook.Plays[i].RoleObjects = append(playbook.Plays[i].RoleObjects, role)
		}

		p.logger.Debug("Loaded %d roles for play %d", len(playbook.Plays[i].RoleObjects), i)
	}
	return nil
}
