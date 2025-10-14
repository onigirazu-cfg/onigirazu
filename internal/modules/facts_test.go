package modules

import (
	"context"
	"os"
	"runtime"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestNewFactsModule(t *testing.T) {
	module := NewFactsModule()
	if module == nil {
		t.Fatal("Expected non-nil module")
	}

	if module.GetName() != "facts" {
		t.Errorf("Expected name 'facts', got '%s'", module.GetName())
	}

	if module.GetDescription() == "" {
		t.Error("Expected non-empty description")
	}
}

func TestFactsModule_Execute(t *testing.T) {
	module := NewFactsModule()
	host := types.Host{
		Name:    "localhost",
		Address: "127.0.0.1",
	}

	ctx := context.Background()
	args := make(map[string]interface{})

	result, err := module.Execute(ctx, host, args)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected successful result")
	}

	if result.Changed {
		t.Error("Facts gathering should not change anything")
	}

	// Check that facts were gathered
	facts, ok := result.Output["onigirazu_facts"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected onigirazu_facts in output")
	}

	// Verify basic facts
	// Note: os_family is set in Execute() but may be overridden by gatherPlatformFacts()
	if facts["os_family"] == nil {
		t.Error("Expected os_family to be set")
	}

	if facts["architecture"] != runtime.GOARCH {
		t.Errorf("Expected architecture=%s, got %v", runtime.GOARCH, facts["architecture"])
	}

	if facts["num_cpus"] != runtime.NumCPU() {
		t.Errorf("Expected num_cpus=%d, got %v", runtime.NumCPU(), facts["num_cpus"])
	}

	// Verify process info
	if facts["pid"] != os.Getpid() {
		t.Errorf("Expected pid=%d, got %v", os.Getpid(), facts["pid"])
	}

	if facts["ppid"] != os.Getppid() {
		t.Errorf("Expected ppid=%d, got %v", os.Getppid(), facts["ppid"])
	}

	// Verify environment variables exist
	env, ok := facts["environment"].(map[string]string)
	if !ok {
		t.Error("Expected environment to be map[string]string")
	}

	// Environment should have some variables
	if len(env) == 0 {
		t.Error("Expected some environment variables")
	}

	// Verify memory stats exist
	memory, ok := facts["memory"].(map[string]interface{})
	if !ok {
		t.Error("Expected memory stats")
	}

	if memory["alloc"] == nil {
		t.Error("Expected memory alloc stat")
	}

	// Verify goroutines count
	if facts["goroutines"] == nil {
		t.Error("Expected goroutines count")
	}

	// Verify network facts
	if facts["network"] == nil {
		t.Error("Expected network facts")
	}

	// Verify filesystem facts
	if facts["filesystem"] == nil {
		t.Error("Expected filesystem facts")
	}

	// Verify timestamps
	if facts["fact_timestamp"] == nil {
		t.Error("Expected fact_timestamp")
	}

	if facts["fact_time"] == nil {
		t.Error("Expected fact_time")
	}
}

func TestFactsModule_Execute_WithCustomFacts(t *testing.T) {
	module := NewFactsModule()
	host := types.Host{
		Name:    "localhost",
		Address: "127.0.0.1",
	}

	ctx := context.Background()
	args := map[string]interface{}{
		"custom_facts": map[string]interface{}{
			"app_name":    "test-app",
			"app_version": "1.0.0",
			"environment": "testing",
		},
	}

	result, err := module.Execute(ctx, host, args)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	facts, ok := result.Output["onigirazu_facts"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected onigirazu_facts in output")
	}

	custom, ok := facts["custom"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected custom facts")
	}

	if custom["app_name"] != "test-app" {
		t.Errorf("Expected app_name='test-app', got %v", custom["app_name"])
	}

	if custom["app_version"] != "1.0.0" {
		t.Errorf("Expected app_version='1.0.0', got %v", custom["app_version"])
	}
}

