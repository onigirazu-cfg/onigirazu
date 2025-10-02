package parser

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestNew(t *testing.T) {
	parser := New()
	assert.NotNil(t, parser)
	assert.NotNil(t, parser.variables)
	assert.Empty(t, parser.variables)
}

func TestParsePlaybook_ValidPlaybook(t *testing.T) {
	// Create a temporary playbook file
	tmpDir := t.TempDir()
	playbookPath := filepath.Join(tmpDir, "playbook.yml")

	playbookContent := `name: Test Playbook
plays:
  - name: Test Play
    hosts: all
    tasks:
      - name: Test Task
        module: debug
        args:
          msg: "Hello World"
`

	err := os.WriteFile(playbookPath, []byte(playbookContent), 0644)
	require.NoError(t, err)

	parser := New()
	ctx := context.Background()

	playbook, err := parser.ParsePlaybook(ctx, playbookPath)

	assert.NoError(t, err)
	assert.NotNil(t, playbook)
	assert.Equal(t, "Test Playbook", playbook.Name)
	assert.Len(t, playbook.Plays, 1)
	assert.Equal(t, "Test Play", playbook.Plays[0].Name)
	assert.Equal(t, "all", playbook.Plays[0].Hosts)
	assert.Len(t, playbook.Plays[0].Tasks, 1)
	assert.Equal(t, "Test Task", playbook.Plays[0].Tasks[0].Name)
	assert.Equal(t, "debug", playbook.Plays[0].Tasks[0].Module)
}

func TestParsePlaybook_FileNotFound(t *testing.T) {
	parser := New()
	ctx := context.Background()

	playbook, err := parser.ParsePlaybook(ctx, "/nonexistent/playbook.yml")

	assert.Error(t, err)
	assert.Nil(t, playbook)
	assert.Contains(t, err.Error(), "error reading playbook")
}

func TestParsePlaybook_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	playbookPath := filepath.Join(tmpDir, "invalid.yml")

	invalidContent := `name: Test
plays:
  - name: Test
    invalid yaml content [[[
`

	err := os.WriteFile(playbookPath, []byte(invalidContent), 0644)
	require.NoError(t, err)

	parser := New()
	ctx := context.Background()

	playbook, err := parser.ParsePlaybook(ctx, playbookPath)

	assert.Error(t, err)
	assert.Nil(t, playbook)
	assert.Contains(t, err.Error(), "error parsing playbook")
}

func TestParsePlaybook_ValidationError(t *testing.T) {
	tmpDir := t.TempDir()
	playbookPath := filepath.Join(tmpDir, "invalid_playbook.yml")

	// Playbook without name
	invalidContent := `plays:
  - name: Test Play
    hosts: all
    tasks:
      - name: Test Task
        module: debug
`

	err := os.WriteFile(playbookPath, []byte(invalidContent), 0644)
	require.NoError(t, err)

	parser := New()
	ctx := context.Background()

	playbook, err := parser.ParsePlaybook(ctx, playbookPath)

	assert.Error(t, err)
	assert.Nil(t, playbook)
	assert.Contains(t, err.Error(), "error validating playbook")
}

func TestValidatePlaybook_EmptyName(t *testing.T) {
	parser := New()
	playbook := &types.Playbook{
		Name: "",
		Plays: []types.Play{
			{
				Name:  "Test Play",
				Hosts: "all",
				Tasks: []types.Task{
					{Name: "Test Task", Module: "debug"},
				},
			},
		},
	}

	err := parser.ValidatePlaybook(playbook)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "playbook name cannot be empty")
}

func TestValidatePlaybook_NoPlays(t *testing.T) {
	parser := New()
	playbook := &types.Playbook{
		Name:  "Test Playbook",
		Plays: []types.Play{},
	}

	err := parser.ValidatePlaybook(playbook)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "playbook must contain at least one play")
}

func TestValidatePlaybook_ValidPlaybook(t *testing.T) {
	parser := New()
	playbook := &types.Playbook{
		Name: "Test Playbook",
		Plays: []types.Play{
			{
				Name:  "Test Play",
				Hosts: "all",
				Tasks: []types.Task{
					{Name: "Test Task", Module: "debug"},
				},
			},
		},
	}

	err := parser.ValidatePlaybook(playbook)

	assert.NoError(t, err)
}

func TestValidatePlay_EmptyName(t *testing.T) {
	parser := New()
	play := &types.Play{
		Name:  "",
		Hosts: "all",
		Tasks: []types.Task{
			{Name: "Test Task", Module: "debug"},
		},
	}

	err := parser.ValidatePlay(play, 0)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "play #1: play name cannot be empty")
}

func TestValidatePlay_EmptyHosts(t *testing.T) {
	parser := New()
	play := &types.Play{
		Name:  "Test Play",
		Hosts: "",
		Tasks: []types.Task{
			{Name: "Test Task", Module: "debug"},
		},
	}

	err := parser.ValidatePlay(play, 0)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "play #1: hosts not specified")
}

func TestValidatePlay_NoTasks(t *testing.T) {
	parser := New()
	play := &types.Play{
		Name:  "Test Play",
		Hosts: "all",
		Tasks: []types.Task{},
	}

	err := parser.ValidatePlay(play, 0)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "play #1: play must contain at least one task")
}

