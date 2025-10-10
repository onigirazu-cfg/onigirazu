package main

import (
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path/filepath"
)

type Package struct {
	Name        string
	Path        string
	Description string
	Content     string
}

type Documentation struct {
	Title    string
	Packages []Package
}

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}} - API Documentation</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
            background-color: #f8f9fa;
        }
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 2rem;
            border-radius: 10px;
            margin-bottom: 2rem;
            text-align: center;
        }
        .header h1 {
            margin: 0;
            font-size: 2.5rem;
        }
        .header p {
            margin: 0.5rem 0 0 0;
            opacity: 0.9;
        }
        .nav {
            background: white;
            padding: 1rem;
            border-radius: 8px;
            margin-bottom: 2rem;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .nav h2 {
            margin-top: 0;
            color: #495057;
        }
        .nav ul {
            list-style: none;
            padding: 0;
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 0.5rem;
        }
        .nav li {
            margin: 0;
        }
        .nav a {
            color: #007bff;
            text-decoration: none;
            padding: 0.5rem;
            display: block;
            border-radius: 4px;
            transition: background-color 0.2s;
        }
        .nav a:hover {
            background-color: #f8f9fa;
            text-decoration: underline;
        }
        .package {
            background: white;
            margin-bottom: 2rem;
            border-radius: 8px;
            overflow: hidden;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .package-header {
            background: #495057;
            color: white;
            padding: 1rem;
        }
        .package-header h2 {
            margin: 0;
            font-size: 1.5rem;
        }
        .package-content {
            padding: 1.5rem;
        }
        .package-content pre {
            background: #f8f9fa;
            padding: 1rem;
            border-radius: 4px;
            overflow-x: auto;
            border-left: 4px solid #007bff;
        }
        .package-content code {
            background: #f8f9fa;
            padding: 0.2rem 0.4rem;
            border-radius: 3px;
            font-family: 'SF Mono', Monaco, 'Cascadia Code', 'Roboto Mono', Consolas, 'Courier New', monospace;
        }
        .footer {
            text-align: center;
            padding: 2rem;
            color: #6c757d;
            border-top: 1px solid #dee2e6;
            margin-top: 3rem;
        }
        @media (max-width: 768px) {
            body {
                padding: 10px;
            }
            .header h1 {
                font-size: 2rem;
            }
            .nav ul {
                grid-template-columns: 1fr;
            }
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>{{.Title}}</h1>
        <p>Comprehensive API Documentation</p>
    </div>

    <div class="nav">
        <h2>📚 Package Index</h2>
        <ul>
            {{range .Packages}}
            <li><a href="#{{.Name}}">{{.Path}}</a></li>
            {{end}}
        </ul>
    </div>

    {{range .Packages}}
    <div class="package" id="{{.Name}}">
        <div class="package-header">
            <h2>📦 {{.Path}}</h2>
        </div>
        <div class="package-content">
            <pre><code>{{.Content}}</code></pre>
        </div>
    </div>
    {{end}}

    <div class="footer">
        <p>Generated with ❤️ for Onigirazu Project</p>
        <p>Documentation auto-generated from Go source code</p>
    </div>
</body>
</html>`

func main() {
	packages := []Package{
		{Name: "types", Path: "pkg/types", Description: "Core types and interfaces"},
		{Name: "utils", Path: "pkg/utils", Description: "Utility functions"},
		{Name: "config", Path: "internal/config", Description: "Configuration management"},
		{Name: "core", Path: "internal/core", Description: "Core engine"},
		{Name: "engine", Path: "internal/engine", Description: "Execution engine"},
		{Name: "modules", Path: "internal/modules", Description: "Built-in modules"},
		{Name: "parser", Path: "internal/parser", Description: "YAML parsing"},
		{Name: "workflow", Path: "internal/workflow", Description: "Workflow orchestration"},
	}

	// Generate documentation content for each package
	for i, pkg := range packages {
		// #nosec G204 -- pkg.Path is from a hardcoded list of internal packages, not user input
		cmd := exec.Command("go", "doc", "-all", "./"+pkg.Path)
		output, err := cmd.Output()
		if err != nil {
			fmt.Printf("Warning: Could not generate docs for %s: %v\n", pkg.Path, err)
			packages[i].Content = fmt.Sprintf("Documentation not available for %s", pkg.Path)
		} else {
			packages[i].Content = string(output)
		}
	}

	doc := Documentation{
		Title:    "Onigirazu API Documentation",
		Packages: packages,
	}

	// Create output directory
	if err := os.MkdirAll("docs/api", 0750); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Generate HTML
	tmpl, err := template.New("docs").Parse(htmlTemplate)
	if err != nil {
		fmt.Printf("Error parsing template: %v\n", err)
		os.Exit(1)
	}

	outputFile := filepath.Join("docs", "api", "index.html")
	file, err := os.Create(outputFile) // #nosec G304 -- outputFile is constructed from fixed paths
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		// Best-effort cleanup in case of panic
		_ = file.Close()
	}()

	err = tmpl.Execute(file, doc)
	if err != nil {
		fmt.Printf("Error executing template: %v\n", err)
		os.Exit(1)
	}

	// Ensure data is flushed to disk before closing
	if err = file.Sync(); err != nil {
		fmt.Printf("Error syncing file: %v\n", err)
		os.Exit(1)
	}

	// Explicitly close and handle any errors
	if err = file.Close(); err != nil {
		fmt.Printf("Error closing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ HTML documentation generated: %s\n", outputFile)
	fmt.Printf("🌐 Open file://%s in your browser to view\n", filepath.Join(getCurrentDir(), outputFile))
}

func getCurrentDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	return dir
}
