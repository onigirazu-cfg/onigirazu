package modules // import "github.com/onigirazu-cfg/onigirazu/internal/modules"


TYPES

type AptManager struct{}
    AptManager implements package management for APT (Debian/Ubuntu)

func (a *AptManager) Clean() error

func (a *AptManager) GetInfo(name string) (PackageInfo, error)

func (a *AptManager) Install(name, version string) error

func (a *AptManager) IsInstalled(name string) (bool, string, error)

func (a *AptManager) ListInstalled() ([]PackageInfo, error)

func (a *AptManager) Remove(name string) error

func (a *AptManager) Search(query string) ([]PackageInfo, error)

func (a *AptManager) Update(name string) error

func (a *AptManager) UpdateAll() error

type BaseModule struct {
	// Has unexported fields.
}
    BaseModule represents base module structure

func NewBaseModule(name string) *BaseModule

func (m *BaseModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error)
    Execute performs basic module logic

func (m *BaseModule) GetName() string

func (m *BaseModule) Validate(args map[string]interface{}) error
    Validate validates argument correctness

type BrewManager struct{}
    BrewManager implements package management for Homebrew (macOS)

func (b *BrewManager) Clean() error

func (b *BrewManager) GetInfo(name string) (PackageInfo, error)

func (b *BrewManager) Install(name, version string) error

func (b *BrewManager) IsInstalled(name string) (bool, string, error)

func (b *BrewManager) ListInstalled() ([]PackageInfo, error)

func (b *BrewManager) Remove(name string) error

func (b *BrewManager) Search(query string) ([]PackageInfo, error)

func (b *BrewManager) Update(name string) error

func (b *BrewManager) UpdateAll() error

type ChocolateyManager struct{ GenericPackageManager }

type CommandModule struct {
	*BaseModule
}
    CommandModule executes shell commands

func NewCommandModule() *CommandModule

func (m *CommandModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error)

func (m *CommandModule) GetDescription() string

func (m *CommandModule) Validate(args map[string]interface{}) error

type ConfigAction string
    ConfigAction represents the action to perform

const (
	ConfigActionSet      ConfigAction = "set"
	ConfigActionGet      ConfigAction = "get"
	ConfigActionDelete   ConfigAction = "delete"
	ConfigActionMerge    ConfigAction = "merge"
	ConfigActionBackup   ConfigAction = "backup"
	ConfigActionRestore  ConfigAction = "restore"
	ConfigActionValidate ConfigAction = "validate"
)
type ConfigFormat string
    ConfigFormat represents supported configuration formats

const (
	FormatJSON ConfigFormat = "json"
	FormatYAML ConfigFormat = "yaml"
	FormatINI  ConfigFormat = "ini"
	FormatTOML ConfigFormat = "toml"
	FormatXML  ConfigFormat = "xml"
)
type ConfigModule struct {
	BaseModule
}
    ConfigModule implements configuration file management

func NewConfigModule() *ConfigModule
    NewConfigModule creates a new config module

func (m *ConfigModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error)
    Execute manages configuration files

func (m *ConfigModule) Validate(args map[string]interface{}) error
    Validate validates config module arguments

type CopyModule struct {
	BaseModule
}
    CopyModule handles file copying operations

func NewCopyModule() *CopyModule
    NewCopyModule creates a new copy module instance

func (m *CopyModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error)
    Execute runs the copy module

func (m *CopyModule) GetDescription() string
    GetDescription returns the module description

func (m *CopyModule) Validate(args map[string]interface{}) error
    Validate checks if the provided arguments are valid

type DnfManager struct{ YumManager }
    Placeholder implementations for other package managers

type FactsModule struct {
	BaseModule
}
    FactsModule implements system facts gathering

func NewFactsModule() *FactsModule
    NewFactsModule creates a new facts module

func (m *FactsModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error)
    Execute gathers system facts

