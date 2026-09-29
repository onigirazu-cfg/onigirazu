// Command module_scaffold writes the skeleton of a new built-in module and
// its test into internal/modules:
//
//	go run ./scripts/module_scaffold -name my_module -desc "Manage X" -params path,state
//
// Then register it in NewRegistry (internal/modules/registry.go), run
// go generate ./internal/cli and document it in docs/modules/README.md.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

var validName = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

type spec struct {
	Name   string   // my_module
	Type   string   // MyModule
	Desc   string   // Manage X
	Params []string // path, state
}

func main() {
	name := flag.String("name", "", "module name: lowercase letters, digits and _")
	desc := flag.String("desc", "", "one-line description")
	params := flag.String("params", "path", "comma-separated arguments; the first is required")
	out := flag.String("output", "internal/modules", "directory of the modules package")
	force := flag.Bool("force", false, "overwrite existing files")
	flag.Parse()

	s, err := newSpec(*name, *desc, *params)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	files, err := render(s)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for file, src := range files {
		path := filepath.Join(*out, file)
		if _, err := os.Stat(path); err == nil && !*force {
			fmt.Fprintf(os.Stderr, "%s exists (use -force)\n", path)
			os.Exit(1)
		}
		if err := os.WriteFile(path, src, 0o644); err != nil { // #nosec G306 -- source file
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("wrote", path)
	}
	fmt.Printf(`Next:
  1. registry.RegisterModule(New%sModule()) in NewRegistry (internal/modules/registry.go);
     add it to checkModeModules once it supports check mode
  2. go generate ./internal/cli   (arguments known to onigirazu lint)
  3. document it in docs/modules/README.md and docs/modules/INDEX.md; add an e2e case
`, s.Type)
}

func newSpec(name, desc, params string) (spec, error) {
	if !validName.MatchString(name) {
		return spec{}, fmt.Errorf("-name %q: use lowercase letters, digits and _", name)
	}
	s := spec{Name: name, Desc: desc}
	if s.Desc == "" {
		s.Desc = "Manage " + strings.ReplaceAll(name, "_", " ")
	}
	for _, part := range strings.Split(name, "_") {
		s.Type += strings.ToUpper(part[:1]) + part[1:]
	}
	for _, p := range strings.Split(params, ",") {
		if p = strings.TrimSpace(p); p != "" {
			if !validName.MatchString(p) {
				return spec{}, fmt.Errorf("-params: %q is not a valid argument name", p)
			}
			s.Params = append(s.Params, p)
		}
	}
	if len(s.Params) == 0 {
		return spec{}, fmt.Errorf("-params: give at least one argument")
	}
	return s, nil
}

func render(s spec) (map[string][]byte, error) {
	out := map[string][]byte{}
	for file, text := range map[string]string{s.Name + ".go": moduleTmpl, s.Name + "_test.go": testTmpl} {
		var b bytes.Buffer
		if err := template.Must(template.New(file).Parse(text)).Execute(&b, s); err != nil {
			return nil, err
		}
		src, err := format.Source(b.Bytes())
		if err != nil {
			return nil, fmt.Errorf("%s: %w\n%s", file, err, b.String())
		}
		out[file] = src
	}
	return out, nil
}

const moduleTmpl = `package modules

import (
	"context"
	"fmt"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// {{.Type}}Module: {{.Desc}}
type {{.Type}}Module struct {
	*BaseModule
}

// New{{.Type}}Module creates the {{.Name}} module
func New{{.Type}}Module() *{{.Type}}Module {
	return &{{.Type}}Module{BaseModule: NewBaseModule("{{.Name}}")}
}

func (m *{{.Type}}Module) GetDescription() string { return "{{.Desc}}" }

func (m *{{.Type}}Module) Validate(args map[string]interface{}) error {
	return requireStringArg(args, "{{index .Params 0}}")
}

func (m *{{.Type}}Module) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	start := time.Now()
	result := types.TaskResult{TaskName: taskName(args), Host: host.Name, Module: m.name,
		Timestamp: start, Success: true, Output: map[string]interface{}{}}
	fail := func(msg string) (types.TaskResult, error) {
		result.Success, result.Error, result.Duration = false, msg, time.Since(start)
		return result, nil
	}
	if err := m.Validate(args); err != nil {
		return fail(err.Error())
	}
{{range .Params}}	{{.}} := getStringArg(args, "{{.}}", "")
{{end}}
	// Read the current state from the host (runOnHost runs a command, with
	// become when the task has it; readHostFile reads a file)
	current, err := runOnHost(ctx, host, args, "true")
	if err != nil {
		return fail(fmt.Sprintf("{{.Name}}: %v", err))
	}
	_ = current
{{range .Params}}	_ = {{.}}
{{end}}
	// Compare with the wanted state; nothing to do: return unchanged
	result.Changed = false
	if !result.Changed || inCheckMode(args) {
		result.Duration = time.Since(start)
		return result, nil
	}

	// Apply the change here
	result.Duration = time.Since(start)
	return result, nil
}
`

const testTmpl = `package modules

import "testing"

func Test{{.Type}}Validate(t *testing.T) {
	m := New{{.Type}}Module()
	if m.GetName() != "{{.Name}}" {
		t.Errorf("name = %q", m.GetName())
	}
	if err := m.Validate(map[string]interface{}{}); err == nil {
		t.Error("{{index .Params 0}} is required")
	}
	if err := m.Validate(map[string]interface{}{"{{index .Params 0}}": "x"}); err != nil {
		t.Error(err)
	}
}
`
