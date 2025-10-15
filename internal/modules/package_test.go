package modules

import (
	"testing"
	"time"
)

// TestUnifiedPackageModule_GetName tests the GetName method
func TestUnifiedPackageModule_GetName(t *testing.T) {
	module := NewUnifiedPackageModule()

	name := module.GetName()
	if name != "package" {
		t.Errorf("Expected name 'package', got '%s'", name)
	}
}

// TestUnifiedPackageModule_GetDescription tests the GetDescription method
func TestUnifiedPackageModule_GetDescription(t *testing.T) {
	module := NewUnifiedPackageModule()

	desc := module.GetDescription()
	if desc == "" {
		t.Errorf("Expected non-empty description")
	}

	expectedDesc := "Unified package management with advanced features"
	if desc != expectedDesc {
		t.Errorf("Expected description '%s', got '%s'", expectedDesc, desc)
	}
}

// TestNewUnifiedPackageModule tests module creation
func TestNewUnifiedPackageModule(t *testing.T) {
	module := NewUnifiedPackageModule()

	if module == nil {
		t.Fatalf("Expected non-nil module")
	}

	if module.GetName() != "package" {
		t.Errorf("Expected module name 'package', got '%s'", module.GetName())
	}
}

// TestUnifiedPackageModule_Validate tests the Validate method
func TestUnifiedPackageModule_Validate(t *testing.T) {
	module := NewUnifiedPackageModule()

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid with name only",
			args: map[string]interface{}{
				"name": "nginx",
			},
			wantErr: false,
		},
		{
			name: "valid with name and state present",
			args: map[string]interface{}{
				"name":  "nginx",
				"state": "present",
			},
			wantErr: false,
		},
		{
			name: "valid with name and state absent",
			args: map[string]interface{}{
				"name":  "nginx",
				"state": "absent",
			},
			wantErr: false,
		},
		{
			name: "valid with name and state latest",
			args: map[string]interface{}{
				"name":  "nginx",
				"state": "latest",
			},
			wantErr: false,
		},
		{
			name: "valid with additional parameters",
			args: map[string]interface{}{
				"name":    "nginx",
				"state":   "present",
				"version": "1.18.0",
			},
			wantErr: false,
		},
		{
			name:    "missing name",
			args:    map[string]interface{}{},
			wantErr: true,
			errMsg:  "name parameter is required",
		},
		{
			name: "missing name with state",
			args: map[string]interface{}{
				"state": "present",
			},
			wantErr: true,
			errMsg:  "name parameter is required",
		},
		{
			name: "invalid state",
			args: map[string]interface{}{
				"name":  "nginx",
				"state": "invalid",
			},
			wantErr: true,
			errMsg:  "invalid state: invalid (must be one of: present, absent, latest)",
		},
		{
			name: "invalid state - installed",
			args: map[string]interface{}{
				"name":  "nginx",
				"state": "installed",
			},
			wantErr: true,
			errMsg:  "invalid state: installed (must be one of: present, absent, latest)",
		},
		{
			name: "invalid state - removed",
			args: map[string]interface{}{
				"name":  "nginx",
				"state": "removed",
			},
			wantErr: true,
			errMsg:  "invalid state: removed (must be one of: present, absent, latest)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := module.Validate(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("Validate() error = %v, want %v", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			}
		})
	}
}

