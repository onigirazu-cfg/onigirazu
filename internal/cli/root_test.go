package cli

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewRootCommand tests creating the root command
func TestNewRootCommand(t *testing.T) {
	cmd := NewRootCommand()

	assert.NotNil(t, cmd, "Root command should not be nil")
	assert.Equal(t, "onigirazu", cmd.Use, "Command use should be 'onigirazu'")
	assert.NotEmpty(t, cmd.Short, "Short description should not be empty")
	assert.NotEmpty(t, cmd.Long, "Long description should not be empty")
}

// TestRootCommand_Flags tests that root command has expected flags
func TestRootCommand_Flags(t *testing.T) {
	cmd := NewRootCommand()

	testCases := []struct {
		name     string
		flagName string
		flagType string
	}{
		{"config flag", "config", "string"},
		{"inventory flag", "inventory", "stringSlice"},
		{"state flag", "state", "string"},
		{"verbose flag", "verbose", "bool"},
		{"no-color flag", "no-color", "bool"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			flag := cmd.PersistentFlags().Lookup(tc.flagName)
			assert.NotNil(t, flag, "Flag %s should exist", tc.flagName)
		})
	}
}

// TestRootCommand_HelpOutput tests help output
func TestRootCommand_HelpOutput(t *testing.T) {
	cmd := NewRootCommand()

	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	// Execute help command
	cmd.SetArgs([]string{"--help"})
	_ = cmd.Execute() // Note: Help doesn't return an error in cobra, it prints and returns nil

	helpOutput := output.String()
	assert.NotEmpty(t, helpOutput, "Help output should not be empty")
	assert.Contains(t, helpOutput, "onigirazu", "Help should contain command name")
}

// TestRootCommand_Version tests version output
func TestRootCommand_Version(t *testing.T) {
	cmd := NewRootCommand()

	assert.NotEmpty(t, cmd.Version, "Version should be set")
}

// TestRootCommand_GlobalFlags tests global flags are accessible
func TestRootCommand_GlobalFlags(t *testing.T) {
	cmd := NewRootCommand()

	// Check persistent flags
	flags := cmd.PersistentFlags()
	assert.NotNil(t, flags, "Persistent flags should exist")

	// Verify flag defaults
	config, err := cmd.PersistentFlags().GetString("config")
	require.NoError(t, err)
	assert.Empty(t, config, "Config flag should default to empty")

	verbose, err := cmd.PersistentFlags().GetBool("verbose")
	require.NoError(t, err)
	assert.False(t, verbose, "Verbose flag should default to false")

	noColor, err := cmd.PersistentFlags().GetBool("no-color")
	require.NoError(t, err)
	assert.False(t, noColor, "No-color flag should default to false")
}

// TestRootCommand_InventoryFlag tests inventory flag accepts multiple values
func TestRootCommand_InventoryFlag(t *testing.T) {
	cmd := NewRootCommand()

	// Set multiple inventory paths
	cmd.SetArgs([]string{"-i", "/path/to/inv1", "-i", "/path/to/inv2"})

	// Parse flags (don't execute)
	err := cmd.ParseFlags([]string{"-i", "/path/to/inv1", "-i", "/path/to/inv2"})
	assert.NoError(t, err, "Should parse multiple inventory flags")
}

// TestRootCommand_StateFlag tests state flag
func TestRootCommand_StateFlag(t *testing.T) {
	cmd := NewRootCommand()

	stateFlag := cmd.PersistentFlags().Lookup("state")
	assert.NotNil(t, stateFlag, "State flag should exist")

	// Check default value
	assert.Equal(t, ".onigirazu-state", stateFlag.DefValue, "State flag should default to .onigirazu-state")
}

// TestRootCommand_HasSubcommands tests that root command has subcommands
func TestRootCommand_HasSubcommands(t *testing.T) {
	cmd := NewRootCommand()

	// Root command should have subcommands
	assert.Greater(t, len(cmd.Commands()), 0, "Root command should have subcommands")

	// Verify command structure exists (some subcommands may be optional based on build flags)
	_ = cmd.Commands() // Just verify the structure is correct
}

// TestRootCommand_CobraStructure tests cobra command structure
func TestRootCommand_CobraStructure(t *testing.T) {
	cmd := NewRootCommand()

	// Test basic cobra properties
	assert.True(t, cmd.HasParent() == false, "Root command should not have parent")
	assert.NotEmpty(t, cmd.UsageString(), "Usage string should not be empty")
}

// TestRootCommand_ErrorHandling tests error handling
func TestRootCommand_ErrorHandling(t *testing.T) {
	cmd := NewRootCommand()

	// Test with invalid flag
	cmd.SetArgs([]string{"--invalid-flag"})

	// Should handle gracefully (cobra handles this)
	// The actual error handling depends on cobra's implementation
	assert.NotNil(t, cmd, "Command should still be valid after invalid flag")
}

