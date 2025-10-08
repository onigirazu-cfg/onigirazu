package parser

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// mockLogger implements a simple logger for testing
type mockLogger struct {
	debugMessages []string
	infoMessages  []string
	warnMessages  []string
}

func (m *mockLogger) Debug(format string, args ...interface{})                                {}
func (m *mockLogger) Info(format string, args ...interface{})                                 {}
func (m *mockLogger) Warn(format string, args ...interface{})                                 {}
func (m *mockLogger) Error(format string, args ...interface{})                                {}
func (m *mockLogger) Fatal(format string, args ...interface{})                                {}
func (m *mockLogger) SetLevel(level string)                                                   {}
func (m *mockLogger) TaskStart(taskName, hostName string)                                     {}
func (m *mockLogger) TaskEnd(taskName, hostName string, changed, success bool)                {}
func (m *mockLogger) PlayStart(playName string, playIndex, totalPlays int)                    {}
func (m *mockLogger) PlayEnd(playName, hostName string, success bool, duration time.Duration) {}
func (m *mockLogger) Progress(completed, total int, currentTask, currentHost string)          {}
func (m *mockLogger) Retry(taskName, hostName string, attempt, maxAttempts int, delay time.Duration, err error) {
}

// mockTemplateEngine implements a simple template engine for testing
type mockTemplateEngine struct {
	renderFunc func(ctx context.Context, template string, vars map[string]interface{}) (string, error)
}

func (m *mockTemplateEngine) Render(ctx context.Context, template string, vars map[string]interface{}) (string, error) {
	if m.renderFunc != nil {
		return m.renderFunc(ctx, template, vars)
	}
	// Default: return template as-is
	return template, nil
}

func (m *mockTemplateEngine) RenderFile(ctx context.Context, filePath string, vars map[string]interface{}) (string, error) {
	return "", nil
}

func (m *mockTemplateEngine) RenderTaskArgs(ctx context.Context, args map[string]interface{}, vars map[string]interface{}) (map[string]interface{}, error) {
	return args, nil
}

func (m *mockTemplateEngine) ValidateTemplate(template string) error {
	return nil
}

func TestNewEnhancedParser(t *testing.T) {
	logger := &mockLogger{}
	templateEngine := &mockTemplateEngine{}

	parser := NewEnhancedParser(templateEngine, logger)

	assert.NotNil(t, parser)
	assert.NotNil(t, parser.variables)
	assert.NotNil(t, parser.inventoryParser)
	assert.Equal(t, templateEngine, parser.templateEngine)
	assert.Equal(t, logger, parser.logger)
}

func TestEnhancedParser_ParsePlaybook_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	playbookPath := filepath.Join(tmpDir, "test_playbook.yml")

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

	logger := &mockLogger{}
	templateEngine := &mockTemplateEngine{}
	parser := NewEnhancedParser(templateEngine, logger)
	ctx := context.Background()

	playbook, err := parser.ParsePlaybook(ctx, playbookPath)

	assert.NoError(t, err)
	assert.NotNil(t, playbook)
	// EnhancedParser sets name to filename, not from YAML
	assert.Equal(t, "test_playbook.yml", playbook.Name)
	assert.Equal(t, playbookPath, playbook.FilePath)
	assert.Len(t, playbook.Plays, 1)
	assert.Equal(t, "Test Play", playbook.Plays[0].Name)
}

func TestEnhancedParser_ParsePlaybook_FileNotFound(t *testing.T) {
	logger := &mockLogger{}
	templateEngine := &mockTemplateEngine{}
	parser := NewEnhancedParser(templateEngine, logger)
	ctx := context.Background()

	playbook, err := parser.ParsePlaybook(ctx, "/nonexistent/playbook.yml")

	assert.Error(t, err)
	assert.Nil(t, playbook)
	assert.Contains(t, err.Error(), "failed to read playbook file")
}

func TestEnhancedParser_ParsePlaybook_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	playbookPath := filepath.Join(tmpDir, "invalid.yml")

	invalidContent := `name: Test
plays:
  - name: Test
    invalid yaml [[[
`

	err := os.WriteFile(playbookPath, []byte(invalidContent), 0644)
	require.NoError(t, err)

	logger := &mockLogger{}
	templateEngine := &mockTemplateEngine{}
	parser := NewEnhancedParser(templateEngine, logger)
	ctx := context.Background()

	playbook, err := parser.ParsePlaybook(ctx, playbookPath)

	assert.Error(t, err)
	assert.Nil(t, playbook)
	assert.Contains(t, err.Error(), "failed to parse YAML")
}