// TestPackageStateCache tests the PackageStateCache functionality
func TestPackageStateCache(t *testing.T) {
	t.Run("NewPackageStateCache", func(t *testing.T) {
		cache := NewPackageStateCache(5 * time.Minute)
		if cache == nil {
			t.Fatal("Expected non-nil cache")
		}
		if cache.ttl != 5*time.Minute {
			t.Errorf("Expected TTL 5m, got %v", cache.ttl)
		}
	})

	t.Run("Set and Get", func(t *testing.T) {
		cache := NewPackageStateCache(5 * time.Minute)
		state := &PackageState{
			Name:      "nginx",
			Installed: true,
			Version:   "1.18.0",
		}

		cache.Set("nginx", state)
		retrieved, ok := cache.Get("nginx")
		if !ok {
			t.Error("Expected to find cached state")
		}
		if retrieved.Name != "nginx" {
			t.Errorf("Expected name 'nginx', got '%s'", retrieved.Name)
		}
		if retrieved.Version != "1.18.0" {
			t.Errorf("Expected version '1.18.0', got '%s'", retrieved.Version)
		}
	})

	t.Run("Get non-existent", func(t *testing.T) {
		cache := NewPackageStateCache(5 * time.Minute)
		_, ok := cache.Get("nonexistent")
		if ok {
			t.Error("Expected not to find non-existent package")
		}
	})

	t.Run("TTL expiration", func(t *testing.T) {
		cache := NewPackageStateCache(100 * time.Millisecond)
		state := &PackageState{
			Name:      "nginx",
			Installed: true,
		}

		cache.Set("nginx", state)

		// Should be available immediately
		_, ok := cache.Get("nginx")
		if !ok {
			t.Error("Expected to find cached state immediately")
		}

		// Wait for expiration
		time.Sleep(150 * time.Millisecond)

		// Should be expired
		_, ok = cache.Get("nginx")
		if ok {
			t.Error("Expected cache entry to be expired")
		}
	})

	t.Run("Delete", func(t *testing.T) {
		cache := NewPackageStateCache(5 * time.Minute)
		state := &PackageState{
			Name:      "nginx",
			Installed: true,
		}

		cache.Set("nginx", state)
		cache.Delete("nginx")

		_, ok := cache.Get("nginx")
		if ok {
			t.Error("Expected cache entry to be deleted")
		}
	})

	t.Run("Clear", func(t *testing.T) {
		cache := NewPackageStateCache(5 * time.Minute)

		cache.Set("nginx", &PackageState{Name: "nginx", Installed: true})
		cache.Set("apache", &PackageState{Name: "apache", Installed: true})
		cache.Set("mysql", &PackageState{Name: "mysql", Installed: true})

		cache.Clear()

		_, ok1 := cache.Get("nginx")
		_, ok2 := cache.Get("apache")
		_, ok3 := cache.Get("mysql")

		if ok1 || ok2 || ok3 {
			t.Error("Expected all cache entries to be cleared")
		}
	})

	t.Run("Stats - hits and misses", func(t *testing.T) {
		cache := NewPackageStateCache(5 * time.Minute)
		state := &PackageState{Name: "nginx", Installed: true}

		cache.Set("nginx", state)

		// Hit
		cache.Get("nginx")
		// Miss
		cache.Get("nonexistent")
		// Another hit
		cache.Get("nginx")
		// Another miss
		cache.Get("another-nonexistent")

		hits, misses := cache.Stats()
		if hits != 2 {
			t.Errorf("Expected 2 hits, got %d", hits)
		}
		if misses != 2 {
			t.Errorf("Expected 2 misses, got %d", misses)
		}
	})
}

