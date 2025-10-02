package config // import "github.com/onigirazu-cfg/onigirazu/internal/config"


TYPES

type Config struct {
	// Execution settings
	MaxConcurrency int           `yaml:"max_concurrency" json:"max_concurrency"`
	DefaultTimeout time.Duration `yaml:"default_timeout" json:"default_timeout"`
	RetryAttempts  int           `yaml:"retry_attempts" json:"retry_attempts"`
	RetryDelay     time.Duration `yaml:"retry_delay" json:"retry_delay"`

	// File paths
	StateFile  string `yaml:"state_file" json:"state_file"`
	ConfigFile string `yaml:"config_file" json:"config_file"`

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
	ColorOutput     bool   `yaml:"color_output" json:"color_output"`
	ProgressBar     bool   `yaml:"progress_bar" json:"progress_bar"`
	InteractiveMode bool   `yaml:"interactive_mode" json:"interactive_mode"`
	OutputFormat    string `yaml:"output_format" json:"output_format"` // "text", "json", "yaml"

	// Monitoring
	EnableMetrics   bool   `yaml:"enable_metrics" json:"enable_metrics"`
	MetricsPort     int    `yaml:"metrics_port" json:"metrics_port"`
	MetricsPath     string `yaml:"metrics_path" json:"metrics_path"`
	EnableProfiling bool   `yaml:"enable_profiling" json:"enable_profiling"`

	// SSH/Connection
	SSHTimeout      time.Duration `yaml:"ssh_timeout" json:"ssh_timeout"`
	SSHKeepAlive    time.Duration `yaml:"ssh_keepalive" json:"ssh_keepalive"`
	SSHMaxSessions  int           `yaml:"ssh_max_sessions" json:"ssh_max_sessions"`
	ConnectionReuse bool          `yaml:"connection_reuse" json:"connection_reuse"`

	// Vault integration
	VaultEnabled bool   `yaml:"vault_enabled" json:"vault_enabled"`
	VaultAddress string `yaml:"vault_address" json:"vault_address"`
	VaultToken   string `yaml:"vault_token" json:"vault_token"`
}
    Config holds all configuration for Onigirazu

func DefaultConfig() *Config
    DefaultConfig returns a configuration with sensible defaults

func LoadConfig(path string) (*Config, error)
    LoadConfig loads configuration from file or returns default config

func NewConfig() *Config
    NewConfig creates a new config instance with defaults

func (c *Config) GetBlockedCommands() []string

func (c *Config) GetCacheTTL() time.Duration

func (c *Config) GetCheckMode() bool

func (c *Config) GetDefaultTimeout() time.Duration

func (c *Config) GetDryRun() bool

func (c *Config) GetLogFormat() string

func (c *Config) GetLogLevel() string

func (c *Config) GetMaxConcurrency() int
    Interface methods for interfaces.Config

func (c *Config) GetMetricsPath() string

func (c *Config) GetMetricsPort() int

func (c *Config) GetOutputFormat() string

func (c *Config) GetParallelStrategy() string

func (c *Config) GetRetryAttempts() int

func (c *Config) GetRetryDelay() time.Duration

func (c *Config) GetSSHKeepAlive() time.Duration

func (c *Config) GetSSHMaxSessions() int

func (c *Config) GetSSHTimeout() time.Duration

func (c *Config) GetStateFile() string

func (c *Config) GetVaultAddress() string

func (c *Config) GetVaultToken() string

func (c *Config) IsCachingEnabled() bool

func (c *Config) IsCheckMode() bool

func (c *Config) IsChecksumEnabled() bool

func (c *Config) IsColorOutputEnabled() bool

func (c *Config) IsConnectionReuseEnabled() bool

func (c *Config) IsDryRun() bool

func (c *Config) IsInteractiveModeEnabled() bool

func (c *Config) IsMetricsEnabled() bool

func (c *Config) IsParallelEnabled() bool
    New getter methods for extended configuration

func (c *Config) IsProfilingEnabled() bool

func (c *Config) IsProgressBarEnabled() bool

func (c *Config) IsShellCommandsAllowed() bool

func (c *Config) IsVaultEnabled() bool

func (c *Config) IsVerbose() bool

func (c *Config) ShouldShowDiff() bool

func (c *Config) Validate() error
    Validate checks if the configuration is valid

