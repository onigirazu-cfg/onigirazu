package modules

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestFindModuleCreation(t *testing.T) {
	module := NewFindModule()
	if module == nil {
		t.Fatalf("expected NewFindModule to return a module, got nil")
	}
	if module.GetName() != "find" {
		t.Errorf("expected module name 'find', got '%s'", module.GetName())
	}
}

func TestFindModuleDescription(t *testing.T) {
	module := NewFindModule()
	desc := module.GetDescription()
	if desc == "" {
		t.Errorf("expected non-empty description")
	}
}

func TestFindModuleValidation(t *testing.T) {
	module := NewFindModule()

	testCases := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid with path and pattern",
			args: map[string]interface{}{
				"path":    "/tmp",
				"pattern": "*.txt",
			},
			wantErr: false,
		},
		{
			name: "valid with file type",
			args: map[string]interface{}{
				"path":    "/tmp",
				"pattern": "*.dat",
				"type":    "file",
			},
			wantErr: false,
		},
		{
			name: "valid with directory type",
			args: map[string]interface{}{
				"path": "/tmp",
				"type": "directory",
			},
			wantErr: false,
		},
		{
			name: "invalid type",
			args: map[string]interface{}{
				"path":    "/tmp",
				"pattern": "*",
				"type":    "invalid",
			},
			wantErr: true,
		},
		{
			name: "negative limit",
			args: map[string]interface{}{
				"path":    "/tmp",
				"pattern": "*",
				"limit":   float64(-1),
			},
			wantErr: true,
		},
		{
			name: "empty path",
			args: map[string]interface{}{
				"path":    "",
				"pattern": "*",
			},
			wantErr: true,
		},
		{
			name: "empty pattern",
			args: map[string]interface{}{
				"path":    "/tmp",
				"pattern": "",
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := module.Validate(tc.args)
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}

func TestFindModuleTypeFlag(t *testing.T) {
	module := NewFindModule()

	testCases := []struct {
		name     string
		fileType string
		expected string
	}{
		{"file", "file", "f"},
		{"directory", "directory", "d"},
		{"link", "link", "l"},
		{"socket", "socket", "s"},
		{"pipe", "pipe", "p"},
		{"block", "block", "b"},
		{"char", "char", "c"},
		{"unknown", "unknown", "f"},
		{"empty", "", "f"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := module.getTypeFlag(tc.fileType)
			if result != tc.expected {
				t.Errorf("getTypeFlag(%q) = %q, expected %q", tc.fileType, result, tc.expected)
			}
		})
	}
}

func TestFindModuleLimitValue(t *testing.T) {
	module := NewFindModule()

	testCases := []struct {
		name     string
		limit    int
		expected int
	}{
		{"zero limit", 0, 999999},
		{"negative limit", -5, 999999},
		{"positive limit", 100, 100},
		{"large limit", 1000000, 1000000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := module.getLimitValue(tc.limit)
			if result != tc.expected {
				t.Errorf("getLimitValue(%d) = %d, expected %d", tc.limit, result, tc.expected)
			}
		})
	}
}