func TestEnhancedParser_ParsePlaybook_ValidationError(t *testing.T) {
	tmpDir := t.TempDir()
	playbookPath := filepath.Join(tmpDir, "invalid_playbook.yml")

	// Playbook without plays
	invalidContent := `name: Test Playbook
plays: []
`

	err := os.WriteFile(playbookPath, []byte(invalidContent), 0644)
	require.NoError(t, err)

	logger := &mockLogger{}
	templateEngine := &mockTemplateEngine{}
	parser := NewEnhancedParser(templateEngine, logger)
	ctx := context.Background()

	playbook, err := parser.ParsePlaybook(ctx, playbookPath)

	assert.Error(t, err)
	assert.Nil(t, playbook)
	assert.Contains(t, err.Error(), "playbook validation failed")
}

func TestEnhancedParser_ValidatePlaybook_NoPlays(t *testing.T) {
	logger := &mockLogger{}
	templateEngine := &mockTemplateEngine{}
	parser := NewEnhancedParser(templateEngine, logger)

	playbook := &types.Playbook{
		Name:  "Test",
		Plays: []types.Play{},
	}

	err := parser.ValidatePlaybook(playbook)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "playbook must contain at least one play")
}

func TestEnhancedParser_ValidatePlaybook_Valid(t *testing.T) {
	logger := &mockLogger{}
	templateEngine := &mockTemplateEngine{}
	parser := NewEnhancedParser(templateEngine, logger)

	playbook := &types.Playbook{
		Name: "Test",
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

func TestEnhancedParser_ValidatePlay_NoName(t *testing.T) {
	logger := &mockLogger{}
	templateEngine := &mockTemplateEngine{}
	parser := NewEnhancedParser(templateEngine, logger)

	play := &types.Play{
		Name:  "",
		Hosts: "all",
		Tasks: []types.Task{
			{Name: "Test Task", Module: "debug"},
		},
	}

	err := parser.validatePlay(play, 0)

	assert.NoError(t, err)
	assert.Equal(t, "Play 1", play.Name) // Should auto-generate name
}

func TestEnhancedParser_ValidatePlay_NoHosts(t *testing.T) {
	logger := &mockLogger{}
	templateEngine := &mockTemplateEngine{}
	parser := NewEnhancedParser(templateEngine, logger)

	play := &types.Play{
		Name:  "Test Play",
		Hosts: "",
		Tasks: []types.Task{
			{Name: "Test Task", Module: "debug"},
		},
	}

	err := parser.validatePlay(play, 0)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must specify hosts")
}

func TestEnhancedParser_ValidatePlay_NoTasks(t *testing.T) {
	logger := &mockLogger{}
	templateEngine := &mockTemplateEngine{}
	parser := NewEnhancedParser(templateEngine, logger)

	play := &types.Play{
		Name:  "Test Play",
		Hosts: "all",
		Tasks: []types.Task{},
	}

	err := parser.validatePlay(play, 0)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must contain at least one task")
}

func TestEnhancedParser_ValidatePlay_WithPreTasks(t *testing.T) {
	logger := &mockLogger{}
	templateEngine := &mockTemplateEngine{}
	parser := NewEnhancedParser(templateEngine, logger)

	play := &types.Play{
		Name:  "Test Play",
		Hosts: "all",
		PreTasks: []types.Task{
			{Name: "Pre Task", Module: "debug"},
		},
	}

	err := parser.validatePlay(play, 0)

	assert.NoError(t, err) // Should pass with pre_tasks
}

func TestEnhancedParser_ValidatePlay_WithPostTasks(t *testing.T) {
	logger := &mockLogger{}
	templateEngine := &mockTemplateEngine{}
	parser := NewEnhancedParser(templateEngine, logger)

	play := &types.Play{
		Name:  "Test Play",
		Hosts: "all",
		PostTasks: []types.Task{
			{Name: "Post Task", Module: "debug"},
		},
	}

	err := parser.validatePlay(play, 0)

	assert.NoError(t, err) // Should pass with post_tasks
}

func TestEnhancedParser_ValidateTask_NoModule(t *testing.T) {
	logger := &mockLogger{}
	templateEngine := &mockTemplateEngine{}
	parser := NewEnhancedParser(templateEngine, logger)

	task := &types.Task{
		Name:   "Test Task",
		Module: "",
	}

	err := parser.validateTask(task, "test")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must specify a module")
}

func TestEnhancedParser_ValidateTask_NoName(t *testing.T) {
	logger := &mockLogger{}
	templateEngine := &mockTemplateEngine{}
	parser := NewEnhancedParser(templateEngine, logger)

	task := &types.Task{
		Name:   "",
		Module: "debug",
	}

	err := parser.validateTask(task, "test")

	assert.NoError(t, err)
	assert.Equal(t, "debug task", task.Name) // Should auto-generate name
}

func TestEnhancedParser_ValidateTask_WithLoop(t *testing.T) {
	logger := &mockLogger{}
	templateEngine := &mockTemplateEngine{}
	parser := NewEnhancedParser(templateEngine, logger)

	task := &types.Task{
		Name:   "Test Task",
		Module: "debug",
		Loop: &types.Loop{
			Items: []interface{}{"item1", "item2"},
		},
	}

	err := parser.validateTask(task, "test")

	assert.NoError(t, err)
}

func TestEnhancedParser_ValidateTask_WithCondition(t *testing.T) {
	logger := &mockLogger{}
	templateEngine := &mockTemplateEngine{}
	parser := NewEnhancedParser(templateEngine, logger)

	task := &types.Task{
		Name:   "Test Task",
		Module: "debug",
		When:   "ansible_os_family == 'Debian'",
	}

	err := parser.validateTask(task, "test")

	assert.NoError(t, err)
}

func TestEnhancedParser_ValidateLoop_NoItemsOrRange(t *testing.T) {
	logger := &mockLogger{}
	templateEngine := &mockTemplateEngine{}
	parser := NewEnhancedParser(templateEngine, logger)

	loop := &types.Loop{}

	err := parser.validateLoop(loop, "test")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must specify either 'items' or 'range'")
}

func TestEnhancedParser_ValidateLoop_BothItemsAndRange(t *testing.T) {
	logger := &mockLogger{}
	templateEngine := &mockTemplateEngine{}
	parser := NewEnhancedParser(templateEngine, logger)

	loop := &types.Loop{
		Items: []interface{}{"item1"},
		Range: "1..10",
	}

	err := parser.validateLoop(loop, "test")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot specify both 'items' and 'range'")
}

func TestEnhancedParser_ValidateLoop_ValidItems(t *testing.T) {
	logger := &mockLogger{}
	templateEngine := &mockTemplateEngine{}
	parser := NewEnhancedParser(templateEngine, logger)

	loop := &types.Loop{
		Items: []interface{}{"item1", "item2"},
	}

	err := parser.validateLoop(loop, "test")

	assert.NoError(t, err)
}

func TestEnhancedParser_ValidateLoop_ValidRange(t *testing.T) {
	logger := &mockLogger{}
	templateEngine := &mockTemplateEngine{}
	parser := NewEnhancedParser(templateEngine, logger)

	loop := &types.Loop{
		Range: "1..10",
	}

	err := parser.validateLoop(loop, "test")

	assert.NoError(t, err)
}

func TestEnhancedParser_ValidateCondition_UnclosedTemplate(t *testing.T) {
	logger := &mockLogger{}
	templateEngine := &mockTemplateEngine{}
	parser := NewEnhancedParser(templateEngine, logger)

	condition := "{{ variable"

	err := parser.validateCondition(condition, "test")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unclosed template expression")
}

func TestEnhancedParser_ValidateCondition_Valid(t *testing.T) {
	logger := &mockLogger{}
	templateEngine := &mockTemplateEngine{}
	parser := NewEnhancedParser(templateEngine, logger)

	condition := "{{ variable }} == 'value'"

	err := parser.validateCondition(condition, "test")

	assert.NoError(t, err)
}

func TestEnhancedParser_SetVariables(t *testing.T) {
	logger := &mockLogger{}
	templateEngine := &mockTemplateEngine{}
	parser := NewEnhancedParser(templateEngine, logger)

	vars := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
	}

	parser.SetVariables(vars)

	assert.Equal(t, vars, parser.variables)
}

func TestEnhancedParser_AddVariable(t *testing.T) {
	logger := &mockLogger{}
	templateEngine := &mockTemplateEngine{}
	parser := NewEnhancedParser(templateEngine, logger)

	parser.AddVariable("key1", "value1")
	parser.AddVariable("key2", 123)

	assert.Equal(t, "value1", parser.variables["key1"])
	assert.Equal(t, 123, parser.variables["key2"])
}

func TestEnhancedParser_AddVariable_NilVariables(t *testing.T) {
	logger := &mockLogger{}
	templateEngine := &mockTemplateEngine{}
	parser := &EnhancedParser{
		templateEngine: templateEngine,
		logger:         logger,
		variables:      nil,
	}

	parser.AddVariable("key1", "value1")

	assert.NotNil(t, parser.variables)
	assert.Equal(t, "value1", parser.variables["key1"])
}

func TestEnhancedParser_GetSupportedFormats(t *testing.T) {
	logger := &mockLogger{}
	templateEngine := &mockTemplateEngine{}
	parser := NewEnhancedParser(templateEngine, logger)

	formats := parser.GetSupportedFormats()

	assert.Contains(t, formats, "yaml")
	assert.Contains(t, formats, "yml")
}

func TestEnhancedParser_ValidateFile_ValidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.yml")

	content := `name: Test
value: 123
`

	err := os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err)

	logger := &mockLogger{}
	templateEngine := &mockTemplateEngine{}
	parser := NewEnhancedParser(templateEngine, logger)

	err = parser.ValidateFile(filePath)

	assert.NoError(t, err)
}

