package config

import (
	"fmt"
	"time"
)

type SecurityConfig struct {
	SSH SSHSecurityConfig `yaml:"ssh"`
}

type SSHSecurityConfig struct {
	StrictHostKeyChecking bool          `yaml:"strict_host_key_checking" default:"true"`
	KnownHostsFile       string        `yaml:"known_hosts_file"`
	ConnectionTimeout    time.Duration `yaml:"connection_timeout" default:"30s"`
	CommandTimeout       time.Duration `yaml:"command_timeout" default:"300s"`
	MaxRetries          int           `yaml:"max_retries" default:"3"`
	AllowedCiphers      []string      `yaml:"allowed_ciphers"`
	AllowedMACs         []string      `yaml:"allowed_macs"`
	AllowedKexAlgos     []string      `yaml:"allowed_kex_algos"`
}

func DefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		SSH: SSHSecurityConfig{
			StrictHostKeyChecking: true,
			ConnectionTimeout:    30 * time.Second,
			CommandTimeout:       300 * time.Second,
			MaxRetries:          3,
			AllowedCiphers: []string{
				"aes128-ctr",
				"aes192-ctr",
				"aes256-ctr",
				"aes128-gcm@openssh.com",
				"aes256-gcm@openssh.com",
			},
			AllowedMACs: []string{
				"hmac-sha2-256-etm@openssh.com",
				"hmac-sha2-512-etm@openssh.com",
				"hmac-sha2-256",
				"hmac-sha2-512",
			},
			AllowedKexAlgos: []string{
				"curve25519-sha256",
				"curve25519-sha256@libssh.org",
				"ecdh-sha2-nistp256",
				"ecdh-sha2-nistp384",
				"ecdh-sha2-nistp521",
				"diffie-hellman-group14-sha256",
				"diffie-hellman-group16-sha512",
			},
		},
	}
}

// ValidateSecurityConfig validates the security configuration
func ValidateSecurityConfig(config SecurityConfig) error {
	if config.SSH.ConnectionTimeout <= 0 {
		return NewValidationError("security", "", "connection timeout must be positive")
	}

	if config.SSH.CommandTimeout <= 0 {
		return NewValidationError("security", "", "command timeout must be positive")
	}

	if config.SSH.MaxRetries < 0 {
		return NewValidationError("security", "", "max retries cannot be negative")
	}

	return nil
}

// NewValidationError creates a validation error (placeholder - should use pkg/errors)
func NewValidationError(module, task, message string) error {
	return fmt.Errorf("validation error in %s: %s", module, message)
}