func TestEscapeSingleQuotes(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"no quotes", "test", "test"},
		{"single quote", "it's", "it'\\''s"},
		{"multiple quotes", "it's ok's", "it'\\''s ok'\\''s"},
		{"only quotes", "'''", "'\\'''\\'''\\''"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := escapeSingleQuotes(tc.input)
			if result != tc.expected {
				t.Errorf("escapeSingleQuotes(%q) = %q, expected %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestHelperFunctions(t *testing.T) {
	t.Run("getStringArg with existing key", func(t *testing.T) {
		args := map[string]interface{}{"key": "value"}
		result := getStringArg(args, "key", "default")
		if result != "value" {
			t.Errorf("expected 'value', got %q", result)
		}
	})

	t.Run("getStringArg with missing key", func(t *testing.T) {
		args := map[string]interface{}{}
		result := getStringArg(args, "key", "default")
		if result != "default" {
			t.Errorf("expected 'default', got %q", result)
		}
	})

	t.Run("getIntArg with float64", func(t *testing.T) {
		args := map[string]interface{}{"limit": float64(100)}
		result := getIntArg(args, "limit", 0)
		if result != 100 {
			t.Errorf("expected 100, got %d", result)
		}
	})

	t.Run("getIntArg with int", func(t *testing.T) {
		args := map[string]interface{}{"limit": 50}
		result := getIntArg(args, "limit", 0)
		if result != 50 {
			t.Errorf("expected 50, got %d", result)
		}
	})

	t.Run("getIntArg with missing key", func(t *testing.T) {
		args := map[string]interface{}{}
		result := getIntArg(args, "limit", 25)
		if result != 25 {
			t.Errorf("expected 25, got %d", result)
		}
	})

	t.Run("getBoolArg with true", func(t *testing.T) {
		args := map[string]interface{}{"flag": true}
		result := getBoolArg(args, "flag", false)
		if result != true {
			t.Errorf("expected true, got %v", result)
		}
	})

	t.Run("getBoolArg with missing key", func(t *testing.T) {
		args := map[string]interface{}{}
		result := getBoolArg(args, "flag", true)
		if result != true {
			t.Errorf("expected true, got %v", result)
		}
	})
}

func TestFindModuleExecute(t *testing.T) {
	module := NewFindModule()

	// Create a mock host
	host := types.Host{
		Name: "localhost",
	}

	t.Run("execute with missing path", func(t *testing.T) {
		args := map[string]interface{}{
			"pattern": "*.txt",
		}
		result, _ := module.Execute(context.Background(), host, args)
		if !result.Failed {
			t.Errorf("expected failed result for missing path argument")
		}
	})

	t.Run("execute with invalid type", func(t *testing.T) {
		args := map[string]interface{}{
			"path":    "/tmp",
			"pattern": "*",
			"type":    "invalid_type",
		}
		result, _ := module.Execute(context.Background(), host, args)
		if !result.Failed {
			t.Errorf("expected failed result for invalid type")
		}
	})
}

// one command lists and stats the files; names with spaces and quotes too
func TestFindModuleStats(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"a.txt", "b c.txt", "it's.txt", "skip.log"} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("12345"), 0o640); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "d.txt"), 0o755); err != nil {
		t.Fatal(err)
	}
	host := types.Host{Name: "localhost", Address: "127.0.0.1"}
	result, err := NewFindModule().Execute(context.Background(), host, map[string]interface{}{"paths": dir, "patterns": "*.txt"})
	if err != nil {
		t.Fatal(err)
	}
	files, _ := result.Output["files"].([]map[string]interface{})
	byName := map[string]map[string]interface{}{}
	for _, f := range files {
		byName[f["name"].(string)] = f
	}
	if len(byName) != 3 || byName["b c.txt"] == nil || byName["it's.txt"] == nil {
		t.Fatalf("files %v", files)
	}
	a := byName["a.txt"]
	if a["size"] != int64(5) || a["mode"] != "0640" || a["isreg"] != true || a["path"] != filepath.Join(dir, "a.txt") {
		t.Errorf("a.txt %v", a)
	}
	if _, ok := a["mtime"].(float64); !ok {
		t.Errorf("mtime %v", a["mtime"])
	}

	result, _ = NewFindModule().Execute(context.Background(), host, map[string]interface{}{"paths": dir, "patterns": "*.txt", "file_type": "directory"})
	if result.Output["matched"] != 1 {
		t.Errorf("directories %v", result.Output["files"])
	}
	result, _ = NewFindModule().Execute(context.Background(), host, map[string]interface{}{"paths": dir, "patterns": "*.none"})
	if result.Output["matched"] != 0 || !result.Success {
		t.Errorf("no match %v", result)
	}
}