func TestEnhancedParser_ValidateFile_UnsupportedExtension(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")

	err := os.WriteFile(filePath, []byte("test"), 0644)
	require.NoError(t, err)

	logger := &mockLogger{}
	templateEngine := &mockTemplateEngine{}
	parser := NewEnhancedParser(templateEngine, logger)

	err = parser.ValidateFile(filePath)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported file format")
}

func TestEnhancedParser_ValidateFile_FileNotFound(t *testing.T) {
	logger := &mockLogger{}
	templateEngine := &mockTemplateEngine{}
	parser := NewEnhancedParser(templateEngine, logger)

	err := parser.ValidateFile("/nonexistent/file.yml")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot read file")
}

func TestEnhancedParser_ValidateFile_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "invalid.yml")

	invalidContent := `name: Test
invalid: [[[
`

	err := os.WriteFile(filePath, []byte(invalidContent), 0644)
	require.NoError(t, err)

	logger := &mockLogger{}
	templateEngine := &mockTemplateEngine{}
	parser := NewEnhancedParser(templateEngine, logger)

	err = parser.ValidateFile(filePath)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid YAML syntax")
}

func TestEnhancedParser_ValidateInventory_NoGroups(t *testing.T) {
	logger := &mockLogger{}
	templateEngine := &mockTemplateEngine{}
	parser := NewEnhancedParser(templateEngine, logger)

	inventory := &types.Inventory{
		Groups: make(map[string]*types.Group),
	}

	err := parser.validateInventory(inventory)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "inventory must contain at least one group")
}

