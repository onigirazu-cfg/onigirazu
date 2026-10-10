// Command gen_modargs writes internal/modules/module_args.go: the arguments
// each built-in module reads, collected from internal/modules. A task with
// other arguments fails, and `onigirazu lint` warns about it. Run it with
// `go generate ./internal/modules`.
package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// argReaders are the helpers that read a module argument: helper(args, "name", ...)
var argReaders = map[string]bool{
	"getStringArg": true, "getBoolArg": true, "getIntArg": true, "getMapArg": true,
	"requireStringArg": true, "getStringListArg": true, "getFloatArg": true,
}

// multiKeyReaders read one argument under any of several names:
// listArg(args, "paths", "path")
var multiKeyReaders = map[string]bool{"listArg": true}

func main() {
	dir := "../modules"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	out := "module_args.go"
	if len(os.Args) > 2 {
		out = os.Args[2]
	}
	pkg := "modules"
	if len(os.Args) > 3 {
		pkg = os.Args[3]
	}
	table, err := collect(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(out, render(table, pkg), 0o644); err != nil { // #nosec G306 -- generated source
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// collect maps each module name to the arguments its code reads: the reads
// in the functions of the file that defines the module, and in the helpers
// they pass args to (transitively, across files)
func collect(dir string) (map[string][]string, error) {
	fset := token.NewFileSet()
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return nil, err
	}
	reads := map[string]map[string]bool{} // function -> arguments it reads itself
	calls := map[string][]string{}        // function -> functions it passes args to
	fileFuncs := map[string][]string{}    // file -> its functions
	moduleFiles := map[string][]string{}  // file -> modules it defines
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		node, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			return nil, err
		}
		for _, decl := range node.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			name, recv, recvType := funcKey(fd)
			fileFuncs[f] = append(fileFuncs[f], name)
			own := map[string]bool{}
			ast.Inspect(fd.Body, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.CallExpr:
					fn := callName(x.Fun)
					if argReaders[fn] && len(x.Args) >= 2 && isIdent(x.Args[0], "args") {
						if s, ok := stringLit(x.Args[1]); ok {
							own[s] = true
						}
					} else if multiKeyReaders[fn] && len(x.Args) >= 2 && isIdent(x.Args[0], "args") {
						for _, a := range x.Args[1:] {
							if s, ok := stringLit(a); ok {
								own[s] = true
							}
						}
					} else {
						for _, a := range x.Args {
							if isIdent(a, "args") {
								calls[name] = append(calls[name], calleeKey(x.Fun, recv, recvType))
								break
							}
						}
					}
					if (fn == "NewBaseModule" || fn == "NewBaseExecutorModule") && len(x.Args) == 1 {
						if s, ok := stringLit(x.Args[0]); ok {
							moduleFiles[f] = append(moduleFiles[f], s)
						}
					}
				case *ast.RangeStmt:
					// a loop over literal names that reads each of them as an argument
					lit, ok := x.X.(*ast.CompositeLit)
					v, vok := x.Value.(*ast.Ident)
					if !ok || !vok || !readsArgWith(x.Body, v.Name) {
						break
					}
					for _, e := range lit.Elts {
						if s, ok := stringLit(e); ok {
							own[s] = true
						}
					}
				case *ast.IndexExpr:
					if isIdent(x.X, "args") {
						if s, ok := stringLit(x.Index); ok {
							own[s] = true
						}
					}
				case *ast.KeyValueExpr:
					// BaseModule{name: "docker_container", ...}
					if isIdent(x.Key, "name") {
						if s, ok := stringLit(x.Value); ok && s != "" && strings.ToLower(s) == s && !strings.Contains(s, " ") {
							moduleFiles[f] = append(moduleFiles[f], s)
						}
					}
				}
				return true
			})
			reads[name] = own
		}
	}
	// what a function reads, through the helpers it hands args to
	var closure func(fn string, seen map[string]bool) map[string]bool
	closure = func(fn string, seen map[string]bool) map[string]bool {
		out := map[string]bool{}
		if seen[fn] {
			return out
		}
		seen[fn] = true
		for a := range reads[fn] {
			out[a] = true
		}
		for _, callee := range calls[fn] {
			for a := range closure(callee, seen) {
				out[a] = true
			}
		}
		return out
	}
	table := map[string][]string{}
	for f, names := range moduleFiles {
		set := map[string]bool{}
		for _, fn := range fileFuncs[f] {
			for a := range closure(fn, map[string]bool{}) {
				set[a] = true
			}
		}
		var list []string
		for a := range set {
			if !strings.HasPrefix(a, "_") {
				list = append(list, a)
			}
		}
		sort.Strings(list)
		for _, name := range names {
			table[name] = mergeSorted(table[name], list)
		}
	}
	return table, nil
}

