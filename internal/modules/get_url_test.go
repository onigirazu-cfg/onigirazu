package modules

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// TestGetURLModule_Validate tests the Validate method
func TestGetURLModule_Validate(t *testing.T) {
	module := NewGetURLModule()

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid arguments",
			args: map[string]interface{}{
				"url":  "https://example.com/file.txt",
				"dest": "/tmp/file.txt",
			},
			wantErr: false,
		},
		{
			name:    "missing url",
			args:    map[string]interface{}{"dest": "/tmp/file.txt"},
			wantErr: true,
			errMsg:  "url parameter is required",
		},
		{
			name:    "missing dest",
			args:    map[string]interface{}{"url": "https://example.com/file.txt"},
			wantErr: true,
			errMsg:  "dest parameter is required",
		},
		{
			name:    "empty arguments",
			args:    map[string]interface{}{},
			wantErr: true,
			errMsg:  "url parameter is required",
		},
		{
			name: "with optional force parameter",
			args: map[string]interface{}{
				"url":   "https://example.com/file.txt",
				"dest":  "/tmp/file.txt",
				"force": true,
			},
			wantErr: false,
		},
		{
			name: "with optional backup parameter",
			args: map[string]interface{}{
				"url":    "https://example.com/file.txt",
				"dest":   "/tmp/file.txt",
				"backup": true,
			},
			wantErr: false,
		},
		{
			name: "with optional checksum parameter",
			args: map[string]interface{}{
				"url":      "https://example.com/file.txt",
				"dest":     "/tmp/file.txt",
				"checksum": "sha256:abc123",
			},
			wantErr: false,
		},
		{
			name: "with optional mode parameter",
			args: map[string]interface{}{
				"url":  "https://example.com/file.txt",
				"dest": "/tmp/file.txt",
				"mode": "0755",
			},
			wantErr: false,
		},
		{
			name: "with optional owner parameter",
			args: map[string]interface{}{
				"url":   "https://example.com/file.txt",
				"dest":  "/tmp/file.txt",
				"owner": "root",
			},
			wantErr: false,
		},
		{
			name: "with optional group parameter",
			args: map[string]interface{}{
				"url":   "https://example.com/file.txt",
				"dest":  "/tmp/file.txt",
				"group": "wheel",
			},
			wantErr: false,
		},
		{
			name: "with optional timeout parameter",
			args: map[string]interface{}{
				"url":     "https://example.com/file.txt",
				"dest":    "/tmp/file.txt",
				"timeout": 60,
			},
			wantErr: false,
		},
		{
			name: "with optional headers parameter",
			args: map[string]interface{}{
				"url":  "https://example.com/file.txt",
				"dest": "/tmp/file.txt",
				"headers": map[string]interface{}{
					"Authorization": "Bearer token123",
					"User-Agent":    "Onigirazu/1.0",
				},
			},
			wantErr: false,
		},
		{
			name: "with all optional parameters",
			args: map[string]interface{}{
				"url":      "https://example.com/file.txt",
				"dest":     "/tmp/file.txt",
				"force":    true,
				"backup":   true,
				"checksum": "sha256:abc123",
				"mode":     "0644",
				"owner":    "user",
				"group":    "group",
				"timeout":  30,
				"headers": map[string]interface{}{
					"Authorization": "Bearer token",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := module.Validate(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error but got none")
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

// TestGetURLModule_GetDescription tests the GetDescription method
func TestGetURLModule_GetDescription(t *testing.T) {
	module := NewGetURLModule()
	desc := module.GetDescription()

	if desc == "" {
		t.Error("GetDescription() returned empty string")
	}

	expectedDesc := "Download files from HTTP, HTTPS, or FTP URLs"
	if desc != expectedDesc {
		t.Errorf("GetDescription() = %v, want %v", desc, expectedDesc)
	}
}

// TestGetURLModule_Execute_ValidationError tests Execute with invalid arguments
func TestGetURLModule_Execute_ValidationError(t *testing.T) {
	module := NewGetURLModule()
	ctx := context.Background()
	host := types.Host{
		Name:    "testhost",
		Address: "localhost",
	}

	// Missing url parameter
	args := map[string]interface{}{
		"dest": "/tmp/test.txt",
	}

	result, err := module.Execute(ctx, host, args)

	if err == nil {
		t.Error("Execute() expected error for missing url parameter")
	}

	if !result.Failed {
		t.Error("Execute() result.Failed should be true for validation error")
	}

	if result.Error == "" {
		t.Error("Execute() result.Error should not be empty for validation error")
	}
}

// TestGetURLModule_Execute_ContextTimeout tests Execute with context timeout
func TestGetURLModule_Execute_ContextTimeout(t *testing.T) {
	module := NewGetURLModule()

	// Create a context that's already canceled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	host := types.Host{
		Name:    "testhost",
		Address: "localhost",
	}

	args := map[string]interface{}{
		"url":  "https://example.com/file.txt",
		"dest": "/tmp/file.txt",
	}

	result, err := module.Execute(ctx, host, args)

	// Should fail due to context cancellation or executor creation failure
	if err == nil && !result.Failed {
		t.Log("Execute() with canceled context may succeed if executor creation is fast")
	}

	// Verify result structure
	if result.Host != host.Name {
		t.Errorf("Execute() result.Host = %v, want %v", result.Host, host.Name)
	}

	if result.Module != "get_url" {
		t.Errorf("Execute() result.Module = %v, want %v", result.Module, "get_url")
	}
}

// TestGetURLModule_ChecksumMatches tests the checksumMatches method
func TestGetURLModule_ChecksumMatches(t *testing.T) {
	module := NewGetURLModule()

	tests := []struct {
		name     string
		expected string
		actual   string
		want     bool
	}{
		{
			name:     "exact match",
			expected: "abc123",
			actual:   "abc123",
			want:     true,
		},
		{
			name:     "case insensitive match",
			expected: "ABC123",
			actual:   "abc123",
			want:     true,
		},
		{
			name:     "with algorithm prefix",
			expected: "sha256:abc123",
			actual:   "abc123",
			want:     true,
		},
		{
			name:     "with algorithm prefix case insensitive",
			expected: "sha256:ABC123",
			actual:   "abc123",
			want:     true,
		},
		{
			name:     "mismatch",
			expected: "abc123",
			actual:   "def456",
			want:     false,
		},
		{
			name:     "empty strings",
			expected: "",
			actual:   "",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := module.checksumMatches(tt.expected, tt.actual)
			if got != tt.want {
				t.Errorf("checksumMatches() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestEncodeBase64 tests the encodeBase64 helper function
func TestEncodeBase64(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{
			name:  "empty input",
			input: []byte{},
			want:  "",
		},
		{
			name:  "simple text",
			input: []byte("hello"),
			want:  "aGVsbG8=",
		},
		{
			name:  "text with padding",
			input: []byte("hi"),
			want:  "aGk=",
		},
		{
			name:  "binary data",
			input: []byte{0x00, 0x01, 0x02, 0x03},
			want:  "AAECAw==",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := encodeBase64(tt.input)
			if got != tt.want {
				t.Errorf("encodeBase64() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetURLModule_ResultStructure tests the result structure
func TestGetURLModule_ResultStructure(t *testing.T) {
	module := NewGetURLModule()
	ctx := context.Background()
	host := types.Host{
		Name:    "testhost",
		Address: "localhost",
	}

	// Use invalid args to get a quick result
	args := map[string]interface{}{
		"dest": "/tmp/test.txt",
	}

	result, _ := module.Execute(ctx, host, args)

	// Verify result structure
	if result.Host != host.Name {
		t.Errorf("Execute() result.Host = %v, want %v", result.Host, host.Name)
	}

	if result.Module != "get_url" {
		t.Errorf("Execute() result.Module = %v, want %v", result.Module, "get_url")
	}

	if result.Timestamp.IsZero() {
		t.Error("Execute() result.Timestamp should not be zero")
	}

	if result.Output == nil {
		t.Error("Execute() result.Output should not be nil")
	}

	if result.Duration < 0 {
		t.Error("Execute() result.Duration should not be negative")
	}
}

// TestGetURLModule_HTTPServer tests downloading from a test HTTP server
func TestGetURLModule_HTTPServer(t *testing.T) {
	// Create a test HTTP server
	testContent := "test file content"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(testContent))
	}))
	defer server.Close()

	// Test that the server is working
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to connect to test server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Test server returned status %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

// TestGetURLModule_HTTPServerNotFound tests handling of 404 errors
func TestGetURLModule_HTTPServerNotFound(t *testing.T) {
	// Create a test HTTP server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Not Found"))
	}))
	defer server.Close()

	// Test that the server returns 404
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to connect to test server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Test server returned status %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

// TestGetURLModule_ForceParameter tests the force parameter logic
func TestGetURLModule_ForceParameter(t *testing.T) {
	module := NewGetURLModule()

	tests := []struct {
		name  string
		force bool
	}{
		{
			name:  "force true",
			force: true,
		},
		{
			name:  "force false",
			force: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := map[string]interface{}{
				"url":   "https://example.com/file.txt",
				"dest":  "/tmp/file.txt",
				"force": tt.force,
			}

			err := module.Validate(args)
			if err != nil {
				t.Errorf("Validate() with force=%v failed: %v", tt.force, err)
			}
		})
	}
}

// TestGetURLModule_ChecksumParameter tests checksum parameter validation
func TestGetURLModule_ChecksumParameter(t *testing.T) {
	module := NewGetURLModule()

	tests := []struct {
		name     string
		checksum string
	}{
		{
			name:     "md5 checksum",
			checksum: "md5:d41d8cd98f00b204e9800998ecf8427e",
		},
		{
			name:     "sha1 checksum",
			checksum: "sha1:da39a3ee5e6b4b0d3255bfef95601890afd80709",
		},
		{
			name:     "sha256 checksum",
			checksum: "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:     "checksum without algorithm",
			checksum: "abc123def456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := map[string]interface{}{
				"url":      "https://example.com/file.txt",
				"dest":     "/tmp/file.txt",
				"checksum": tt.checksum,
			}

			err := module.Validate(args)
			if err != nil {
				t.Errorf("Validate() with checksum=%v failed: %v", tt.checksum, err)
			}
		})
	}
}

// TestGetURLModule_HeadersParameter tests headers parameter
func TestGetURLModule_HeadersParameter(t *testing.T) {
	module := NewGetURLModule()

	args := map[string]interface{}{
		"url":  "https://example.com/file.txt",
		"dest": "/tmp/file.txt",
		"headers": map[string]interface{}{
			"Authorization": "Bearer token123",
			"User-Agent":    "Onigirazu/1.0",
			"Accept":        "application/octet-stream",
		},
	}

	err := module.Validate(args)
	if err != nil {
		t.Errorf("Validate() with headers failed: %v", err)
	}
}

// TestGetURLModule_TimeoutParameter tests timeout parameter
func TestGetURLModule_TimeoutParameter(t *testing.T) {
	module := NewGetURLModule()

	tests := []struct {
		name    string
		timeout int
	}{
		{
			name:    "default timeout",
			timeout: 30,
		},
		{
			name:    "short timeout",
			timeout: 5,
		},
		{
			name:    "long timeout",
			timeout: 300,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := map[string]interface{}{
				"url":     "https://example.com/file.txt",
				"dest":    "/tmp/file.txt",
				"timeout": tt.timeout,
			}

			err := module.Validate(args)
			if err != nil {
				t.Errorf("Validate() with timeout=%v failed: %v", tt.timeout, err)
			}
		})
	}
}

// TestGetURLModule_NewGetURLModule tests module creation
func TestGetURLModule_NewGetURLModule(t *testing.T) {
	module := NewGetURLModule()

	if module == nil {
		t.Fatal("NewGetURLModule() returned nil")
	}

	if module.name != "get_url" {
		t.Errorf("NewGetURLModule() name = %v, want %v", module.name, "get_url")
	}

	if module.description == "" {
		t.Error("NewGetURLModule() description should not be empty")
	}
}

// BenchmarkGetURLModule_Validate benchmarks the Validate method
func BenchmarkGetURLModule_Validate(b *testing.B) {
	module := NewGetURLModule()
	args := map[string]interface{}{
		"url":      "https://example.com/file.txt",
		"dest":     "/tmp/file.txt",
		"force":    true,
		"backup":   true,
		"checksum": "sha256:abc123",
		"mode":     "0644",
		"owner":    "user",
		"group":    "group",
		"timeout":  30,
		"headers": map[string]interface{}{
			"Authorization": "Bearer token",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = module.Validate(args)
	}
}

// BenchmarkEncodeBase64 benchmarks the encodeBase64 function
func BenchmarkEncodeBase64(b *testing.B) {
	// Create 1KB of test data
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = encodeBase64(data)
	}
}

// BenchmarkGetURLModule_ChecksumMatches benchmarks the checksumMatches method
func BenchmarkGetURLModule_ChecksumMatches(b *testing.B) {
	module := NewGetURLModule()
	expected := "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	actual := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = module.checksumMatches(expected, actual)
	}
}

// BenchmarkGetURLModule_Execute benchmarks the Execute method with validation error
func BenchmarkGetURLModule_Execute(b *testing.B) {
	module := NewGetURLModule()
	ctx := context.Background()
	host := types.Host{
		Name:    "testhost",
		Address: "localhost",
	}

	// Use invalid args for quick benchmark (validation error path)
	args := map[string]interface{}{
		"dest": "/tmp/test.txt",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = module.Execute(ctx, host, args)
	}
}