func TestEnhancedParser_ValidateInventory_Valid(t *testing.T) {
	logger := &mockLogger{}
	templateEngine := &mockTemplateEngine{}
	parser := NewEnhancedParser(templateEngine, logger)

	inventory := &types.Inventory{
		Groups: map[string]*types.Group{
			"all": {
				Name: "all",
				Hosts: map[string]*types.Host{
					"host1": {Name: "host1", Address: "192.168.1.1"},
				},
			},
		},
	}

	err := parser.validateInventory(inventory)

	assert.NoError(t, err)
}

func TestEnhancedParser_ValidateGroup_NoHostsOrChildren(t *testing.T) {
	logger := &mockLogger{}
	templateEngine := &mockTemplateEngine{}
	parser := NewEnhancedParser(templateEngine, logger)

	group := &types.Group{
		Name:     "test",
		Hosts:    make(map[string]*types.Host),
		Children: []string{},
	}

	err := parser.validateGroup(group, "test")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must contain either hosts or children")
}

func TestEnhancedParser_ValidateGroup_ValidWithHosts(t *testing.T) {
	logger := &mockLogger{}
	templateEngine := &mockTemplateEngine{}
	parser := NewEnhancedParser(templateEngine, logger)

	group := &types.Group{
		Name: "test",
		Hosts: map[string]*types.Host{
			"host1": {Name: "host1", Address: "192.168.1.1"},
		},
	}

	err := parser.validateGroup(group, "test")

	assert.NoError(t, err)
}