func TestValidatePlay_ValidPlay(t *testing.T) {
	parser := New()
	play := &types.Play{
		Name:  "Test Play",
		Hosts: "all",
		Tasks: []types.Task{
			{Name: "Test Task", Module: "debug"},
		},
	}

	err := parser.ValidatePlay(play, 0)

	assert.NoError(t, err)
}

func TestValidateTask_EmptyName(t *testing.T) {
	parser := New()
	task := &types.Task{
		Name:   "",
		Module: "debug",
	}

	err := parser.ValidateTask(task, 0, 0)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "play #1, task #1: task name cannot be empty")
}

func TestValidateTask_EmptyModule(t *testing.T) {
	parser := New()
	task := &types.Task{
		Name:   "Test Task",
		Module: "",
	}

	err := parser.ValidateTask(task, 0, 0)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "play #1, task #1: module not specified")
}

func TestValidateTask_ValidTask(t *testing.T) {
	parser := New()
	task := &types.Task{
		Name:   "Test Task",
		Module: "debug",
	}

	err := parser.ValidateTask(task, 0, 0)

	assert.NoError(t, err)
}

func TestParseInventory_ValidInventory(t *testing.T) {
	tmpDir := t.TempDir()
	inventoryPath := filepath.Join(tmpDir, "inventory.yml")

	inventoryContent := `all:
  hosts:
    host1:
      ansible_host: 192.168.1.1
    host2:
      ansible_host: 192.168.1.2
  vars:
    ansible_user: admin
`

	err := os.WriteFile(inventoryPath, []byte(inventoryContent), 0644)
	require.NoError(t, err)

	parser := New()
	ctx := context.Background()

	inventory, err := parser.ParseInventory(ctx, inventoryPath)

	assert.NoError(t, err)
	assert.NotNil(t, inventory)
}

func TestParseInventory_FileNotFound(t *testing.T) {
	parser := New()
	ctx := context.Background()

	inventory, err := parser.ParseInventory(ctx, "/nonexistent/inventory.yml")

	assert.Error(t, err)
	assert.Nil(t, inventory)
	assert.Contains(t, err.Error(), "error reading inventory")
}

func TestParseInventory_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	inventoryPath := filepath.Join(tmpDir, "invalid_inventory.yml")

	// Use truly invalid YAML that will fail parsing
	invalidContent := `all:
  hosts:
    host1:
      - invalid
      - structure
      - [unclosed bracket
`

	err := os.WriteFile(inventoryPath, []byte(invalidContent), 0644)
	require.NoError(t, err)

	parser := New()
	ctx := context.Background()

	inventory, err := parser.ParseInventory(ctx, inventoryPath)

	assert.Error(t, err)
	assert.Nil(t, inventory)
	assert.Contains(t, err.Error(), "error parsing inventory")
}

func TestSetVariables(t *testing.T) {
	parser := New()

	vars := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
		"key3": true,
	}

	parser.SetVariables(vars)

	assert.Equal(t, vars, parser.variables)
}

func TestAddVariable(t *testing.T) {
	parser := New()

	parser.AddVariable("key1", "value1")
	parser.AddVariable("key2", 123)

	assert.Equal(t, "value1", parser.variables["key1"])
	assert.Equal(t, 123, parser.variables["key2"])
	assert.Len(t, parser.variables, 2)
}

func TestAddVariable_NilVariables(t *testing.T) {
	parser := &Parser{
		variables: nil,
	}

	parser.AddVariable("key1", "value1")

	assert.NotNil(t, parser.variables)
	assert.Equal(t, "value1", parser.variables["key1"])
}

func TestParsePlaybook_MultiplePlaysTasks(t *testing.T) {
	tmpDir := t.TempDir()
	playbookPath := filepath.Join(tmpDir, "complex_playbook.yml")

	playbookContent := `name: Complex Playbook
plays:
  - name: First Play
    hosts: webservers
    tasks:
      - name: Install nginx
        module: package
        args:
          name: nginx
          state: present
      - name: Start nginx
        module: service
        args:
          name: nginx
          state: started
  - name: Second Play
    hosts: databases
    tasks:
      - name: Install postgresql
        module: package
        args:
          name: postgresql
          state: present
`

	err := os.WriteFile(playbookPath, []byte(playbookContent), 0644)
	require.NoError(t, err)

	parser := New()
	ctx := context.Background()

	playbook, err := parser.ParsePlaybook(ctx, playbookPath)

	assert.NoError(t, err)
	assert.NotNil(t, playbook)
	assert.Equal(t, "Complex Playbook", playbook.Name)
	assert.Len(t, playbook.Plays, 2)

	// Validate first play
	assert.Equal(t, "First Play", playbook.Plays[0].Name)
	assert.Equal(t, "webservers", playbook.Plays[0].Hosts)
	assert.Len(t, playbook.Plays[0].Tasks, 2)
	assert.Equal(t, "Install nginx", playbook.Plays[0].Tasks[0].Name)
	assert.Equal(t, "package", playbook.Plays[0].Tasks[0].Module)

	// Validate second play
	assert.Equal(t, "Second Play", playbook.Plays[1].Name)
	assert.Equal(t, "databases", playbook.Plays[1].Hosts)
	assert.Len(t, playbook.Plays[1].Tasks, 1)
	assert.Equal(t, "Install postgresql", playbook.Plays[1].Tasks[0].Name)
}