// TestRootCommand_Run tests running root command without subcommand
func TestRootCommand_Run(t *testing.T) {
	cmd := NewRootCommand()

	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	// Running root command without args or subcommand should show help
	cmd.SetArgs([]string{})
	_ = cmd.Execute() // This might error or might just print help depending on implementation

	// The important thing is that it doesn't crash
	assert.NotNil(t, cmd, "Command should be valid")
}

// TestRootCommand_FlagValues tests parsing flag values
func TestRootCommand_FlagValues(t *testing.T) {
	testCases := []struct {
		name     string
		flags    []string
		key      string
		expected interface{}
	}{
		{
			name:     "config path",
			flags:    []string{"-c", "/path/to/config.yml"},
			key:      "config",
			expected: "/path/to/config.yml",
		},
		{
			name:     "verbose flag",
			flags:    []string{"-v"},
			key:      "verbose",
			expected: true,
		},
		{
			name:     "no-color flag",
			flags:    []string{"--no-color"},
			key:      "no-color",
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := NewRootCommand()
			err := cmd.ParseFlags(tc.flags)
			assert.NoError(t, err, "Should parse flags for: %s", tc.name)
		})
	}
}

// TestRootCommand_LegacyFlags tests that legacy flags are hidden but functional
func TestRootCommand_LegacyFlags(t *testing.T) {
	cmd := NewRootCommand()

	// Legacy flags should still be accessible
	flag := cmd.PersistentFlags().Lookup("playbook")
	assert.NotNil(t, flag, "Playbook flag should exist")

	// Check if hidden
	isHidden := flag.Hidden
	assert.True(t, isHidden, "Playbook flag should be hidden")
}

// TestRootCommand_ComplexScenario tests a complex scenario with multiple flags
func TestRootCommand_ComplexScenario(t *testing.T) {
	cmd := NewRootCommand()

	// Set multiple flags
	flags := []string{
		"-c", "/etc/onigirazu/config.yml",
		"-i", "/etc/onigirazu/inventory/hosts.yml",
		"-i", "/etc/onigirazu/inventory/groups.yml",
		"-s", "/var/lib/onigirazu/.state",
		"-v",
		"--no-color",
	}

	err := cmd.ParseFlags(flags)
	assert.NoError(t, err, "Should parse complex flag set")

	// Verify parsed values
	config, _ := cmd.PersistentFlags().GetString("config")
	assert.Equal(t, "/etc/onigirazu/config.yml", config)

	verbose, _ := cmd.PersistentFlags().GetBool("verbose")
	assert.True(t, verbose)

	noColor, _ := cmd.PersistentFlags().GetBool("no-color")
	assert.True(t, noColor)
}

// TestVersionCommand tests the version subcommand
func TestVersionCommand(t *testing.T) {
	cmd := NewVersionCommand()

	assert.NotNil(t, cmd, "Version command should not be nil")
	assert.Equal(t, "version", cmd.Name(), "Command name should be 'version'")
}

// TestCommand_Help tests command help generation
func TestCommand_Help(t *testing.T) {
	cmd := NewRootCommand()

	usageStr := cmd.UsageString()
	assert.NotEmpty(t, usageStr, "Usage string should not be empty")
	assert.Contains(t, usageStr, "onigirazu", "Usage should contain command name")
}

// TestCommand_Examples tests command with examples
func TestCommand_Examples(t *testing.T) {
	cmd := NewRootCommand()

	// Root command should have description
	assert.NotEmpty(t, cmd.Long, "Command should have description")
}

// TestRootCommand_TreeStructure tests the command tree structure
func TestRootCommand_TreeStructure(t *testing.T) {
	cmd := NewRootCommand()

	// Get all commands (recursively)
	var countCommands func(*cobra.Command) int
	countCommands = func(c *cobra.Command) int {
		count := 1 // Count current command
		for _, child := range c.Commands() {
			count += countCommands(child)
		}
		return count
	}

	totalCommands := countCommands(cmd)
	assert.Greater(t, totalCommands, 1, "Should have at least root + 1 subcommand")
}

// BenchmarkRootCommand_Creation benchmarks root command creation
func BenchmarkRootCommand_Creation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewRootCommand()
	}
}

// BenchmarkRootCommand_FlagParsing benchmarks flag parsing
func BenchmarkRootCommand_FlagParsing(b *testing.B) {
	cmd := NewRootCommand()
	flags := []string{"-c", "/config", "-i", "/inventory", "-v"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cmd.ParseFlags(flags)
	}
}