func TestFactsModule_Validate(t *testing.T) {
	module := NewFactsModule()

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name:    "no args",
			args:    map[string]interface{}{},
			wantErr: false,
		},
		{
			name: "valid custom_facts",
			args: map[string]interface{}{
				"custom_facts": map[string]interface{}{
					"key": "value",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid custom_facts type",
			args: map[string]interface{}{
				"custom_facts": "not a map",
			},
			wantErr: true,
		},
		{
			name: "invalid custom_facts type - array",
			args: map[string]interface{}{
				"custom_facts": []string{"a", "b"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := module.Validate(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFactsModule_IsSensitiveEnvVar(t *testing.T) {
	module := NewFactsModule()

	tests := []struct {
		name      string
		key       string
		sensitive bool
	}{
		{"password", "DB_PASSWORD", true},
		{"passwd", "USER_PASSWD", true},
		{"secret", "API_SECRET", true},
		{"token", "AUTH_TOKEN", true},
		{"key", "API_KEY", true},
		{"private", "PRIVATE_KEY", true},
		{"ssh_key", "SSH_KEY", true},
		{"credential", "AWS_CREDENTIAL", true},
		{"cert", "SSL_CERT", true},
		{"auth", "OAUTH_TOKEN", true},
		{"normal var", "HOME", false},
		{"path", "PATH", false},
		{"user", "USER", false},
		{"shell", "SHELL", false},
		{"lang", "LANG", false},
		{"lowercase password", "db_password", true},
		{"mixed case", "Api_Key", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := module.isSensitiveEnvVar(tt.key)
			if result != tt.sensitive {
				t.Errorf("isSensitiveEnvVar(%s) = %v, want %v", tt.key, result, tt.sensitive)
			}
		})
	}
}

func TestFactsModule_GatherPlatformFacts(t *testing.T) {
	module := NewFactsModule()
	facts := module.gatherPlatformFacts()

	if facts == nil {
		t.Fatal("Expected non-nil facts")
	}

	if facts["platform"] == nil {
		t.Error("Expected platform fact")
	}

	if facts["os_family"] == nil {
		t.Error("Expected os_family fact")
	}

	// Platform-specific checks
	switch runtime.GOOS {
	case "darwin":
		if facts["platform"] != "Darwin" {
			t.Errorf("Expected platform=Darwin on macOS, got %v", facts["platform"])
		}
		if facts["os_version"] == nil {
			t.Error("Expected os_version on macOS")
		}

	case "linux":
		if facts["platform"] != "Linux" {
			t.Errorf("Expected platform=Linux, got %v", facts["platform"])
		}
		if facts["distribution"] == nil {
			t.Error("Expected distribution on Linux")
		}

	case "windows":
		if facts["platform"] != "Win32NT" {
			t.Errorf("Expected platform=Win32NT on Windows, got %v", facts["platform"])
		}
	}
}

func TestFactsModule_GatherNetworkFacts(t *testing.T) {
	module := NewFactsModule()
	facts := module.gatherNetworkFacts()

	if facts == nil {
		t.Fatal("Expected non-nil facts")
	}

	if facts["interfaces"] == nil {
		t.Error("Expected interfaces")
	}

	if facts["default_ipv4"] == nil {
		t.Error("Expected default_ipv4")
	}
}

func TestFactsModule_GatherFileSystemFacts(t *testing.T) {
	module := NewFactsModule()
	facts := module.gatherFileSystemFacts()

	if facts == nil {
		t.Fatal("Expected non-nil facts")
	}

	if facts["mounts"] == nil {
		t.Error("Expected mounts")
	}
}

func TestFactsModule_GetSystemLoad(t *testing.T) {
	module := NewFactsModule()
	load := module.GetSystemLoad()

	if load == nil {
		t.Fatal("Expected non-nil load")
	}

	if load["memory_usage"] == nil {
		t.Error("Expected memory_usage")
	}

	if load["goroutines"] == nil {
		t.Error("Expected goroutines")
	}

	if load["gc_runs"] == nil {
		t.Error("Expected gc_runs")
	}
}

func TestFactsModule_GetProcessInfo(t *testing.T) {
	module := NewFactsModule()
	info := module.GetProcessInfo()

	if info == nil {
		t.Fatal("Expected non-nil info")
	}

	if info["pid"] != os.Getpid() {
		t.Errorf("Expected pid=%d, got %v", os.Getpid(), info["pid"])
	}

	if info["ppid"] != os.Getppid() {
		t.Errorf("Expected ppid=%d, got %v", os.Getppid(), info["ppid"])
	}

	if info["uid"] != os.Getuid() {
		t.Errorf("Expected uid=%d, got %v", os.Getuid(), info["uid"])
	}

	if info["gid"] != os.Getgid() {
		t.Errorf("Expected gid=%d, got %v", os.Getgid(), info["gid"])
	}

	if info["working_directory"] == nil {
		t.Error("Expected working_directory")
	}
}

func TestFactsModule_GetRuntimeInfo(t *testing.T) {
	module := NewFactsModule()
	info := module.GetRuntimeInfo()

	if info == nil {
		t.Fatal("Expected non-nil info")
	}

	if info["go_version"] != runtime.Version() {
		t.Errorf("Expected go_version=%s, got %v", runtime.Version(), info["go_version"])
	}

	if info["go_os"] != runtime.GOOS {
		t.Errorf("Expected go_os=%s, got %v", runtime.GOOS, info["go_os"])
	}

	if info["go_arch"] != runtime.GOARCH {
		t.Errorf("Expected go_arch=%s, got %v", runtime.GOARCH, info["go_arch"])
	}

	if info["num_cpu"] != runtime.NumCPU() {
		t.Errorf("Expected num_cpu=%d, got %v", runtime.NumCPU(), info["num_cpu"])
	}

	if info["num_goroutine"] == nil {
		t.Error("Expected num_goroutine")
	}
}

func TestFactsModule_GetEnvironmentInfo(t *testing.T) {
	module := NewFactsModule()
	info := module.GetEnvironmentInfo()

	if info == nil {
		t.Fatal("Expected non-nil info")
	}

	vars, ok := info["variables"].(map[string]string)
	if !ok {
		t.Fatal("Expected variables to be map[string]string")
	}

	// Should have some environment variables
	if len(vars) == 0 {
		t.Error("Expected some environment variables")
	}

	// Verify sensitive variables are filtered
	for key := range vars {
		if module.isSensitiveEnvVar(key) {
			t.Errorf("Sensitive variable %s should be filtered", key)
		}
	}

	if info["hostname"] == nil {
		t.Error("Expected hostname")
	}
}

func TestFactsModule_GetMemoryInfo(t *testing.T) {
	module := NewFactsModule()
	info := module.GetMemoryInfo()

	if info == nil {
		t.Fatal("Expected non-nil info")
	}

	requiredFields := []string{
		"alloc", "total_alloc", "sys", "heap_alloc", "heap_sys",
		"heap_idle", "heap_inuse", "heap_released", "heap_objects",
		"stack_inuse", "stack_sys", "gc_runs", "gc_pause_total",
		"gc_pause_recent", "mallocs", "frees", "lookups",
	}

	for _, field := range requiredFields {
		if info[field] == nil {
			t.Errorf("Expected %s in memory info", field)
		}
	}
}

func TestFactsModule_GetMacOSVersion(t *testing.T) {
	module := NewFactsModule()
	version := module.getMacOSVersion()

	// Should return a version string (even if placeholder)
	if version == "" {
		t.Error("Expected non-empty macOS version")
	}
}

func TestFactsModule_GetLinuxDistribution(t *testing.T) {
	module := NewFactsModule()
	distro := module.getLinuxDistribution()

	// Should return a distribution name (even if placeholder)
	if distro == "" {
		t.Error("Expected non-empty Linux distribution")
	}
}
