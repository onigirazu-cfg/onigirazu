package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// newFmtCmd creates the fmt command
func newFmtCmd() *cobra.Command {
	var (
		check     bool
		write     bool
		indent    int
		recursive bool
		diff      bool
	)

	cmd := &cobra.Command{
		Use:   "fmt [playbook-file...]",
		Short: "Format playbook YAML files",
		Long: `Format playbook YAML files with consistent indentation and style.

By default, fmt will format files in-place. Use --check to verify formatting
without modifying files, or --diff to see what would change.

Examples:
  # Format a single playbook
  onigirazu fmt playbook.yml

  # Check if files are formatted (exit code 1 if not)
  onigirazu fmt --check playbook.yml

  # Show formatting differences
  onigirazu fmt --diff playbook.yml

  # Format all YAML files in current directory
  onigirazu fmt *.yml

  # Format all YAML files recursively
  onigirazu fmt --recursive .

  # Use custom indentation (default: 2)
  onigirazu fmt --indent 4 playbook.yml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("no playbook files specified")
			}

			// Collect files to format
			var files []string
			for _, arg := range args {
				if recursive {
					// Walk directory recursively
					err := filepath.Walk(arg, func(path string, info os.FileInfo, err error) error {
						if err != nil {
							return err
						}
						if !info.IsDir() && isYAMLFile(path) {
							files = append(files, path)
						}
						return nil
					})
					if err != nil {
						return fmt.Errorf("failed to walk directory %s: %w", arg, err)
					}
				} else {
					// Check if it's a YAML file
					if isYAMLFile(arg) {
						files = append(files, arg)
					}
				}
			}

			if len(files) == 0 {
				return fmt.Errorf("no YAML files found")
			}

			// Format each file
			hasUnformatted := false
			for _, file := range files {
				formatted, err := formatFile(file, indent, check, diff, write)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error formatting %s: %v\n", file, err)
					continue
				}

				if !formatted {
					hasUnformatted = true
				}
			}

			// Exit with code 1 if any files are unformatted in check mode
			if check && hasUnformatted {
				os.Exit(1)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&check, "check", false, "Check if files are formatted (don't modify)")
	cmd.Flags().BoolVarP(&write, "write", "w", false, "Write result to source file (default behavior)")
	cmd.Flags().IntVar(&indent, "indent", 2, "Number of spaces for indentation")
	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Format files recursively")
	cmd.Flags().BoolVarP(&diff, "diff", "d", false, "Show formatting differences")

	return cmd
}

// isYAMLFile checks if a file has a YAML extension
func isYAMLFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yml" || ext == ".yaml"
}

// formatFile formats a single YAML file
func formatFile(path string, indent int, check, showDiff, write bool) (bool, error) {
	// Read original file
	// #nosec G304 - path is provided by user as CLI argument, this is expected behavior for a formatting tool
	originalData, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("failed to read file: %w", err)
	}

	// Parse YAML
	var data interface{}
	if err := yaml.Unmarshal(originalData, &data); err != nil {
		return false, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Format YAML with custom encoder
	var formattedData []byte
	encoder := yaml.NewEncoder(nil)
	encoder.SetIndent(indent)

	// Create a buffer to capture formatted output
	var buf strings.Builder
	encoder = yaml.NewEncoder(&buf)
	encoder.SetIndent(indent)

	if err := encoder.Encode(data); err != nil {
		return false, fmt.Errorf("failed to encode YAML: %w", err)
	}

	if err := encoder.Close(); err != nil {
		return false, fmt.Errorf("failed to close encoder: %w", err)
	}

	formattedData = []byte(buf.String())

	// Normalize line endings and trim trailing whitespace
	formattedData = normalizeYAML(formattedData)
	originalNormalized := normalizeYAML(originalData)

	// Check if file is already formatted
	isFormatted := string(originalNormalized) == string(formattedData)

	if check {
		if isFormatted {
			fmt.Printf("✓ %s is formatted\n", path)
		} else {
			fmt.Printf("✗ %s needs formatting\n", path)
		}
		return isFormatted, nil
	}

	if showDiff {
		if !isFormatted {
			fmt.Printf("--- %s (original)\n", path)
			fmt.Printf("+++ %s (formatted)\n", path)
			showDifference(string(originalNormalized), string(formattedData))
		} else {
			fmt.Printf("✓ %s is already formatted\n", path)
		}
		return isFormatted, nil
	}

	// Write formatted file (default behavior)
	if !isFormatted {
		// #nosec G306 - 0644 is appropriate for YAML config files that need to be readable by other tools
		if err := os.WriteFile(path, formattedData, 0644); err != nil {
			return false, fmt.Errorf("failed to write file: %w", err)
		}
		fmt.Printf("✓ Formatted %s\n", path)
	} else {
		fmt.Printf("✓ %s is already formatted\n", path)
	}

	return isFormatted, nil
}

// normalizeYAML normalizes YAML content for comparison
func normalizeYAML(data []byte) []byte {
	content := string(data)

	// Normalize line endings
	content = strings.ReplaceAll(content, "\r\n", "\n")

	// Remove trailing whitespace from each line
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	content = strings.Join(lines, "\n")

	// Ensure single trailing newline
	content = strings.TrimRight(content, "\n") + "\n"

	return []byte(content)
}

// showDifference shows a simple diff between original and formatted content
func showDifference(original, formatted string) {
	origLines := strings.Split(original, "\n")
	formLines := strings.Split(formatted, "\n")

	maxLines := len(origLines)
	if len(formLines) > maxLines {
		maxLines = len(formLines)
	}

	for i := 0; i < maxLines; i++ {
		var origLine, formLine string
		if i < len(origLines) {
			origLine = origLines[i]
		}
		if i < len(formLines) {
			formLine = formLines[i]
		}

		if origLine != formLine {
			if origLine != "" {
				fmt.Printf("- %s\n", origLine)
			}
			if formLine != "" {
				fmt.Printf("+ %s\n", formLine)
			}
		}
	}
}
