package folderstructure

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestVariableLoaderLoadVariables(t *testing.T) {
	tmpDir := t.TempDir()

	// Create defaults and vars directories
	defaultsDir := filepath.Join(tmpDir, "defaults")
	varsDir := filepath.Join(tmpDir, "vars")

	if err := os.Mkdir(defaultsDir, 0755); err != nil {
		t.Fatalf("failed to create defaults dir: %v", err)
	}
	if err := os.Mkdir(varsDir, 0755); err != nil {
		t.Fatalf("failed to create vars dir: %v", err)
	}

	// Create defaults/main.yml
	defaultsData := map[string]interface{}{
		"default_var": "default_value",
		"shared_var":  "default",
	}
	defaultsFile := filepath.Join(defaultsDir, "main.yml")
	if data, err := yaml.Marshal(defaultsData); err != nil {
		t.Fatalf("failed to marshal defaults: %v", err)
	} else if err := os.WriteFile(defaultsFile, data, 0644); err != nil {
		t.Fatalf("failed to write defaults: %v", err)
	}

	// Create vars/main.yml
	varsData := map[string]interface{}{
		"var_var":    "var_value",
		"shared_var": "override",
	}
	varsFile := filepath.Join(varsDir, "main.yml")
	if data, err := yaml.Marshal(varsData); err != nil {
		t.Fatalf("failed to marshal vars: %v", err)
	} else if err := os.WriteFile(varsFile, data, 0644); err != nil {
		t.Fatalf("failed to write vars: %v", err)
	}

	detector := NewDetector()
	loader := NewVariableLoader(detector)

	varSet, err := loader.LoadVariables(tmpDir)
	if err != nil {
		t.Fatalf("failed to load variables: %v", err)
	}

	// Check that defaults were loaded
	if val, ok := varSet.Variables["default_var"]; !ok || val != "default_value" {
		t.Error("expected default_var to be loaded")
	}

	// Check that vars were loaded
	if val, ok := varSet.Variables["var_var"]; !ok || val != "var_value" {
		t.Error("expected var_var to be loaded")
	}

	// Check that vars has higher priority (should override defaults)
	if val, ok := varSet.Variables["shared_var"]; !ok || val != "override" {
		t.Errorf("expected shared_var to be 'override', got %v", val)
	}
}

func TestVariableLoaderMergeVariables(t *testing.T) {
	set1 := &VariableSet{
		Variables: map[string]interface{}{
			"var1": "value1",
			"var2": "value2_set1",
		},
		Metadata: map[string]VariableSource{
			"var1": {Source: "set1", Priority: 1},
			"var2": {Source: "set1", Priority: 1},
		},
	}

	set2 := &VariableSet{
		Variables: map[string]interface{}{
			"var2": "value2_set2",
			"var3": "value3",
		},
		Metadata: map[string]VariableSource{
			"var2": {Source: "set2", Priority: 1},
			"var3": {Source: "set2", Priority: 1},
		},
	}

	detector := NewDetector()
	loader := NewVariableLoader(detector)

	merged := loader.MergeVariables(set1, set2)

	// Check all vars are present
	if len(merged.Variables) != 3 {
		t.Errorf("expected 3 variables, got %d", len(merged.Variables))
	}

	// Check values
	if val, ok := merged.Variables["var1"]; !ok || val != "value1" {
		t.Error("expected var1 from set1")
	}

	if val, ok := merged.Variables["var3"]; !ok || val != "value3" {
		t.Error("expected var3 from set2")
	}
}

func TestVariableLoaderCaching(t *testing.T) {
	tmpDir := t.TempDir()

	// Create defaults directory
	defaultsDir := filepath.Join(tmpDir, "defaults")
	if err := os.Mkdir(defaultsDir, 0755); err != nil {
		t.Fatalf("failed to create defaults dir: %v", err)
	}

	// Create defaults/main.yml
	defaultsData := map[string]interface{}{
		"var1": "value1",
	}
	defaultsFile := filepath.Join(defaultsDir, "main.yml")
	if data, err := yaml.Marshal(defaultsData); err != nil {
		t.Fatalf("failed to marshal defaults: %v", err)
	} else if err := os.WriteFile(defaultsFile, data, 0644); err != nil {
		t.Fatalf("failed to write defaults: %v", err)
	}

	detector := NewDetector()
	loader := NewVariableLoader(detector)

	// First load
	varSet1, err := loader.LoadVariables(tmpDir)
	if err != nil {
		t.Fatalf("first load failed: %v", err)
	}

	// Second load should come from cache
	varSet2, err := loader.LoadVariables(tmpDir)
	if err != nil {
		t.Fatalf("second load failed: %v", err)
	}

	// Both should have the same variables
	if len(varSet1.Variables) != len(varSet2.Variables) {
		t.Error("cached variables should match original")
	}
}

func TestVariableLoaderEmptyDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create empty defaults and vars directories
	if err := os.Mkdir(filepath.Join(tmpDir, "defaults"), 0755); err != nil {
		t.Fatalf("failed to create defaults: %v", err)
	}
	if err := os.Mkdir(filepath.Join(tmpDir, "vars"), 0755); err != nil {
		t.Fatalf("failed to create vars: %v", err)
	}

	detector := NewDetector()
	loader := NewVariableLoader(detector)

	varSet, err := loader.LoadVariables(tmpDir)
	if err != nil {
		t.Fatalf("failed to load variables: %v", err)
	}

	// Should have empty variables
	if len(varSet.Variables) != 0 {
		t.Errorf("expected 0 variables, got %d", len(varSet.Variables))
	}
}

func TestVariableLoaderClearCache(t *testing.T) {
	tmpDir := t.TempDir()

	// Create defaults directory
	defaultsDir := filepath.Join(tmpDir, "defaults")
	if err := os.Mkdir(defaultsDir, 0755); err != nil {
		t.Fatalf("failed to create defaults: %v", err)
	}

	// Create defaults/main.yml
	defaultsFile := filepath.Join(defaultsDir, "main.yml")
	if err := os.WriteFile(defaultsFile, []byte("var1: value1"), 0644); err != nil {
		t.Fatalf("failed to write defaults: %v", err)
	}

	detector := NewDetector()
	loader := NewVariableLoader(detector)

	// Load variables to populate cache
	_, err := loader.LoadVariables(tmpDir)
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}

	// Clear cache
	loader.ClearCache()

	// Should still be able to load after cache clear
	varSet, err := loader.LoadVariables(tmpDir)
	if err != nil {
		t.Fatalf("failed to load after cache clear: %v", err)
	}

	if len(varSet.Variables) == 0 {
		t.Error("expected variables after cache clear")
	}
}
