package parser

import (
	"context"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

type Parser struct{
	variables map[string]interface{}
}

func New() *Parser {
	return &Parser{
		variables: make(map[string]interface{}),
	}
}

// ParsePlaybook parses playbook from YAML file
func (p *Parser) ParsePlaybook(ctx context.Context, filePath string) (*types.Playbook, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading playbook: %w", err)
	}

	var playbook types.Playbook
	if err := yaml.Unmarshal(data, &playbook); err != nil {
		return nil, fmt.Errorf("error parsing playbook: %w", err)
	}

	// Validate playbook
	if err := p.ValidatePlaybook(&playbook); err != nil {
		return nil, fmt.Errorf("error validating playbook: %w", err)
	}

	return &playbook, nil
}

// ValidatePlaybook validates playbook correctness
func (p *Parser) ValidatePlaybook(playbook *types.Playbook) error {
	if playbook.Name == "" {
		return fmt.Errorf("playbook name cannot be empty")
	}

	if len(playbook.Plays) == 0 {
		return fmt.Errorf("playbook must contain at least one play")
	}

	for i, play := range playbook.Plays {
		if err := p.ValidatePlay(&play, i); err != nil {
			return err
		}
	}

	return nil
}

// ValidatePlay validates play correctness
func (p *Parser) ValidatePlay(play *types.Play, index int) error {
	if play.Name == "" {
		return fmt.Errorf("play #%d: play name cannot be empty", index+1)
	}

	if play.Hosts == "" {
		return fmt.Errorf("play #%d: hosts not specified", index+1)
	}

	if len(play.Tasks) == 0 {
		return fmt.Errorf("play #%d: play must contain at least one task", index+1)
	}

	for i, task := range play.Tasks {
		if err := p.ValidateTask(&task, index, i); err != nil {
			return err
		}
	}

	return nil
}

// ValidateTask validates task correctness
func (p *Parser) ValidateTask(task *types.Task, playIndex, taskIndex int) error {
	if task.Name == "" {
		return fmt.Errorf("play #%d, task #%d: task name cannot be empty", playIndex+1, taskIndex+1)
	}

	if task.Module == "" {
		return fmt.Errorf("play #%d, task #%d: module not specified", playIndex+1, taskIndex+1)
	}

	return nil
}

// ParseInventory parses inventory from YAML file
func (p *Parser) ParseInventory(ctx context.Context, filePath string) (*types.Inventory, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading inventory: %w", err)
	}

	var inventory types.Inventory
	if err := yaml.Unmarshal(data, &inventory); err != nil {
		return nil, fmt.Errorf("error parsing inventory: %w", err)
	}

	return &inventory, nil
}

// SetVariables sets global variables for template rendering
func (p *Parser) SetVariables(variables map[string]interface{}) {
	p.variables = variables
}

// AddVariable adds a single variable
func (p *Parser) AddVariable(key string, value interface{}) {
	if p.variables == nil {
		p.variables = make(map[string]interface{})
	}
	p.variables[key] = value
}