// funcKey names a function, a method as Type.Method; it also returns the
// receiver variable and type, to resolve calls such as m.handleJob(args)
func funcKey(fd *ast.FuncDecl) (key, recv, recvType string) {
	if fd.Recv == nil || len(fd.Recv.List) == 0 {
		return fd.Name.Name, "", ""
	}
	field := fd.Recv.List[0]
	t := field.Type
	if star, ok := t.(*ast.StarExpr); ok {
		t = star.X
	}
	if id, ok := t.(*ast.Ident); ok {
		recvType = id.Name
	}
	if len(field.Names) > 0 {
		recv = field.Names[0].Name
	}
	return recvType + "." + fd.Name.Name, recv, recvType
}

// calleeKey names the function a call goes to: a method of the same
// receiver (m.foo) is Type.foo, a function is its name
func calleeKey(fun ast.Expr, recv, recvType string) string {
	if sel, ok := fun.(*ast.SelectorExpr); ok {
		if recv != "" && isIdent(sel.X, recv) {
			return recvType + "." + sel.Sel.Name
		}
		return sel.Sel.Name
	}
	return callName(fun)
}

func render(table map[string][]string, pkg string) []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, "// Code generated by gen_modargs; DO NOT EDIT.\n\npackage %s\n\n", pkg)
	b.WriteString("// ModuleArgs are the arguments each built-in module reads\nvar ModuleArgs = map[string][]string{\n")
	names := make([]string, 0, len(table))
	for n := range table {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		quoted := make([]string, len(table[n]))
		for i, a := range table[n] {
			quoted[i] = strconv.Quote(a)
		}
		fmt.Fprintf(&b, "\t%q: {%s},\n", n, strings.Join(quoted, ", "))
	}
	b.WriteString("}\n")
	src, err := format.Source(b.Bytes())
	if err != nil {
		return b.Bytes()
	}
	return src
}

func mergeSorted(a, b []string) []string {
	set := map[string]bool{}
	for _, x := range append(a, b...) {
		set[x] = true
	}
	out := make([]string, 0, len(set))
	for x := range set {
		out = append(out, x)
	}
	sort.Strings(out)
	return out
}

// readsArgWith tells whether body reads an argument whose name is variable v
func readsArgWith(body ast.Node, v string) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			if argReaders[callName(x.Fun)] && len(x.Args) >= 2 && isIdent(x.Args[0], "args") && isIdent(x.Args[1], v) {
				found = true
			}
		case *ast.IndexExpr:
			if isIdent(x.X, "args") && isIdent(x.Index, v) {
				found = true
			}
		}
		return !found
	})
	return found
}

func callName(e ast.Expr) string {
	switch f := e.(type) {
	case *ast.Ident:
		return f.Name
	case *ast.SelectorExpr:
		return f.Sel.Name
	}
	return ""
}

func isIdent(e ast.Expr, name string) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == name
}

func stringLit(e ast.Expr) (string, bool) {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	s, err := strconv.Unquote(lit.Value)
	return s, err == nil
}
