package adhoc

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Parser handles parsing of ad-hoc commands in multiple formats
type Parser struct{}

// NewParser creates a new command parser
func NewParser() *Parser {
	return &Parser{}
}

// Parse attempts to parse a command string using multiple strategies
func (p *Parser) Parse(commandStr string, moduleName string, args []string) (*Command, error) {
	// If module name is explicitly provided (Ansible-like syntax)
	if moduleName != "" {
		return p.parseAnsibleLike(moduleName, args)
	}

	// Try to detect format and parse accordingly
	commandStr = strings.TrimSpace(commandStr)

	// Try JSON format
	if strings.HasPrefix(commandStr, "{") {
		if cmd, err := p.parseJSON(commandStr); err == nil {
			return cmd, nil
		}
	}

	// Try YAML format
	if strings.Contains(commandStr, "module:") || strings.Contains(commandStr, "args:") {
		if cmd, err := p.parseYAML(commandStr); err == nil {
			return cmd, nil
		}
	}

	// Try simple module:args syntax (before natural language to avoid conflicts)
	if strings.Contains(commandStr, ":") && !strings.Contains(commandStr, " ") {
		if cmd, err := p.parseModuleArgsSyntax(commandStr); err == nil {
			return cmd, nil
		}
	}

	// Try natural language
	if cmd, err := p.parseNaturalLanguage(commandStr); err == nil {
		return cmd, nil
	}

	// Fallback: treat as simple shell command
	return p.parseSimpleCommand(commandStr), nil
}

// parseSimpleCommand treats the input as a simple shell command
func (p *Parser) parseSimpleCommand(commandStr string) *Command {
	return &Command{
		Module: "command",
		Args: map[string]interface{}{
			"command": commandStr,
		},
		Format: FormatSimple,
		Raw:    commandStr,
	}
}

// parseAnsibleLike parses Ansible-like syntax: -m module key=value key=value
func (p *Parser) parseAnsibleLike(moduleName string, args []string) (*Command, error) {
	cmd := &Command{
		Module: moduleName,
		Args:   make(map[string]interface{}),
		Format: FormatAnsibleLike,
		Raw:    fmt.Sprintf("-m %s %s", moduleName, strings.Join(args, " ")),
	}

	// Parse key=value pairs
	for _, arg := range args {
		if err := p.parseKeyValue(arg, cmd.Args); err != nil {
			return nil, fmt.Errorf("failed to parse argument %q: %w", arg, err)
		}
	}

	return cmd, nil
}

// parseNaturalLanguage parses natural language commands
func (p *Parser) parseNaturalLanguage(commandStr string) (*Command, error) {
	words := strings.Fields(strings.ToLower(commandStr))
	if len(words) < 2 {
		return nil, fmt.Errorf("invalid natural language command")
	}

	action := words[0]
	target := words[1]

	cmd := &Command{
		Args:   make(map[string]interface{}),
		Format: FormatNaturalLanguage,
		Raw:    commandStr,
	}

	// Package operations
	if target == "package" || (len(words) > 2 && words[2] == "package") {
		return p.parsePackageCommand(action, words, cmd)
	}

	// Service operations
	if target == "service" || (len(words) > 2 && words[2] == "service") {
		return p.parseServiceCommand(action, words, cmd)
	}

	// File operations
	if action == "create" || action == "delete" || action == "touch" {
		return p.parseFileCommand(action, words, cmd)
	}

	return nil, fmt.Errorf("unknown natural language pattern: %s", commandStr)
}

// parsePackageCommand parses package-related natural language commands
func (p *Parser) parsePackageCommand(action string, words []string, cmd *Command) (*Command, error) {
	cmd.Module = "package"

	// Extract package name (word before "package" or after action)
	var packageName string
	for i, word := range words {
		if word != "package" && word != action && word != "the" && word != "a" {
			packageName = word
			break
		}
		if word == "package" && i > 0 {
			packageName = words[i-1]
			break
		}
	}

	if packageName == "" {
		return nil, fmt.Errorf("package name not found")
	}

	cmd.Args["name"] = packageName

	// Determine state based on action
	switch action {
	case "install", "add":
		cmd.Args["state"] = "present"
	case "remove", "uninstall", "delete":
		cmd.Args["state"] = "absent"
	case "update", "upgrade":
		cmd.Args["state"] = "latest"
	default:
		return nil, fmt.Errorf("unknown package action: %s", action)
	}

	return cmd, nil
}