// TestPackageMetrics tests the PackageMetrics functionality
func TestPackageMetrics(t *testing.T) {
	t.Run("RecordOperation - successful install", func(t *testing.T) {
		metrics := &PackageMetrics{}
		op := &PackageOperation{
			Package:   "nginx",
			Operation: "install",
			Success:   true,
			Changed:   true,
			Duration:  100 * time.Millisecond,
		}

		metrics.RecordOperation(op)

		if metrics.TotalOperations != 1 {
			t.Errorf("Expected 1 total operation, got %d", metrics.TotalOperations)
		}
		if metrics.SuccessfulOps != 1 {
			t.Errorf("Expected 1 successful operation, got %d", metrics.SuccessfulOps)
		}
		if metrics.FailedOps != 0 {
			t.Errorf("Expected 0 failed operations, got %d", metrics.FailedOps)
		}
		if metrics.PackagesInstalled != 1 {
			t.Errorf("Expected 1 package installed, got %d", metrics.PackagesInstalled)
		}
	})

	t.Run("RecordOperation - failed operation", func(t *testing.T) {
		metrics := &PackageMetrics{}
		op := &PackageOperation{
			Package:   "nginx",
			Operation: "install",
			Success:   false,
			Changed:   false,
			Duration:  50 * time.Millisecond,
			Error:     "installation failed",
		}

		metrics.RecordOperation(op)

		if metrics.SuccessfulOps != 0 {
			t.Errorf("Expected 0 successful operations, got %d", metrics.SuccessfulOps)
		}
		if metrics.FailedOps != 1 {
			t.Errorf("Expected 1 failed operation, got %d", metrics.FailedOps)
		}
	})

	t.Run("RecordOperation - remove operation", func(t *testing.T) {
		metrics := &PackageMetrics{}
		op := &PackageOperation{
			Package:   "nginx",
			Operation: "remove",
			Success:   true,
			Changed:   true,
			Duration:  80 * time.Millisecond,
		}

		metrics.RecordOperation(op)

		if metrics.PackagesRemoved != 1 {
			t.Errorf("Expected 1 package removed, got %d", metrics.PackagesRemoved)
		}
	})

	t.Run("RecordOperation - update operation", func(t *testing.T) {
		metrics := &PackageMetrics{}
		op := &PackageOperation{
			Package:   "nginx",
			Operation: "update",
			Success:   true,
			Changed:   true,
			Duration:  120 * time.Millisecond,
		}

		metrics.RecordOperation(op)

		if metrics.PackagesUpdated != 1 {
			t.Errorf("Expected 1 package updated, got %d", metrics.PackagesUpdated)
		}
	})

	t.Run("RecordOperation - multiple operations", func(t *testing.T) {
		metrics := &PackageMetrics{}

		ops := []*PackageOperation{
			{Package: "nginx", Operation: "install", Success: true, Changed: true, Duration: 100 * time.Millisecond},
			{Package: "apache", Operation: "install", Success: true, Changed: true, Duration: 150 * time.Millisecond},
			{Package: "mysql", Operation: "remove", Success: true, Changed: true, Duration: 80 * time.Millisecond},
			{Package: "redis", Operation: "install", Success: false, Changed: false, Duration: 50 * time.Millisecond},
		}

		for _, op := range ops {
			metrics.RecordOperation(op)
		}

		if metrics.TotalOperations != 4 {
			t.Errorf("Expected 4 total operations, got %d", metrics.TotalOperations)
		}
		if metrics.SuccessfulOps != 3 {
			t.Errorf("Expected 3 successful operations, got %d", metrics.SuccessfulOps)
		}
		if metrics.FailedOps != 1 {
			t.Errorf("Expected 1 failed operation, got %d", metrics.FailedOps)
		}
		if metrics.PackagesInstalled != 2 {
			t.Errorf("Expected 2 packages installed, got %d", metrics.PackagesInstalled)
		}
		if metrics.PackagesRemoved != 1 {
			t.Errorf("Expected 1 package removed, got %d", metrics.PackagesRemoved)
		}

		expectedDuration := 380 * time.Millisecond
		if metrics.TotalDuration != expectedDuration {
			t.Errorf("Expected total duration %v, got %v", expectedDuration, metrics.TotalDuration)
		}
	})

	t.Run("GetMetrics", func(t *testing.T) {
		metrics := &PackageMetrics{}

		// Record some operations
		metrics.RecordOperation(&PackageOperation{
			Package: "nginx", Operation: "install", Success: true, Changed: true, Duration: 100 * time.Millisecond,
		})
		metrics.RecordOperation(&PackageOperation{
			Package: "apache", Operation: "install", Success: true, Changed: true, Duration: 200 * time.Millisecond,
		})

		result := metrics.GetMetrics()

		if result.TotalOperations != 2 {
			t.Errorf("Expected 2 total operations, got %d", result.TotalOperations)
		}
		if result.SuccessfulOps != 2 {
			t.Errorf("Expected 2 successful operations, got %d", result.SuccessfulOps)
		}

		expectedAvg := 150 * time.Millisecond
		if result.AverageDuration != expectedAvg {
			t.Errorf("Expected average duration %v, got %v", expectedAvg, result.AverageDuration)
		}
	})
}
