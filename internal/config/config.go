package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for Onigirazu
type Config struct {
	// Execution settings
	MaxConcurrency int           `yaml:"max_concurrency" json:"max_concurrency"`
	DefaultTimeout time.Duration `yaml:"default_timeout" json:"default_timeout"`
	RetryAttempts  int           `yaml:"retry_attempts" json:"retry_attempts"`
	RetryDelay     time.Duration `yaml:"retry_delay" json:"retry_delay"`

	// File paths
	StateFile     string `yaml:"state_file" json:"state_file"`
	ConfigFile    string `yaml:"config_file" json:"config_file"`

	// Logging
	LogLevel  string `yaml:"log_level" json:"log_level"`
	LogFormat string `yaml:"log_format" json:"log_format"`

	// Security
	AllowShellCommands bool     `yaml:"allow_shell_commands" json:"allow_shell_commands"`
	BlockedCommands    []string `yaml:"blocked_commands" json:"blocked_commands"`

	// Performance
	EnableCaching    bool          `yaml:"enable_caching" json:"enable_caching"`
	CacheTTL         time.Duration `yaml:"cache_ttl" json:"cache_ttl"`
	EnableChecksum   bool          `yaml:"enable_checksum" json:"enable_checksum"`
	EnableParallel   bool          `yaml:"enable_parallel" json:"enable_parallel"`
	ParallelStrategy string        `yaml:"parallel_strategy" json:"parallel_strategy"` // "linear", "free"

	// Execution modes
	DryRun    bool `yaml:"dry_run" json:"dry_run"`
	CheckMode bool `yaml:"check_mode" json:"check_mode"`
	Verbose   bool `yaml:"verbose" json:"verbose"`
	ShowDiff  bool `yaml:"show_diff" json:"show_diff"`

	// UI/UX
	ColorOutput    bool   `yaml:"color_output" json:"color_output"`
	ProgressBar    bool   `yaml:"progress_bar" json:"progress_bar"`
	InteractiveMode bool  `yaml:"interactive_mode" json:"interactive_mode"`
	OutputFormat   string `yaml:"output_format" json:"output_format"` // "text", "json", "yaml"

	// Monitoring
	EnableMetrics   bool   `yaml:"enable_metrics" json:"enable_metrics"`
	MetricsPort     int    `yaml:"metrics_port" json:"metrics_port"`
	MetricsPath     string `yaml:"metrics_path" json:"metrics_path"`
	EnableProfiling bool   `yaml:"enable_profiling" json:"enable_profiling"`

	// SSH/Connection
	SSHTimeout       time.Duration `yaml:"ssh_timeout" json:"ssh_timeout"`
	SSHKeepAlive     time.Duration `yaml:"ssh_keepalive" json:"ssh_keepalive"`
	SSHMaxSessions   int           `yaml:"ssh_max_sessions" json:"ssh_max_sessions"`
	ConnectionReuse  bool          `yaml:"connection_reuse" json:"connection_reuse"`

	// Vault integration
	VaultEnabled bool   `yaml:"vault_enabled" json:"vault_enabled"`
	VaultAddress string `yaml:"vault_address" json:"vault_address"`
	VaultToken   string `yaml:"vault_token" json:"vault_token"`
}

// NewConfig creates a new config instance with defaults
func NewConfig() *Config {
	return DefaultConfig()
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		MaxConcurrency:     getEnvInt("ONIGIRAZU_MAX_CONCURRENCY", 10),
		DefaultTimeout:     getEnvDuration("ONIGIRAZU_TIMEOUT", 30*time.Second),
		RetryAttempts:      getEnvInt("ONIGIRAZU_RETRY_ATTEMPTS", 3),
		RetryDelay:         getEnvDuration("ONIGIRAZU_RETRY_DELAY", 5*time.Second),
		StateFile:          getEnvString("ONIGIRAZU_STATE_FILE", ".onigirazu-state"),
		ConfigFile:         getEnvString("ONIGIRAZU_CONFIG_FILE", "onigirazu.yml"),
		LogLevel:           getEnvString("ONIGIRAZU_LOG_LEVEL", "info"),
		LogFormat:          getEnvString("ONIGIRAZU_LOG_FORMAT", "text"),
		AllowShellCommands: getEnvBool("ONIGIRAZU_ALLOW_SHELL", true),
		BlockedCommands:    []string{"rm -rf", "format", "mkfs", "dd if=", ":(){ :|:& };:"},
		EnableCaching:      getEnvBool("ONIGIRAZU_ENABLE_CACHE", true),
		CacheTTL:           getEnvDuration("ONIGIRAZU_CACHE_TTL", 5*time.Minute),
		EnableChecksum:     getEnvBool("ONIGIRAZU_ENABLE_CHECKSUM", true),
		EnableParallel:     getEnvBool("ONIGIRAZU_ENABLE_PARALLEL", false),
		ParallelStrategy:   getEnvString("ONIGIRAZU_PARALLEL_STRATEGY", "linear"),
		DryRun:             getEnvBool("ONIGIRAZU_DRY_RUN", false),
		CheckMode:          getEnvBool("ONIGIRAZU_CHECK_MODE", false),
		Verbose:            getEnvBool("ONIGIRAZU_VERBOSE", false),
		ShowDiff:           getEnvBool("ONIGIRAZU_SHOW_DIFF", false),
		ColorOutput:        getEnvBool("ONIGIRAZU_COLOR_OUTPUT", true),
		ProgressBar:        getEnvBool("ONIGIRAZU_PROGRESS_BAR", true),
		InteractiveMode:    getEnvBool("ONIGIRAZU_INTERACTIVE", false),
		OutputFormat:       getEnvString("ONIGIRAZU_OUTPUT_FORMAT", "text"),
		EnableMetrics:      getEnvBool("ONIGIRAZU_ENABLE_METRICS", false),
		MetricsPort:        getEnvInt("ONIGIRAZU_METRICS_PORT", 9090),
		MetricsPath:        getEnvString("ONIGIRAZU_METRICS_PATH", "/metrics"),
		EnableProfiling:    getEnvBool("ONIGIRAZU_ENABLE_PROFILING", false),
		SSHTimeout:         getEnvDuration("ONIGIRAZU_SSH_TIMEOUT", 30*time.Second),
		SSHKeepAlive:       getEnvDuration("ONIGIRAZU_SSH_KEEPALIVE", 60*time.Second),
		SSHMaxSessions:     getEnvInt("ONIGIRAZU_SSH_MAX_SESSIONS", 10),
		ConnectionReuse:    getEnvBool("ONIGIRAZU_CONNECTION_REUSE", true),
		VaultEnabled:       getEnvBool("ONIGIRAZU_VAULT_ENABLED", false),
		VaultAddress:       getEnvString("ONIGIRAZU_VAULT_ADDRESS", ""),
		VaultToken:         getEnvString("ONIGIRAZU_VAULT_TOKEN", ""),
	}
}

