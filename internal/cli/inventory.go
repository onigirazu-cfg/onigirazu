package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/onigirazu-cfg/onigirazu/internal/cache"
	"github.com/onigirazu-cfg/onigirazu/internal/inventory"
	"github.com/onigirazu-cfg/onigirazu/internal/logger"
	"github.com/onigirazu-cfg/onigirazu/internal/parser"
	"github.com/onigirazu-cfg/onigirazu/internal/template"
)

var (
	inventoryFile   string
	inventoryHost   string
	inventoryGroup  string
	inventoryList   bool
	inventoryGraph  bool
)

var inventoryCmd = &cobra.Command{
	Use:   "inventory",
	Short: "Manage and query inventory",
	Long:  `Display information about hosts and groups in the inventory`,
	RunE:  runInventory,
}

func init() {
	inventoryCmd.Flags().StringVarP(&inventoryFile, "inventory", "i", "", "Path to inventory file")
	inventoryCmd.Flags().StringVar(&inventoryHost, "host", "", "Show groups for a specific host")
	inventoryCmd.Flags().StringVar(&inventoryGroup, "group", "", "Show hierarchy for a specific group")
	inventoryCmd.Flags().BoolVar(&inventoryList, "list", false, "List all hosts and groups")
	inventoryCmd.Flags().BoolVar(&inventoryGraph, "graph", false, "Show group hierarchy as a graph")
}

func runInventory(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	log, err := logger.NewEnhancedLogger("info", "text", os.Stdout)
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}

	// Initialize parser and cache
	invParser := parser.NewInventoryParser(log)
	cacheManager := cache.NewManager(0)

	// Find inventory file if not specified
	if inventoryFile == "" {
		var err error
		inventoryFile, err = invParser.FindInventoryFile(".")
		if err != nil {
			return fmt.Errorf("failed to find inventory file: %w", err)
		}
	}

	// Create inventory manager
	templateEngine := template.NewEngine()
	playbookParser := parser.NewEnhancedParser(templateEngine, log)
	invManager := inventory.NewManager(playbookParser, log, cacheManager)

	// Load inventory
	if err := invManager.LoadInventory(ctx, inventoryFile); err != nil {
		return fmt.Errorf("failed to load inventory: %w", err)
	}

	// Handle different query modes
	switch {
	case inventoryHost != "":
		return showHostGroups(invManager, inventoryHost)
	case inventoryGroup != "":
		return showGroupHierarchy(invManager, inventoryGroup)
	case inventoryList:
		return listInventory(invManager)
	case inventoryGraph:
		return showInventoryGraph(invManager)
	default:
		return showInventoryStats(invManager)
	}
}

func showHostGroups(invManager *inventory.Manager, hostName string) error {
	groups := invManager.GetHostGroups(hostName)

	if len(groups) == 0 {
		fmt.Printf("Host '%s' is not in any groups\n", hostName)
		return nil
	}

	fmt.Printf("Host '%s' belongs to the following groups:\n", hostName)
	for _, group := range groups {
		fmt.Printf("  - %s\n", group)
	}

	return nil
}

func showGroupHierarchy(invManager *inventory.Manager, groupName string) error {
	hierarchy, err := invManager.GetGroupHierarchy(groupName)
	if err != nil {
		return err
	}

	fmt.Printf("Group: %s\n\n", hierarchy.Name)

	if len(hierarchy.Parents) > 0 {
		fmt.Println("Parent Groups:")
		for _, parent := range hierarchy.Parents {
			fmt.Printf("  - %s\n", parent)
		}
		fmt.Println()
	}

	if len(hierarchy.Children) > 0 {
		fmt.Println("Child Groups:")
		for _, child := range hierarchy.Children {
			fmt.Printf("  - %s\n", child)
		}
		fmt.Println()
	}

	if len(hierarchy.Hosts) > 0 {
		fmt.Printf("Direct Hosts (%d):\n", len(hierarchy.Hosts))
		for _, host := range hierarchy.Hosts {
			fmt.Printf("  - %s\n", host)
		}
		fmt.Println()
	}

	// Show all hosts including from child groups
	allHosts, err := invManager.GetAllHostsInGroup(groupName)
	if err != nil {
		return err
	}

	if len(allHosts) > len(hierarchy.Hosts) {
		fmt.Printf("All Hosts (including from child groups): %d\n", len(allHosts))
		for _, host := range allHosts {
			fmt.Printf("  - %s\n", host)
		}
	}

	return nil
}

