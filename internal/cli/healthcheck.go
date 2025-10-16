package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/onigirazu-cfg/onigirazu/internal/cache"
	"github.com/onigirazu-cfg/onigirazu/internal/healthcheck"
	"github.com/onigirazu-cfg/onigirazu/internal/inventory"
	"github.com/onigirazu-cfg/onigirazu/internal/logger"
	"github.com/onigirazu-cfg/onigirazu/internal/parser"
	"github.com/onigirazu-cfg/onigirazu/internal/template"
)

var (
	healthcheckInventoryFile   string
	healthcheckFormat          string
	healthcheckOutput          string
	healthcheckTimeout         int
	healthcheckDiskThreshold   int
	healthcheckMemoryThreshold int
	healthcheckCPUThreshold    int
	healthcheckServices        string
	healthcheckChecks          string
	healthcheckVerbose         bool
	healthcheckSkipUnavailable bool
)

var healthcheckCmd = &cobra.Command{
	Use:   "healthcheck",
	Short: "Check health of inventory hosts",
	Long:  `Performs comprehensive health checks on all inventory hosts including connectivity, disk space, memory, CPU, and services`,
	RunE:  runHealthcheck,
}

func init() {
	healthcheckCmd.Flags().StringVarP(&healthcheckInventoryFile, "inventory", "i", "", "Path to inventory file")
	healthcheckCmd.Flags().StringVar(&healthcheckFormat, "format", "text", "Output format: text, json, csv, html, markdown")
	healthcheckCmd.Flags().StringVar(&healthcheckOutput, "output", "", "Output file (if not specified, prints to stdout)")
	healthcheckCmd.Flags().IntVar(&healthcheckTimeout, "timeout", 30, "Timeout in seconds for each host check")
	healthcheckCmd.Flags().IntVar(&healthcheckDiskThreshold, "disk-threshold", 80, "Disk usage warning threshold (percentage)")
	healthcheckCmd.Flags().IntVar(&healthcheckMemoryThreshold, "memory-threshold", 80, "Memory usage warning threshold (percentage)")
	healthcheckCmd.Flags().IntVar(&healthcheckCPUThreshold, "cpu-threshold", 80, "CPU usage warning threshold (percentage)")
	healthcheckCmd.Flags().StringVar(&healthcheckServices, "services", "", "Comma-separated list of services to check")
	healthcheckCmd.Flags().StringVar(&healthcheckChecks, "checks", "connectivity,disk_space,memory", "Comma-separated list of checks to perform")
	healthcheckCmd.Flags().BoolVar(&healthcheckVerbose, "verbose", false, "Verbose output")
	healthcheckCmd.Flags().BoolVar(&healthcheckSkipUnavailable, "skip-unavailable", false, "Skip unavailable hosts instead of failing")
}

func runHealthcheck(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Create logger
	logLevel := "info"
	if healthcheckVerbose {
		logLevel = "debug"
	}

	log, err := logger.NewEnhancedLogger(logLevel, "text", os.Stdout)
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}

	// Initialize parser and cache
	invParser := parser.NewInventoryParser(log)
	cacheManager := cache.NewManager(0)

	// Find inventory file if not specified
	if healthcheckInventoryFile == "" {
		var err error
		healthcheckInventoryFile, err = invParser.FindInventoryFile(".")
		if err != nil {
			return fmt.Errorf("failed to find inventory file: %w", err)
		}
	}

	// Load inventory
	templateEngine := template.NewEngine()
	playbookParser := parser.NewEnhancedParser(templateEngine, log)
	invManager := inventory.NewManager(playbookParser, log, cacheManager)

	if err := invManager.LoadInventory(ctx, healthcheckInventoryFile); err != nil {
		return fmt.Errorf("failed to load inventory: %w", err)
	}

	// Get all hosts from inventory
	hosts, err := invManager.GetHosts("*")
	if err != nil {
		return fmt.Errorf("failed to get hosts: %w", err)
	}
	if len(hosts) == 0 {
		return fmt.Errorf("no hosts found in inventory")
	}

	log.Info("Starting health check on %d hosts", len(hosts))

	// Build health check configuration
	config := healthcheck.NewCheckConfig()
	config.Timeout = time.Duration(healthcheckTimeout) * time.Second
	config.DiskSpaceThreshold = healthcheckDiskThreshold
	config.MemoryThreshold = healthcheckMemoryThreshold
	config.CPUThreshold = healthcheckCPUThreshold
	config.SkipUnavailableHosts = healthcheckSkipUnavailable

	// Parse check types
	config.CheckTypes = parseCheckTypes(healthcheckChecks)

	// Parse services to check
	if healthcheckServices != "" {
		config.Services = strings.Split(healthcheckServices, ",")
		for i := range config.Services {
			config.Services[i] = strings.TrimSpace(config.Services[i])
		}
	}

	// Run health checks
	checker := healthcheck.NewChecker(hosts, config, log)
	report, err := checker.CheckAll(ctx)
	if err != nil && !healthcheckSkipUnavailable {
		log.Warn("Health check completed with errors: %v", err)
	}

	// Format and output report
	if err := outputHealthcheckReport(report); err != nil {
		return fmt.Errorf("failed to output report: %w", err)
	}

	// Exit with appropriate code based on status
	switch report.OverallStatus {
	case healthcheck.StatusCritical:
		return fmt.Errorf("health check failed: critical issues found")
	case healthcheck.StatusWarning:
		log.Warn("Health check completed with warnings")
	}

	return nil
}