// Helper functions to read environment variables with defaults
func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// LoadConfig loads configuration from file or returns default config
func LoadConfig(path string) (*Config, error) {
	// If no path provided or file doesn't exist, return default config
	if path == "" {
		return DefaultConfig(), nil
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := DefaultConfig()
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.MaxConcurrency <= 0 {
		c.MaxConcurrency = 1
	}
	if c.DefaultTimeout <= 0 {
		c.DefaultTimeout = 30 * time.Second
	}
	if c.RetryAttempts < 0 {
		c.RetryAttempts = 0
	}
	if c.RetryDelay < 0 {
		c.RetryDelay = 0
	}
	return nil
}

// Interface methods for interfaces.Config
func (c *Config) GetMaxConcurrency() int {
	return c.MaxConcurrency
}

func (c *Config) GetDefaultTimeout() time.Duration {
	return c.DefaultTimeout
}

func (c *Config) GetRetryAttempts() int {
	return c.RetryAttempts
}

func (c *Config) GetRetryDelay() time.Duration {
	return c.RetryDelay
}

func (c *Config) GetStateFile() string {
	return c.StateFile
}

func (c *Config) GetLogLevel() string {
	return c.LogLevel
}

func (c *Config) GetLogFormat() string {
	return c.LogFormat
}

func (c *Config) IsShellCommandsAllowed() bool {
	return c.AllowShellCommands
}

func (c *Config) GetBlockedCommands() []string {
	return c.BlockedCommands
}

func (c *Config) IsCachingEnabled() bool {
	return c.EnableCaching
}

func (c *Config) GetCacheTTL() time.Duration {
	return c.CacheTTL
}

func (c *Config) IsChecksumEnabled() bool {
	return c.EnableChecksum
}

func (c *Config) IsDryRun() bool {
	return c.DryRun
}

func (c *Config) IsCheckMode() bool {
	return c.CheckMode
}

func (c *Config) GetDryRun() bool {
	return c.DryRun
}

func (c *Config) GetCheckMode() bool {
	return c.CheckMode
}

// New getter methods for extended configuration
func (c *Config) IsParallelEnabled() bool {
	return c.EnableParallel
}

func (c *Config) GetParallelStrategy() string {
	return c.ParallelStrategy
}

func (c *Config) IsVerbose() bool {
	return c.Verbose
}

func (c *Config) ShouldShowDiff() bool {
	return c.ShowDiff
}

func (c *Config) IsColorOutputEnabled() bool {
	return c.ColorOutput
}

func (c *Config) IsProgressBarEnabled() bool {
	return c.ProgressBar
}

func (c *Config) IsInteractiveModeEnabled() bool {
	return c.InteractiveMode
}

func (c *Config) GetOutputFormat() string {
	return c.OutputFormat
}

func (c *Config) IsMetricsEnabled() bool {
	return c.EnableMetrics
}

func (c *Config) GetMetricsPort() int {
	return c.MetricsPort
}

func (c *Config) GetMetricsPath() string {
	return c.MetricsPath
}

func (c *Config) IsProfilingEnabled() bool {
	return c.EnableProfiling
}

func (c *Config) GetSSHTimeout() time.Duration {
	return c.SSHTimeout
}

func (c *Config) GetSSHKeepAlive() time.Duration {
	return c.SSHKeepAlive
}

func (c *Config) GetSSHMaxSessions() int {
	return c.SSHMaxSessions
}

func (c *Config) IsConnectionReuseEnabled() bool {
	return c.ConnectionReuse
}

func (c *Config) IsVaultEnabled() bool {
	return c.VaultEnabled
}

func (c *Config) GetVaultAddress() string {
	return c.VaultAddress
}

func (c *Config) GetVaultToken() string {
	return c.VaultToken
}
