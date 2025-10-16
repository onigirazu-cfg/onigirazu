package config

import (
	"os"
	"testing"
)

func TestDefaultInsecureIgnoreHostKey(t *testing.T) {
	// Test default value
	cfg := DefaultConfig()
	if cfg.DefaultInsecureIgnoreHostKey != false {
		t.Errorf("Expected DefaultInsecureIgnoreHostKey to be false by default, got %v", cfg.DefaultInsecureIgnoreHostKey)
	}

	// Test environment variable
	os.Setenv("ONIGIRAZU_DEFAULT_INSECURE_IGNORE_HOST_KEY", "true")
	defer os.Unsetenv("ONIGIRAZU_DEFAULT_INSECURE_IGNORE_HOST_KEY")

	cfg = DefaultConfig()
	if cfg.DefaultInsecureIgnoreHostKey != true {
		t.Errorf("Expected DefaultInsecureIgnoreHostKey to be true from env var, got %v", cfg.DefaultInsecureIgnoreHostKey)
	}

	// Test getter method
	if !cfg.GetDefaultInsecureIgnoreHostKey() {
		t.Error("GetDefaultInsecureIgnoreHostKey() should return true")
	}
}