// parseServiceCommand parses service-related natural language commands
func (p *Parser) parseServiceCommand(action string, words []string, cmd *Command) (*Command, error) {
	cmd.Module = "service"

	// Extract service name
	var serviceName string
	for i, word := range words {
		if word != "service" && word != action && word != "the" {
			serviceName = word
			break
		}
		if word == "service" && i > 0 {
			serviceName = words[i-1]
			break
		}
	}

	if serviceName == "" {
		return nil, fmt.Errorf("service name not found")
	}

	cmd.Args["name"] = serviceName

	// Determine state based on action
	switch action {
	case "start":
		cmd.Args["state"] = "started"
	case "stop":
		cmd.Args["state"] = "stopped"
	case "restart":
		cmd.Args["state"] = "restarted"
	case "reload":
		cmd.Args["state"] = "reloaded"
	default:
		return nil, fmt.Errorf("unknown service action: %s", action)
	}

	return cmd, nil
}

// parseFileCommand parses file-related natural language commands
func (p *Parser) parseFileCommand(action string, words []string, cmd *Command) (*Command, error) {
	cmd.Module = "file"

	// Extract file path (usually the last word or after "file")
	var filePath string
	for i := len(words) - 1; i >= 0; i-- {
		if words[i] != "file" && words[i] != action {
			filePath = words[i]
			break
		}
	}

	if filePath == "" {
		return nil, fmt.Errorf("file path not found")
	}

	cmd.Args["path"] = filePath

	// Determine state based on action
	switch action {
	case "create":
		cmd.Args["state"] = "touch"
	case "delete", "remove":
		cmd.Args["state"] = "absent"
	case "touch":
		cmd.Args["state"] = "touch"
	default:
		return nil, fmt.Errorf("unknown file action: %s", action)
	}

	return cmd, nil
}

// parseJSON parses JSON format commands
func (p *Parser) parseJSON(commandStr string) (*Command, error) {
	var data struct {
		Module string                 `json:"module"`
		Args   map[string]interface{} `json:"args"`
	}

	if err := json.Unmarshal([]byte(commandStr), &data); err != nil {
		return nil, err
	}

	if data.Module == "" {
		return nil, fmt.Errorf("module name is required")
	}

	return &Command{
		Module: data.Module,
		Args:   data.Args,
		Format: FormatJSON,
		Raw:    commandStr,
	}, nil
}

// parseYAML parses YAML format commands
func (p *Parser) parseYAML(commandStr string) (*Command, error) {
	var data struct {
		Module string                 `yaml:"module"`
		Args   map[string]interface{} `yaml:"args"`
	}

	if err := yaml.Unmarshal([]byte(commandStr), &data); err != nil {
		return nil, err
	}

	if data.Module == "" {
		return nil, fmt.Errorf("module name is required")
	}

	return &Command{
		Module: data.Module,
		Args:   data.Args,
		Format: FormatYAML,
		Raw:    commandStr,
	}, nil
}

// parseModuleArgsSyntax parses module:args syntax
func (p *Parser) parseModuleArgsSyntax(commandStr string) (*Command, error) {
	parts := strings.SplitN(commandStr, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid module:args syntax")
	}

	moduleName := strings.TrimSpace(parts[0])
	argsStr := strings.TrimSpace(parts[1])

	cmd := &Command{
		Module: moduleName,
		Args:   make(map[string]interface{}),
		Format: FormatAnsibleLike,
		Raw:    commandStr,
	}

	// Parse comma-separated key=value pairs
	argPairs := strings.Split(argsStr, ",")
	for _, pair := range argPairs {
		if err := p.parseKeyValue(strings.TrimSpace(pair), cmd.Args); err != nil {
			return nil, err
		}
	}

	return cmd, nil
}

// parseKeyValue parses a key=value pair and adds it to the args map
func (p *Parser) parseKeyValue(pair string, args map[string]interface{}) error {
	// Handle quoted values: key="value with spaces"
	re := regexp.MustCompile(`^([^=]+)=(.+)$`)
	matches := re.FindStringSubmatch(pair)
	if len(matches) != 3 {
		return fmt.Errorf("invalid key=value pair: %s", pair)
	}

	key := strings.TrimSpace(matches[1])
	value := strings.TrimSpace(matches[2])

	// Remove quotes if present
	if (strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`)) ||
		(strings.HasPrefix(value, `'`) && strings.HasSuffix(value, `'`)) {
		value = value[1 : len(value)-1]
	}

	// Try to parse as boolean
	if value == "true" || value == "yes" {
		args[key] = true
		return nil
	}
	if value == "false" || value == "no" {
		args[key] = false
		return nil
	}

	// Store as string (module will handle type conversion)
	args[key] = value
	return nil
}
