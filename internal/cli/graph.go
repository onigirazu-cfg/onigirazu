package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newGraphCmd() *cobra.Command {
	var (
		outputFormat string
		showVars     bool
		showHandlers bool
		compact      bool
	)

	cmd := &cobra.Command{
		Use:   "graph [playbook]",
		Short: "Generate a visual graph of playbook structure",
		Long: `Generate a visual representation of playbook structure showing:
  - Plays and their relationships
  - Tasks and their execution order
  - Handlers and notifications
  - Variable dependencies
  - Conditional execution paths

Output formats:
  - ascii: Simple ASCII art (default)
  - dot: GraphViz DOT format
  - mermaid: Mermaid diagram format`,
		Example: `  # Generate ASCII graph
  onigirazu graph playbook.yml

  # Generate DOT format for GraphViz
  onigirazu graph --format=dot playbook.yml > graph.dot
  dot -Tpng graph.dot -o graph.png

  # Generate Mermaid diagram
  onigirazu graph --format=mermaid playbook.yml

  # Show variables and handlers
  onigirazu graph --show-vars --show-handlers playbook.yml

  # Compact view (less details)
  onigirazu graph --compact playbook.yml`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filename := args[0]

			// Read and parse playbook
			data, err := os.ReadFile(filename)
			if err != nil {
				return fmt.Errorf("cannot read file: %w", err)
			}

			var playbook types.Playbook
			if err := yaml.Unmarshal(data, &playbook); err != nil {
				return fmt.Errorf("cannot parse playbook: %w", err)
			}

			// Generate graph based on format
			switch outputFormat {
			case "ascii":
				generateASCIIGraph(&playbook, showVars, showHandlers, compact)
			case "dot":
				generateDOTGraph(&playbook, showVars, showHandlers)
			case "mermaid":
				generateMermaidGraph(&playbook, showVars, showHandlers)
			default:
				return fmt.Errorf("unknown format: %s (supported: ascii, dot, mermaid)", outputFormat)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "format", "f", "ascii", "Output format (ascii, dot, mermaid)")
	cmd.Flags().BoolVar(&showVars, "show-vars", false, "Show variable definitions and usage")
	cmd.Flags().BoolVar(&showHandlers, "show-handlers", false, "Show handlers and notifications")
	cmd.Flags().BoolVar(&compact, "compact", false, "Compact view with less details")

	return cmd
}

// generateASCIIGraph generates a simple ASCII art representation
func generateASCIIGraph(playbook *types.Playbook, showVars, showHandlers, compact bool) {
	fmt.Printf("Playbook: %s\n", getPlaybookName(playbook))
	fmt.Println(strings.Repeat("=", 80))

	if showVars && len(playbook.Vars) > 0 {
		fmt.Println("\n📦 Global Variables:")
		for key := range playbook.Vars {
			fmt.Printf("  • %s\n", key)
		}
	}

	for i, play := range playbook.Plays {
		fmt.Printf("\n┌─ Play %d: %s\n", i+1, getPlayName(&play))
		fmt.Printf("│  Hosts: %s\n", play.Hosts)

		if showVars && len(play.Vars) > 0 {
			fmt.Println("│  Variables:")
			for key := range play.Vars {
				fmt.Printf("│    • %s\n", key)
			}
		}

		// Pre-tasks
		if len(play.PreTasks) > 0 {
			fmt.Println("│")
			fmt.Println("│  ┌─ Pre-Tasks")
			for j, task := range play.PreTasks {
				printTask(&task, j+1, "│  │", compact)
			}
		}

		// Main tasks
		if len(play.Tasks) > 0 {
			fmt.Println("│")
			fmt.Println("│  ┌─ Tasks")
			for j, task := range play.Tasks {
				printTask(&task, j+1, "│  │", compact)
			}
		}

		// Post-tasks
		if len(play.PostTasks) > 0 {
			fmt.Println("│")
			fmt.Println("│  ┌─ Post-Tasks")
			for j, task := range play.PostTasks {
				printTask(&task, j+1, "│  │", compact)
			}
		}

		// Handlers
		if showHandlers && len(play.Handlers) > 0 {
			fmt.Println("│")
			fmt.Println("│  ┌─ Handlers")
			for j, handler := range play.Handlers {
				printTask(&handler, j+1, "│  │", compact)
			}
		}

		fmt.Println("└─")
	}
}

func printTask(task *types.Task, index int, prefix string, compact bool) {
	taskName := task.Name
	if taskName == "" {
		taskName = fmt.Sprintf("(unnamed %s)", task.Module)
	}

	fmt.Printf("%s  %d. %s\n", prefix, index, taskName)

	if !compact {
		fmt.Printf("%s     Module: %s\n", prefix, task.Module)

		if task.When != "" {
			fmt.Printf("%s     When: %s\n", prefix, task.When)
		}

		if task.Loop != nil {
			fmt.Printf("%s     Loop: %v\n", prefix, task.Loop.Items)
		}

		if task.Register != "" {
			fmt.Printf("%s     Register: %s\n", prefix, task.Register)
		}

		if len(task.Notify) > 0 {
			fmt.Printf("%s     Notify: %s\n", prefix, strings.Join(task.Notify, ", "))
		}

		if task.Become {
			becomeUser := task.BecomeUser
			if becomeUser == "" {
				becomeUser = "root"
			}
			fmt.Printf("%s     Become: %s\n", prefix, becomeUser)
		}
	}
}

// generateDOTGraph generates GraphViz DOT format
func generateDOTGraph(playbook *types.Playbook, showVars, showHandlers bool) {
	fmt.Println("digraph Playbook {")
	fmt.Println("  rankdir=TB;")
	fmt.Println("  node [shape=box, style=rounded];")
	fmt.Println()

	// Playbook node
	playbookName := escapeLabel(getPlaybookName(playbook))
	fmt.Printf("  playbook [label=\"%s\", shape=folder, style=filled, fillcolor=lightblue];\n", playbookName)
	fmt.Println()

	// Global variables
	if showVars && len(playbook.Vars) > 0 {
		fmt.Println("  subgraph cluster_global_vars {")
		fmt.Println("    label=\"Global Variables\";")
		fmt.Println("    style=dashed;")
		for key := range playbook.Vars {
			varID := fmt.Sprintf("gvar_%s", sanitizeID(key))
			fmt.Printf("    %s [label=\"%s\", shape=ellipse, style=filled, fillcolor=lightyellow];\n", varID, escapeLabel(key))
		}
		fmt.Println("  }")
		fmt.Println()
	}

	// Process each play
	for i, play := range playbook.Plays {
		playID := fmt.Sprintf("play_%d", i)
		playName := escapeLabel(getPlayName(&play))

		fmt.Printf("  subgraph cluster_%s {\n", playID)
		fmt.Printf("    label=\"Play: %s\\nHosts: %s\";\n", playName, escapeLabel(play.Hosts))
		fmt.Println("    style=filled;")
		fmt.Println("    fillcolor=lightgray;")
		fmt.Println()

		// Play variables
		if showVars && len(play.Vars) > 0 {
			for key := range play.Vars {
				varID := fmt.Sprintf("%s_var_%s", playID, sanitizeID(key))
				fmt.Printf("    %s [label=\"%s\", shape=ellipse, style=filled, fillcolor=lightyellow];\n", varID, escapeLabel(key))
			}
			fmt.Println()
		}

		// Tasks
		taskOffset := 0
		if len(play.PreTasks) > 0 {
			fmt.Println("    // Pre-tasks")
			for j, task := range play.PreTasks {
				taskID := fmt.Sprintf("%s_pretask_%d", playID, j)
				printDOTTask(taskID, &task, playID, j, taskOffset)
			}
			taskOffset += len(play.PreTasks)
			fmt.Println()
		}

		if len(play.Tasks) > 0 {
			fmt.Println("    // Tasks")
			for j, task := range play.Tasks {
				taskID := fmt.Sprintf("%s_task_%d", playID, j)
				printDOTTask(taskID, &task, playID, j, taskOffset)
			}
			taskOffset += len(play.Tasks)
			fmt.Println()
		}

		if len(play.PostTasks) > 0 {
			fmt.Println("    // Post-tasks")
			for j, task := range play.PostTasks {
				taskID := fmt.Sprintf("%s_posttask_%d", playID, j)
				printDOTTask(taskID, &task, playID, j, taskOffset)
			}
			fmt.Println()
		}

		// Handlers
		if showHandlers && len(play.Handlers) > 0 {
			fmt.Println("    // Handlers")
			for j, handler := range play.Handlers {
				handlerID := fmt.Sprintf("%s_handler_%d", playID, j)
				handlerName := handler.Name
				if handlerName == "" {
					handlerName = fmt.Sprintf("(unnamed %s)", handler.Module)
				}
				fmt.Printf("    %s [label=\"%s\", shape=component, style=filled, fillcolor=lightgreen];\n", handlerID, escapeLabel(handlerName))
			}
			fmt.Println()
		}

		fmt.Println("  }")
		fmt.Println()

		// Connect playbook to play
		fmt.Printf("  playbook -> %s_task_0 [style=bold];\n", playID)
	}

	// Create edges for task flow and notifications
	for i, play := range playbook.Plays {
		playID := fmt.Sprintf("play_%d", i)

		// Connect pre-tasks
		connectTasks(playID, "pretask", play.PreTasks)

		// Connect to main tasks
		if len(play.PreTasks) > 0 && len(play.Tasks) > 0 {
			lastPreTask := len(play.PreTasks) - 1
			fmt.Printf("  %s_pretask_%d -> %s_task_0;\n", playID, lastPreTask, playID)
		}

		// Connect main tasks
		connectTasks(playID, "task", play.Tasks)

		// Connect to post-tasks
		if len(play.Tasks) > 0 && len(play.PostTasks) > 0 {
			lastTask := len(play.Tasks) - 1
			fmt.Printf("  %s_task_%d -> %s_posttask_0;\n", playID, lastTask, playID)
		}

		// Connect post-tasks
		connectTasks(playID, "posttask", play.PostTasks)

		// Connect notifications to handlers
		if showHandlers {
			allTasks := append([]types.Task{}, play.PreTasks...)
			allTasks = append(allTasks, play.Tasks...)
			allTasks = append(allTasks, play.PostTasks...)

			for j, task := range allTasks {
				if len(task.Notify) > 0 {
					taskType := "task"
					taskIdx := j
					if j < len(play.PreTasks) {
						taskType = "pretask"
					} else if j >= len(play.PreTasks)+len(play.Tasks) {
						taskType = "posttask"
						taskIdx = j - len(play.PreTasks) - len(play.Tasks)
					} else {
						taskIdx = j - len(play.PreTasks)
					}

					taskID := fmt.Sprintf("%s_%s_%d", playID, taskType, taskIdx)

					for _, notifyName := range task.Notify {
						// Find handler by name
						for k, handler := range play.Handlers {
							if handler.Name == notifyName {
								handlerID := fmt.Sprintf("%s_handler_%d", playID, k)
								fmt.Printf("  %s -> %s [style=dashed, color=orange, label=\"notify\"];\n", taskID, handlerID)
							}
						}
					}
				}
			}
		}
	}

	fmt.Println("}")
}

func printDOTTask(taskID string, task *types.Task, playID string, index, offset int) {
	taskName := task.Name
	if taskName == "" {
		taskName = fmt.Sprintf("(unnamed %s)", task.Module)
	}

	label := escapeLabel(taskName)
	color := "white"

	if task.When != "" {
		color = "lightyellow"
		label += "\\n[conditional]"
	}

	if task.Loop != nil {
		label += "\\n[loop]"
	}

	fmt.Printf("    %s [label=\"%s\", style=filled, fillcolor=%s];\n", taskID, label, color)
}

func connectTasks(playID, taskType string, tasks []types.Task) {
	for i := 0; i < len(tasks)-1; i++ {
		fmt.Printf("  %s_%s_%d -> %s_%s_%d;\n", playID, taskType, i, playID, taskType, i+1)
	}
}

// generateMermaidGraph generates Mermaid diagram format
func generateMermaidGraph(playbook *types.Playbook, showVars, showHandlers bool) {
	fmt.Println("graph TD")
	fmt.Printf("  PB[%s]\n", escapeMermaid(getPlaybookName(playbook)))
	fmt.Println()

	// Global variables
	if showVars && len(playbook.Vars) > 0 {
		for key := range playbook.Vars {
			varID := fmt.Sprintf("GVAR_%s", sanitizeID(key))
			fmt.Printf("  %s{{%s}}\n", varID, escapeMermaid(key))
			fmt.Printf("  PB -.-> %s\n", varID)
		}
		fmt.Println()
	}

	// Process each play
	for i, play := range playbook.Plays {
		playID := fmt.Sprintf("P%d", i)
		playName := escapeMermaid(getPlayName(&play))

		fmt.Printf("  %s[Play: %s<br/>Hosts: %s]\n", playID, playName, escapeMermaid(play.Hosts))
		fmt.Printf("  PB --> %s\n", playID)
		fmt.Println()

		// Play variables
		if showVars && len(play.Vars) > 0 {
			for key := range play.Vars {
				varID := fmt.Sprintf("%s_VAR_%s", playID, sanitizeID(key))
				fmt.Printf("  %s{{%s}}\n", varID, escapeMermaid(key))
				fmt.Printf("  %s -.-> %s\n", playID, varID)
			}
			fmt.Println()
		}

		// Tasks
		taskCounter := 0
		if len(play.PreTasks) > 0 {
			fmt.Println("  %% Pre-tasks")
			for j, task := range play.PreTasks {
				taskID := fmt.Sprintf("%s_PT%d", playID, j)
				printMermaidTask(taskID, &task)
				if j == 0 {
					fmt.Printf("  %s --> %s\n", playID, taskID)
				} else {
					prevID := fmt.Sprintf("%s_PT%d", playID, j-1)
					fmt.Printf("  %s --> %s\n", prevID, taskID)
				}
				taskCounter++
			}
			fmt.Println()
		}

		if len(play.Tasks) > 0 {
			fmt.Println("  %% Tasks")
			for j, task := range play.Tasks {
				taskID := fmt.Sprintf("%s_T%d", playID, j)
				printMermaidTask(taskID, &task)
				if j == 0 {
					if len(play.PreTasks) > 0 {
						prevID := fmt.Sprintf("%s_PT%d", playID, len(play.PreTasks)-1)
						fmt.Printf("  %s --> %s\n", prevID, taskID)
					} else {
						fmt.Printf("  %s --> %s\n", playID, taskID)
					}
				} else {
					prevID := fmt.Sprintf("%s_T%d", playID, j-1)
					fmt.Printf("  %s --> %s\n", prevID, taskID)
				}
				taskCounter++
			}
			fmt.Println()
		}

		if len(play.PostTasks) > 0 {
			fmt.Println("  %% Post-tasks")
			for j, task := range play.PostTasks {
				taskID := fmt.Sprintf("%s_PST%d", playID, j)
				printMermaidTask(taskID, &task)
				if j == 0 {
					if len(play.Tasks) > 0 {
						prevID := fmt.Sprintf("%s_T%d", playID, len(play.Tasks)-1)
						fmt.Printf("  %s --> %s\n", prevID, taskID)
					} else if len(play.PreTasks) > 0 {
						prevID := fmt.Sprintf("%s_PT%d", playID, len(play.PreTasks)-1)
						fmt.Printf("  %s --> %s\n", prevID, taskID)
					} else {
						fmt.Printf("  %s --> %s\n", playID, taskID)
					}
				} else {
					prevID := fmt.Sprintf("%s_PST%d", playID, j-1)
					fmt.Printf("  %s --> %s\n", prevID, taskID)
				}
			}
			fmt.Println()
		}

		// Handlers
		if showHandlers && len(play.Handlers) > 0 {
			fmt.Println("  %% Handlers")
			for j, handler := range play.Handlers {
				handlerID := fmt.Sprintf("%s_H%d", playID, j)
				handlerName := handler.Name
				if handlerName == "" {
					handlerName = fmt.Sprintf("(unnamed %s)", handler.Module)
				}
				fmt.Printf("  %s([%s])\n", handlerID, escapeMermaid(handlerName))
			}
			fmt.Println()

			// Connect notifications
			allTasks := append([]types.Task{}, play.PreTasks...)
			allTasks = append(allTasks, play.Tasks...)
			allTasks = append(allTasks, play.PostTasks...)

			for j, task := range allTasks {
				if len(task.Notify) > 0 {
					taskType := "T"
					taskIdx := j
					if j < len(play.PreTasks) {
						taskType = "PT"
					} else if j >= len(play.PreTasks)+len(play.Tasks) {
						taskType = "PST"
						taskIdx = j - len(play.PreTasks) - len(play.Tasks)
					} else {
						taskIdx = j - len(play.PreTasks)
					}

					taskID := fmt.Sprintf("%s_%s%d", playID, taskType, taskIdx)

					for _, notifyName := range task.Notify {
						for k, handler := range play.Handlers {
							if handler.Name == notifyName {
								handlerID := fmt.Sprintf("%s_H%d", playID, k)
								fmt.Printf("  %s -.notify.-> %s\n", taskID, handlerID)
							}
						}
					}
				}
			}
		}
	}

	// Add styling
	fmt.Println()
	fmt.Println("  classDef playClass fill:#e1f5ff,stroke:#01579b")
	fmt.Println("  classDef taskClass fill:#fff9c4,stroke:#f57f17")
	fmt.Println("  classDef handlerClass fill:#c8e6c9,stroke:#2e7d32")
	fmt.Println("  classDef varClass fill:#ffe0b2,stroke:#e65100")
}

func printMermaidTask(taskID string, task *types.Task) {
	taskName := task.Name
	if taskName == "" {
		taskName = fmt.Sprintf("(unnamed %s)", task.Module)
	}

	shape := "["
	endShape := "]"

	if task.When != "" {
		shape = "{"
		endShape = "}"
	}

	fmt.Printf("  %s%s%s%s\n", taskID, shape, escapeMermaid(taskName), endShape)
}

// Helper functions

func getPlaybookName(playbook *types.Playbook) string {
	if playbook.Name != "" {
		return playbook.Name
	}
	return "Unnamed Playbook"
}

func getPlayName(play *types.Play) string {
	if play.Name != "" {
		return play.Name
	}
	return "Unnamed Play"
}

func escapeLabel(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}

func escapeMermaid(s string) string {
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func sanitizeID(s string) string {
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, ".", "_")
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "/", "_")
	return s
}
