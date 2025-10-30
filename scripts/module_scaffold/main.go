package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	// Define command-line flags
	moduleName := flag.String("name", "", "Module name (required, lowercase with underscores)")
	description := flag.String("desc", "", "Module description")
	outputDir := flag.String("output", "/Users/denys.rastiegaiev/work/onigirazu_project/onigirazu/internal/modules", "Output directory")
	parameters := flag.String("params", "", "Comma-separated list of parameters")
	idempotent := flag.Bool("idempotent", true, "Generate idempotency tests")
	help := flag.Bool("help", false, "Show help message")

	flag.Parse()

	if *help || *moduleName == "" {
		printHelp()
		return
	}

	// Parse parameters
	var params []string
	if *parameters != "" {
		params = strings.Split(*parameters, ",")
		for i, p := range params {
			params[i] = strings.TrimSpace(p)
		}
	}

	// Create scaffold
	scaffold := &ModuleScaffold{
		ModuleName:        *moduleName,
		Description:       *description,
		Parameters:        params,
		OutputDir:         *outputDir,
		IncludeIdempotent: *idempotent,
	}

	// Generate
	if err := scaffold.Generate(); err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		os.Exit(1)
	}

	// Print summary
	printSummary(scaffold)
}

func printHelp() {
	fmt.Println(`
╔════════════════════════════════════════════════════════════════╗
║         Onigirazu Module Scaffolding Tool                      ║
║  Generate boilerplate code for new Onigirazu modules           ║
╚════════════════════════════════════════════════════════════════╝

USAGE:
  go run ./scripts/module_scaffold -name MODULE_NAME [options]

REQUIRED FLAGS:
  -name string
    Module name (must be lowercase with underscores)
    Example: my_awesome_module

OPTIONAL FLAGS:
  -desc string
    Module description (default: "")
  -output string
    Output directory (default: internal/modules)
  -params string
    Comma-separated parameter names (default: "")
    Example: "target,action,state"
  -idempotent bool
    Generate idempotency tests (default: true)
  -help
    Show this help message

EXAMPLES:

1. Generate basic module:
   go run ./scripts/module_scaffold -name hello_world

2. Generate module with parameters:
   go run ./scripts/module_scaffold \
     -name package_manager \
     -desc "Manage system packages" \
     -params "name,state,version"

3. Generate module without idempotency tests:
   go run ./scripts/module_scaffold -name quick_module -idempotent=false

OUTPUT FILES:
  - MODULE_NAME.go              (Module implementation)
  - MODULE_NAME_test.go         (Unit tests)
  - MODULE_NAME_idempotency_test.go (Idempotency tests, if enabled)

NEXT STEPS:
  1. Open the generated files
  2. Implement the Execute() method
  3. Update parameter validation in Validate()
  4. Add test cases in _test.go
  5. Run: go test ./internal/modules -run MODULE_NAME -v
  6. Ensure 100% coverage for new code

TEMPLATE INCLUDES:
  ✓ Proper package and import structure
  ✓ BaseModule inheritance
  ✓ Validation method template
  ✓ Execute method scaffold
  ✓ Idempotency support
  ✓ Unit tests with table-driven approach
  ✓ Benchmark tests
  ✓ Mock objects for testing
  ✓ JSDoc-style comments

BEST PRACTICES:
  • Always implement error handling
  • Set the Changed flag correctly
  • Make modules idempotent when possible
  • Write comprehensive tests
  • Use table-driven tests for multiple cases
  • Document all parameters in validation
  • Run: go test ./internal/modules -cover
`)
}

func printSummary(scaffold *ModuleScaffold) {
	fmt.Printf(`
✅ Module scaffold generated successfully!

📋 Module Details:
   Name:        %s
   Package:     %s
   Output Dir:  %s
   Parameters:  %d
   Idempotent:  %v

📁 Generated Files:
   ✓ %s/%s.go
   ✓ %s/%s_test.go
%s
🚀 Next Steps:
   1. Edit the module implementation (%s.go)
   2. Update the Validate() method
   3. Implement the Execute() method
   4. Add test cases to %s_test.go
   5. Run tests: cd internal/modules && go test -run %s -v
   6. Check coverage: go test -cover

📖 Documentation:
   - Parameter names: %s
   - Description: %s

💡 Tips:
   • Use BaseModule.Logger for logging
   • Always set Changed flag in results
   • Test with both local and remote execution
   • Ensure idempotency when possible
   • Add examples to YAML for users

Happy coding! 🎉
`,
		scaffold.ModuleName,
		scaffold.PackageName,
		scaffold.OutputDir,
		len(scaffold.Parameters),
		scaffold.IncludeIdempotent,
		scaffold.OutputDir,
		scaffold.ModuleName,
		scaffold.OutputDir,
		scaffold.ModuleName,
		getIdempotentFileInfo(scaffold),
		scaffold.ModuleName,
		scaffold.ModuleName,
		scaffold.ModuleName,
		formatParameters(scaffold.Parameters),
		scaffold.Description,
	)
}

func getIdempotentFileInfo(scaffold *ModuleScaffold) string {
	if scaffold.IncludeIdempotent {
		return fmt.Sprintf("   ✓ %s/%s_idempotency_test.go\n", scaffold.OutputDir, scaffold.ModuleName)
	}
	return ""
}

func formatParameters(params []string) string {
	if len(params) == 0 {
		return "None"
	}
	return strings.Join(params, ", ")
}