// outputHealthcheckReport outputs the health check report
func outputHealthcheckReport(report *healthcheck.HealthCheckReport) error {
	reporter := healthcheck.NewReporter(true)

	switch healthcheckFormat {
	case "text":
		reporter.PrintReport(report)
	case "json":
		json, err := reporter.FormatJSON(report)
		if err != nil {
			return err
		}
		if healthcheckOutput != "" {
			if err := os.WriteFile(healthcheckOutput, []byte(json), 0644); err != nil {
				return fmt.Errorf("failed to write output file: %w", err)
			}
			fmt.Printf("Report saved to: %s\n", healthcheckOutput)
		} else {
			fmt.Println(json)
		}
	case "csv":
		csv := reporter.FormatCSV(report)
		if healthcheckOutput != "" {
			if err := os.WriteFile(healthcheckOutput, []byte(csv), 0644); err != nil {
				return fmt.Errorf("failed to write output file: %w", err)
			}
			fmt.Printf("Report saved to: %s\n", healthcheckOutput)
		} else {
			fmt.Println(csv)
		}
	case "html":
		html := reporter.FormatHTML(report)
		if healthcheckOutput != "" {
			if err := os.WriteFile(healthcheckOutput, []byte(html), 0644); err != nil {
				return fmt.Errorf("failed to write output file: %w", err)
			}
			fmt.Printf("Report saved to: %s\n", healthcheckOutput)
		} else {
			fmt.Println(html)
		}
	case "markdown", "md":
		md := reporter.FormatMarkdown(report)
		if healthcheckOutput != "" {
			if err := os.WriteFile(healthcheckOutput, []byte(md), 0644); err != nil {
				return fmt.Errorf("failed to write output file: %w", err)
			}
			fmt.Printf("Report saved to: %s\n", healthcheckOutput)
		} else {
			fmt.Println(md)
		}
	default:
		return fmt.Errorf("unsupported output format: %s", healthcheckFormat)
	}

	return nil
}

// parseCheckTypes parses comma-separated check types
func parseCheckTypes(checksStr string) []healthcheck.CheckType {
	checks := []healthcheck.CheckType{}
	for _, check := range strings.Split(checksStr, ",") {
		check = strings.TrimSpace(strings.ToLower(check))
		switch check {
		case "connectivity":
			checks = append(checks, healthcheck.CheckConnectivity)
		case "disk_space", "disk":
			checks = append(checks, healthcheck.CheckDiskSpace)
		case "memory", "mem":
			checks = append(checks, healthcheck.CheckMemory)
		case "cpu":
			checks = append(checks, healthcheck.CheckCPU)
		case "services":
			checks = append(checks, healthcheck.CheckServices)
		case "network", "net":
			checks = append(checks, healthcheck.CheckNetwork)
		case "certificates", "cert":
			checks = append(checks, healthcheck.CheckCertificates)
		}
	}
	if len(checks) == 0 {
		checks = []healthcheck.CheckType{healthcheck.CheckConnectivity}
	}
	return checks
}