func (m *FactsModule) GetEnvironmentInfo() map[string]interface{}
    GetEnvironmentInfo returns filtered environment information

func (m *FactsModule) GetMemoryInfo() map[string]interface{}
    GetMemoryInfo returns detailed memory information

func (m *FactsModule) GetProcessInfo() map[string]interface{}
    GetProcessInfo returns information about the current process

func (m *FactsModule) GetRuntimeInfo() map[string]interface{}
    GetRuntimeInfo returns Go runtime information

func (m *FactsModule) GetSystemLoad() map[string]interface{}
    GetSystemLoad returns current system load information

func (m *FactsModule) Validate(args map[string]interface{}) error
    Validate validates facts module arguments

type FileModule struct {
	*BaseModule
}
    FileModule manages files

func NewFileModule() *FileModule

func (m *FileModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error)

func (m *FileModule) GetDescription() string

func (m *FileModule) Validate(args map[string]interface{}) error

type GenericManager struct{}
    GenericManager implements basic service management

func (g *GenericManager) Disable(name string) error

func (g *GenericManager) Enable(name string) error

func (g *GenericManager) GetStatus(name string) (ServiceStatus, error)

func (g *GenericManager) IsEnabled(name string) (bool, error)

func (g *GenericManager) IsRunning(name string) (bool, error)

func (g *GenericManager) Reload(name string) error

func (g *GenericManager) Restart(name string) error

func (g *GenericManager) Start(name string) error

func (g *GenericManager) Stop(name string) error

type GenericPackageManager struct{}
    GenericPackageManager provides a fallback implementation

func (g *GenericPackageManager) Clean() error

func (g *GenericPackageManager) GetInfo(name string) (PackageInfo, error)

func (g *GenericPackageManager) Install(name, version string) error

func (g *GenericPackageManager) IsInstalled(name string) (bool, string, error)

func (g *GenericPackageManager) ListInstalled() ([]PackageInfo, error)

func (g *GenericPackageManager) Remove(name string) error

func (g *GenericPackageManager) Search(query string) ([]PackageInfo, error)

func (g *GenericPackageManager) Update(name string) error

func (g *GenericPackageManager) UpdateAll() error

type GitModule struct {
	*BaseModule
}
    GitModule handles Git repository operations

func NewGitModule() *GitModule
    NewGitModule creates a new git module

func (m *GitModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error)
    Execute performs git operations

func (m *GitModule) GetDescription() string

func (m *GitModule) Validate(args map[string]interface{}) error
    Validate validates git module arguments

type GroupModule struct {
	*BaseModule
}
    GroupModule manages system groups

func NewGroupModule() *GroupModule

func (m *GroupModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error)

func (m *GroupModule) GetDescription() string

func (m *GroupModule) Validate(args map[string]interface{}) error

type LaunchdManager struct{}
    LaunchdManager implements service management for macOS launchd

func (l *LaunchdManager) Disable(name string) error

func (l *LaunchdManager) Enable(name string) error

func (l *LaunchdManager) GetStatus(name string) (ServiceStatus, error)

func (l *LaunchdManager) IsEnabled(name string) (bool, error)

func (l *LaunchdManager) IsRunning(name string) (bool, error)

func (l *LaunchdManager) Reload(name string) error

func (l *LaunchdManager) Restart(name string) error

func (l *LaunchdManager) Start(name string) error

func (l *LaunchdManager) Stop(name string) error

type PackageInfo struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	Description  string `json:"description"`
	Architecture string `json:"architecture"`
	Repository   string `json:"repository"`
	Size         string `json:"size"`
	Installed    bool   `json:"installed"`
	Upgradable   bool   `json:"upgradable"`
	NewVersion   string `json:"new_version,omitempty"`
}
    PackageInfo represents package information

