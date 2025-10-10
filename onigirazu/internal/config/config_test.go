package config

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("NewConfig() returned nil")
	}

	// Test default values
	if config.GetMaxConcurrency() != 10 {
		t.Errorf("Expected MaxConcurrency to be 10, got %d", config.GetMaxConcurrency())
	}

	if config.GetDefaultTimeout() != 30*time.Second {
		t.Errorf("Expected DefaultTimeout to be 30s, got %v", config.GetDefaultTimeout())
	}

	if config.GetRetryAttempts() != 3 {
		t.Errorf("Expected RetryAttempts to be 3, got %d", config.GetRetryAttempts())
	}

	if config.GetRetryDelay() != 5*time.Second {
		t.Errorf("Expected RetryDelay to be 5s, got %v", config.GetRetryDelay())
	}

	if config.GetLogLevel() != "info" {
		t.Errorf("Expected LogLevel to be 'info', got %s", config.GetLogLevel())
	}

	if config.GetLogFormat() != "text" {
		t.Errorf("Expected LogFormat to be 'text', got %s", config.GetLogFormat())
	}

	if !config.IsShellCommandsAllowed() {
		t.Error("Expected ShellCommandsAllowed to be true")
	}

	if !config.IsCachingEnabled() {
		t.Error("Expected CachingEnabled to be true")
	}

	if config.GetCacheTTL() != 5*time.Minute {
		t.Errorf("Expected CacheTTL to be 5m, got %v", config.GetCacheTTL())
	}

	if !config.IsChecksumEnabled() {
		t.Error("Expected ChecksumEnabled to be true")
	}
}

func TestConfigSetters(t *testing.T) {
	config := DefaultConfig()

	// Test setting values
	config.MaxConcurrency = 20
	if config.GetMaxConcurrency() != 20 {
		t.Errorf("Expected MaxConcurrency to be 20, got %d", config.GetMaxConcurrency())
	}

	config.DefaultTimeout = 60 * time.Second
	if config.GetDefaultTimeout() != 60*time.Second {
		t.Errorf("Expected DefaultTimeout to be 60s, got %v", config.GetDefaultTimeout())
	}

	config.LogLevel = "debug"
	if config.GetLogLevel() != "debug" {
		t.Errorf("Expected LogLevel to be 'debug', got %s", config.GetLogLevel())
	}

	config.AllowShellCommands = false
	if config.IsShellCommandsAllowed() {
		t.Error("Expected AllowShellCommands to be false")
	}
}

func TestBlockedCommands(t *testing.T) {
	config := DefaultConfig()

	blockedCommands := config.GetBlockedCommands()
	expectedCommands := []string{"rm -rf", "format", "mkfs", "dd if=", ":(){ :|:& };:"}

	if len(blockedCommands) != len(expectedCommands) {
		t.Errorf("Expected %d blocked commands, got %d", len(expectedCommands), len(blockedCommands))
	}

	for i, expected := range expectedCommands {
		if i >= len(blockedCommands) || blockedCommands[i] != expected {
			t.Errorf("Expected blocked command %d to be '%s', got '%s'", i, expected, blockedCommands[i])
		}
	}
}