func listInventory(invManager *inventory.Manager) error {
	hosts := invManager.ListHosts()
	groups := invManager.ListGroups()

	fmt.Printf("Inventory Summary\n")
	fmt.Printf("=================\n\n")

	fmt.Printf("Hosts (%d):\n", len(hosts))
	for _, host := range hosts {
		hostGroups := invManager.GetHostGroups(host)
		fmt.Printf("  %-30s groups: %v\n", host, hostGroups)
	}
	fmt.Println()

	fmt.Printf("Groups (%d):\n", len(groups))
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "  NAME\tHOSTS\tCHILDREN\tPARENTS")
	fmt.Fprintln(w, "  ----\t-----\t--------\t-------")

	for _, groupName := range groups {
		hierarchy, err := invManager.GetGroupHierarchy(groupName)
		if err != nil {
			continue
		}

		allHosts, _ := invManager.GetAllHostsInGroup(groupName)

		fmt.Fprintf(w, "  %s\t%d\t%d\t%d\n",
			groupName,
			len(allHosts),
			len(hierarchy.Children),
			len(hierarchy.Parents),
		)
	}
	_ = w.Flush()

	return nil
}

func showInventoryGraph(invManager *inventory.Manager) error {
	groups := invManager.ListGroups()

	fmt.Println("Group Hierarchy Graph")
	fmt.Println("=====================\n")

	visited := make(map[string]bool)

	for _, groupName := range groups {
		hierarchy, err := invManager.GetGroupHierarchy(groupName)
		if err != nil {
			continue
		}

		if len(hierarchy.Parents) == 0 && !visited[groupName] {
			printGroupTree(invManager, groupName, "", visited)
		}
	}

	return nil
}

func printGroupTree(invManager *inventory.Manager, groupName string, prefix string, visited map[string]bool) {
	if visited[groupName] {
		fmt.Printf("%s%s (already shown)\n", prefix, groupName)
		return
	}

	visited[groupName] = true

	hierarchy, err := invManager.GetGroupHierarchy(groupName)
	if err != nil {
		return
	}

	allHosts, _ := invManager.GetAllHostsInGroup(groupName)
	fmt.Printf("%s%s (%d hosts)\n", prefix, groupName, len(allHosts))

	if len(hierarchy.Hosts) > 0 {
		for i, host := range hierarchy.Hosts {
			if i == len(hierarchy.Hosts)-1 && len(hierarchy.Children) == 0 {
				fmt.Printf("%s  └─ %s\n", prefix, host)
			} else {
				fmt.Printf("%s  ├─ %s\n", prefix, host)
			}
		}
	}

	for i, child := range hierarchy.Children {
		if i == len(hierarchy.Children)-1 {
			fmt.Printf("%s  └─ ", prefix)
			printGroupTree(invManager, child, prefix+"     ", visited)
		} else {
			fmt.Printf("%s  ├─ ", prefix)
			printGroupTree(invManager, child, prefix+"  │  ", visited)
		}
	}
}

func showInventoryStats(invManager *inventory.Manager) error {
	stats := invManager.GetInventoryStats()

	fmt.Println("Inventory Statistics")
	fmt.Println("====================\n")

	if loaded, ok := stats["loaded"].(bool); ok && loaded {
		fmt.Printf("Total Groups: %v\n", stats["groups"])
		fmt.Printf("Total Hosts:  %v\n", stats["total_hosts"])
		fmt.Printf("Last Updated: %v\n", stats["last_updated"])
		fmt.Println()

		if groupStats, ok := stats["group_stats"].(map[string]interface{}); ok {
			fmt.Println("Group Details:")
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "  GROUP\tHOSTS\tCHILDREN\tVARS")
			fmt.Fprintln(w, "  -----\t-----\t--------\t----")

			for groupName, stats := range groupStats {
				if s, ok := stats.(map[string]interface{}); ok {
					fmt.Fprintf(w, "  %s\t%v\t%v\t%v\n",
						groupName,
						s["hosts"],
						s["children"],
						s["vars"],
					)
				}
			}
			_ = w.Flush()
		}
	} else {
		fmt.Println("No inventory loaded")
	}

	return nil
}