type PackageManager interface {
	Install(name, version string) error
	Remove(name string) error
	Update(name string) error
	UpdateAll() error
	IsInstalled(name string) (bool, string, error)
	Search(query string) ([]PackageInfo, error)
	ListInstalled() ([]PackageInfo, error)
	GetInfo(name string) (PackageInfo, error)
	Clean() error
}
    PackageManager interface for different package management systems

type PackageModule struct {
	BaseModule
	// Has unexported fields.
}
    PackageModule implements package management

func NewPackageModule() *PackageModule
    NewPackageModule creates a new package module

func (m *PackageModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error)
    Execute manages system packages

func (m *PackageModule) Validate(args map[string]interface{}) error
    Validate validates package module arguments

type PackageState string
    PackageState represents the desired package state

const (
	PackageStatePresent PackageState = "present"
	PackageStateAbsent  PackageState = "absent"
	PackageStateLatest  PackageState = "latest"
)
type PacmanManager struct{ GenericPackageManager }

type Registry struct {
	// Has unexported fields.
}
    Registry manages available modules

func NewRegistry() *Registry

func (r *Registry) ExecuteTask(ctx context.Context, task *types.Task, host types.Host, variables map[string]interface{}) (types.TaskResult, error)
    ExecuteTask executes task using appropriate module

func (r *Registry) Get(name string) (interfaces.ModuleExecutor, error)
    Get returns module by name (interface method)

func (r *Registry) GetModule(name string) (types.Module, error)
    GetModule returns module by name

func (r *Registry) List() []string
    List returns list of available modules (interface method)

func (r *Registry) ListModules() []string
    ListModules returns list of available modules

func (r *Registry) Register(name string, module interfaces.ModuleExecutor) error
    Register registers a new module (interface method)

func (r *Registry) RegisterModule(module types.Module)
    RegisterModule registers a new module (internal method)

func (r *Registry) Unregister(name string) error
    Unregister removes a module (interface method)

type ServiceAction string
    ServiceAction represents the action to perform

const (
	ServiceActionStart   ServiceAction = "start"
	ServiceActionStop    ServiceAction = "stop"
	ServiceActionRestart ServiceAction = "restart"
	ServiceActionReload  ServiceAction = "reload"
	ServiceActionEnable  ServiceAction = "enable"
	ServiceActionDisable ServiceAction = "disable"
	ServiceActionStatus  ServiceAction = "status"
)
type ServiceManager interface {
	Start(name string) error
	Stop(name string) error
	Restart(name string) error
	Reload(name string) error
	Enable(name string) error
	Disable(name string) error
	IsRunning(name string) (bool, error)
	IsEnabled(name string) (bool, error)
	GetStatus(name string) (ServiceStatus, error)
}
    ServiceManager interface for different service management systems

type ServiceModule struct {
	BaseModule
	// Has unexported fields.
}
    ServiceModule implements service management

func NewServiceModule() *ServiceModule
    NewServiceModule creates a new service module

func (m *ServiceModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error)
    Execute manages system services

func (m *ServiceModule) Validate(args map[string]interface{}) error
    Validate validates service module arguments

type ServiceStatus struct {
	Name        string    `json:"name"`
	Running     bool      `json:"running"`
	Enabled     bool      `json:"enabled"`
	PID         int       `json:"pid,omitempty"`
	Uptime      string    `json:"uptime,omitempty"`
	Memory      string    `json:"memory,omitempty"`
	CPU         string    `json:"cpu,omitempty"`
	Description string    `json:"description,omitempty"`
	LoadState   string    `json:"load_state,omitempty"`
	ActiveState string    `json:"active_state,omitempty"`
	SubState    string    `json:"sub_state,omitempty"`
	LastStarted time.Time `json:"last_started,omitempty"`
}
    ServiceStatus represents service status information

type ShellModule struct {
	BaseModule
}
    ShellModule executes shell commands with advanced features

func NewShellModule() *ShellModule

func (m *ShellModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error)

func (m *ShellModule) GetDescription() string