func TestEnhancedParser_ValidateGroup_ValidWithChildren(t *testing.T) {
	logger := &mockLogger{}
	templateEngine := &mockTemplateEngine{}
	parser := NewEnhancedParser(templateEngine, logger)

	group := &types.Group{
		Name:     "test",
		Hosts:    make(map[string]*types.Host),
		Children: []string{"child1", "child2"},
	}

	err := parser.validateGroup(group, "test")

	assert.NoError(t, err)
}

func TestEnhancedParser_ValidateHost_NoAddress(t *testing.T) {
	logger := &mockLogger{}
	templateEngine := &mockTemplateEngine{}
	parser := NewEnhancedParser(templateEngine, logger)

	host := &types.Host{
		Name:    "host1",
		Address: "",
	}

	err := parser.validateHost(host, "host1")

	assert.NoError(t, err)
	assert.Equal(t, "host1", host.Address) // Should use name as address
}

func TestEnhancedParser_ValidateHost_NoPort(t *testing.T) {
	logger := &mockLogger{}
	templateEngine := &mockTemplateEngine{}
	parser := NewEnhancedParser(templateEngine, logger)

	host := &types.Host{
		Name:    "host1",
		Address: "192.168.1.1",
		Port:    0,
	}

	err := parser.validateHost(host, "host1")

	assert.NoError(t, err)
	assert.Equal(t, 22, host.Port) // Should default to 22
}

func TestEnhancedParser_ValidateHost_Valid(t *testing.T) {
	logger := &mockLogger{}
	templateEngine := &mockTemplateEngine{}
	parser := NewEnhancedParser(templateEngine, logger)

	host := &types.Host{
		Name:    "host1",
		Address: "192.168.1.1",
		Port:    2222,
	}

	err := parser.validateHost(host, "host1")

	assert.NoError(t, err)
	assert.Equal(t, "192.168.1.1", host.Address)
	assert.Equal(t, 2222, host.Port)
}

func TestEnhancedParser_CountHosts(t *testing.T) {
	logger := &mockLogger{}
	templateEngine := &mockTemplateEngine{}
	parser := NewEnhancedParser(templateEngine, logger)

	inventory := &types.Inventory{
		Groups: map[string]*types.Group{
			"group1": {
				Hosts: map[string]*types.Host{
					"host1": {},
					"host2": {},
				},
			},
			"group2": {
				Hosts: map[string]*types.Host{
					"host3": {},
				},
			},
		},
	}

	count := parser.countHosts(inventory)

	assert.Equal(t, 3, count)
}

func TestEnhancedParser_ParsePlaybook_WithMultiplePlays(t *testing.T) {
	tmpDir := t.TempDir()
	playbookPath := filepath.Join(tmpDir, "multi_play.yml")

	playbookContent := `name: Multi Play Playbook
plays:
  - name: First Play
    hosts: webservers
    tasks:
      - name: Task 1
        module: debug
  - name: Second Play
    hosts: databases
    tasks:
      - name: Task 2
        module: debug
`

	err := os.WriteFile(playbookPath, []byte(playbookContent), 0644)
	require.NoError(t, err)

	logger := &mockLogger{}
	templateEngine := &mockTemplateEngine{}
	parser := NewEnhancedParser(templateEngine, logger)
	ctx := context.Background()

	playbook, err := parser.ParsePlaybook(ctx, playbookPath)

	assert.NoError(t, err)
	assert.NotNil(t, playbook)
	assert.Len(t, playbook.Plays, 2)
	assert.Equal(t, "First Play", playbook.Plays[0].Name)
	assert.Equal(t, "Second Play", playbook.Plays[1].Name)
}
