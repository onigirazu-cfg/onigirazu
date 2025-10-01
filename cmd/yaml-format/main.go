package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/onigirazu-cfg/onigirazu/pkg/formatter"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"gopkg.in/yaml.v3"
)

func main() {
	var (
		inputFile  = flag.String("input", "", "Input YAML file to format")
		outputFile = flag.String("output", "", "Output file (default: stdout)")
		inPlace    = flag.Bool("in-place", false, "Edit file in place")
		help       = flag.Bool("help", false, "Show help")
	)
	flag.Parse()

	if *help || *inputFile == "" {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s -input playbook.yml\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -input playbook.yml -output formatted.yml\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -input playbook.yml -in-place\n", os.Args[0])
		os.Exit(1)
	}

	// Read input file
	data, err := ioutil.ReadFile(*inputFile)
	if err != nil {
		log.Fatalf("Failed to read input file: %v", err)
	}

	// Parse YAML
	var playbook types.Playbook
	if err := yaml.Unmarshal(data, &playbook); err != nil {
		log.Fatalf("Failed to parse YAML: %v", err)
	}

	// Format with our custom formatter
	yamlFormatter := formatter.NewYAMLFormatter()
	formatted, err := yamlFormatter.FormatPlaybook(&playbook)
	if err != nil {
		log.Fatalf("Failed to format playbook: %v", err)
	}

	// Determine output destination
	var output string
	if *inPlace {
		output = *inputFile
	} else if *outputFile != "" {
		output = *outputFile
	}

	// Write output
	if output != "" {
		// Create directory if it doesn't exist
		dir := filepath.Dir(output)
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Failed to create output directory: %v", err)
		}

		// Write to file
		if err := os.WriteFile(output, []byte(formatted), 0644); err != nil {
			log.Fatalf("Failed to write output file: %v", err)
		}
		fmt.Fprintf(os.Stderr, "Formatted YAML written to: %s\n", output)
	} else {
		// Write to stdout
		fmt.Print(formatted)
	}
}