func (m *ShellModule) IsIdempotent() bool

func (m *ShellModule) Validate(args map[string]interface{}) error

type SysVInitManager struct{}
    SysVInitManager implements service management for SysV init

func (s *SysVInitManager) Disable(name string) error

func (s *SysVInitManager) Enable(name string) error

func (s *SysVInitManager) GetStatus(name string) (ServiceStatus, error)

func (s *SysVInitManager) IsEnabled(name string) (bool, error)

func (s *SysVInitManager) IsRunning(name string) (bool, error)

func (s *SysVInitManager) Reload(name string) error

func (s *SysVInitManager) Restart(name string) error

func (s *SysVInitManager) Start(name string) error

func (s *SysVInitManager) Stop(name string) error

type SystemdManager struct{}
    SystemdManager implements service management for systemd

func (s *SystemdManager) Disable(name string) error

func (s *SystemdManager) Enable(name string) error

func (s *SystemdManager) GetStatus(name string) (ServiceStatus, error)

func (s *SystemdManager) IsEnabled(name string) (bool, error)

func (s *SystemdManager) IsRunning(name string) (bool, error)

func (s *SystemdManager) Reload(name string) error

func (s *SystemdManager) Restart(name string) error

func (s *SystemdManager) Start(name string) error

func (s *SystemdManager) Stop(name string) error

type TemplateModule struct {
	*BaseModule
	// Has unexported fields.
}
    TemplateModule handles template file processing

func NewTemplateModule() *TemplateModule
    NewTemplateModule creates a new template module

func (m *TemplateModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error)
    Execute processes a template file

func (m *TemplateModule) GetDescription() string

func (m *TemplateModule) Validate(args map[string]interface{}) error
    Validate validates template module arguments

type TemplateOptions struct {
	TrimBlocks          bool   `json:"trim_blocks"`
	LStripBlocks        bool   `json:"lstrip_blocks"`
	KeepTrailingNewline bool   `json:"keep_trailing_newline"`
	BlockStartString    string `json:"block_start_string"`
	BlockEndString      string `json:"block_end_string"`
	VariableStartString string `json:"variable_start_string"`
	VariableEndString   string `json:"variable_end_string"`
	CommentStartString  string `json:"comment_start_string"`
	CommentEndString    string `json:"comment_end_string"`
}
    TemplateOptions holds template processing options

type UserModule struct {
	*BaseModule
}
    UserModule manages system users

func NewUserModule() *UserModule

func (m *UserModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error)

func (m *UserModule) GetDescription() string

func (m *UserModule) Validate(args map[string]interface{}) error

type WindowsServiceManager struct{}
    WindowsServiceManager implements service management for Windows

func (w *WindowsServiceManager) Disable(name string) error

func (w *WindowsServiceManager) Enable(name string) error

func (w *WindowsServiceManager) GetStatus(name string) (ServiceStatus, error)

func (w *WindowsServiceManager) IsEnabled(name string) (bool, error)

func (w *WindowsServiceManager) IsRunning(name string) (bool, error)

func (w *WindowsServiceManager) Reload(name string) error

func (w *WindowsServiceManager) Restart(name string) error

func (w *WindowsServiceManager) Start(name string) error

func (w *WindowsServiceManager) Stop(name string) error

type YumManager struct{}
    YumManager implements package management for YUM (RHEL/CentOS)

func (y *YumManager) Clean() error

func (y *YumManager) GetInfo(name string) (PackageInfo, error)

func (y *YumManager) Install(name, version string) error

func (y *YumManager) IsInstalled(name string) (bool, string, error)

func (y *YumManager) ListInstalled() ([]PackageInfo, error)

func (y *YumManager) Remove(name string) error

func (y *YumManager) Search(query string) ([]PackageInfo, error)

func (y *YumManager) Update(name string) error

func (y *YumManager) UpdateAll() error

type ZypperManager struct{ GenericPackageManager }

