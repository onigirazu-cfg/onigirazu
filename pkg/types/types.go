package types

import (
	"context"
	"encoding/json"
	"time"
)

// Host represents a target host
type Host struct {
	Name                  string                 `yaml:"name"`
	Address               string                 `yaml:"address"`
	Port                  int                    `yaml:"port,omitempty"`
	User                  string                 `yaml:"user,omitempty"`
	Password              string                 `yaml:"password,omitempty"`
	KeyFile               string                 `yaml:"key_file,omitempty"`
	InsecureIgnoreHostKey bool                   `yaml:"insecure_ignore_host_key,omitempty"`
	Vars                  map[string]interface{} `yaml:"vars,omitempty"`
}

// UnmarshalYAML implements custom YAML unmarshaling for Host
// This allows capturing both standard fields and Onigirazu-style variables (onigirazu_host, onigirazu_user, etc.)
func (h *Host) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// First unmarshal as a map to get all fields
	var hostMap map[string]interface{}
	if err := unmarshal(&hostMap); err != nil {
		return err
	}

	// Define reserved field names that map to struct fields
	reservedFields := map[string]bool{
		"name":                     true,
		"address":                  true,
		"port":                     true,
		"user":                     true,
		"password":                 true,
		"key_file":                 true,
		"insecure_ignore_host_key": true,
		"vars":                     true,
	}

	// Initialize Vars map
	h.Vars = make(map[string]interface{})

	// Extract known fields
	if name, ok := hostMap["name"].(string); ok {
		h.Name = name
	}
	if address, ok := hostMap["address"].(string); ok {
		h.Address = address
	}
	if port, ok := hostMap["port"].(int); ok {
		h.Port = port
	}
	if user, ok := hostMap["user"].(string); ok {
		h.User = user
	}
	if password, ok := hostMap["password"].(string); ok {
		h.Password = password
	}
	if keyFile, ok := hostMap["key_file"].(string); ok {
		h.KeyFile = keyFile
	}
	if insecure, ok := hostMap["insecure_ignore_host_key"].(bool); ok {
		h.InsecureIgnoreHostKey = insecure
	}

	// Handle vars field if it exists (nested vars)
	if vars, ok := hostMap["vars"].(map[string]interface{}); ok {
		for k, v := range vars {
			h.Vars[k] = v
		}
	}

	// Collect all other fields (including onigirazu_* variables) into Vars
	for key, value := range hostMap {
		if !reservedFields[key] {
			h.Vars[key] = value
		}
	}

	return nil
}

// UnmarshalJSON implements custom JSON unmarshaling for Host
func (h *Host) UnmarshalJSON(data []byte) error {
	// First unmarshal as a map to get all fields
	var hostMap map[string]interface{}
	if err := json.Unmarshal(data, &hostMap); err != nil {
		return err
	}

	// Define reserved field names that map to struct fields
	reservedFields := map[string]bool{
		"name":                     true,
		"address":                  true,
		"port":                     true,
		"user":                     true,
		"password":                 true,
		"key_file":                 true,
		"insecure_ignore_host_key": true,
		"vars":                     true,
	}

	// Initialize Vars map
	h.Vars = make(map[string]interface{})

	// Extract known fields
	if name, ok := hostMap["name"].(string); ok {
		h.Name = name
	}
	if address, ok := hostMap["address"].(string); ok {
		h.Address = address
	}
	// JSON numbers are float64 by default
	if port, ok := hostMap["port"].(float64); ok {
		h.Port = int(port)
	}
	if user, ok := hostMap["user"].(string); ok {
		h.User = user
	}
	if password, ok := hostMap["password"].(string); ok {
		h.Password = password
	}
	if keyFile, ok := hostMap["key_file"].(string); ok {
		h.KeyFile = keyFile
	}
	if insecure, ok := hostMap["insecure_ignore_host_key"].(bool); ok {
		h.InsecureIgnoreHostKey = insecure
	}

	// Handle vars field if it exists (nested vars)
	if vars, ok := hostMap["vars"].(map[string]interface{}); ok {
		for k, v := range vars {
			h.Vars[k] = v
		}
	}

	// Collect all other fields into Vars
	for key, value := range hostMap {
		if !reservedFields[key] {
			h.Vars[key] = value
		}
	}

	return nil
}

// Inventory represents a host inventory
type Inventory struct {
	Hosts  []Host            `yaml:"hosts"`
	Groups map[string]*Group `yaml:"groups,omitempty"`
}

// Task represents a single task
type Task struct {
	Name         string                 `yaml:"name"`
	Module       string                 `yaml:"module"`
	Args         map[string]interface{} `yaml:"args,omitempty"`
	When         string                 `yaml:"when,omitempty"`
	Loop         *Loop                  `yaml:"loop,omitempty"`
	Register     string                 `yaml:"register,omitempty"`
	IgnoreErrors bool                   `yaml:"ignore_errors,omitempty"`
	Tags         []string               `yaml:"tags,omitempty"`
	Notify       []string               `yaml:"notify,omitempty"`
	Timeout      time.Duration          `yaml:"timeout,omitempty"`
	Retries      int                    `yaml:"retries,omitempty"`
	Delay        time.Duration          `yaml:"delay,omitempty"`
	Until        string                 `yaml:"until,omitempty"`
	ChangedWhen  string                 `yaml:"changed_when,omitempty"`
	FailedWhen   string                 `yaml:"failed_when,omitempty"`
	Include      string                 `yaml:"include,omitempty"`
	Serial       bool                   `yaml:"serial,omitempty"`
	RetryDelay   time.Duration          `yaml:"retry_delay,omitempty"`
	Become       bool                   `yaml:"become,omitempty"`
	BecomeUser   string                 `yaml:"become_user,omitempty"`
	BecomeMethod string                 `yaml:"become_method,omitempty"`
}

// UnmarshalYAML implements custom YAML unmarshaling for Task
// Supports both old syntax (with args:) and new syntax (inline args)
func (t *Task) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// First try to unmarshal as a map to get all fields
	var taskMap map[string]interface{}
	if err := unmarshal(&taskMap); err != nil {
		return err
	}

	// Define reserved field names that are not module arguments
	reservedFields := map[string]bool{
		"name":          true,
		"module":        true,
		"args":          true,
		"when":          true,
		"loop":          true,
		"register":      true,
		"ignore_errors": true,
		"tags":          true,
		"notify":        true,
		"timeout":       true,
		"retries":       true,
		"delay":         true,
		"until":         true,
		"changed_when":  true,
		"failed_when":   true,
		"include":       true,
		"serial":        true,
		"retry_delay":   true,
		"become":        true,
		"become_user":   true,
		"become_method": true,
	}

	// Extract basic fields
	if name, ok := taskMap["name"].(string); ok {
		t.Name = name
	}
	// Extract module field first (will be overridden later if it's a map)
	if module, ok := taskMap["module"].(string); ok {
		t.Module = module
	}
	if when, ok := taskMap["when"].(string); ok {
		t.When = when
	}
	if register, ok := taskMap["register"].(string); ok {
		t.Register = register
	}
	if ignoreErrors, ok := taskMap["ignore_errors"].(bool); ok {
		t.IgnoreErrors = ignoreErrors
	}
	if include, ok := taskMap["include"].(string); ok {
		t.Include = include
	}
	if serial, ok := taskMap["serial"].(bool); ok {
		t.Serial = serial
	}
	if retries, ok := taskMap["retries"].(int); ok {
		t.Retries = retries
	}
	if become, ok := taskMap["become"].(bool); ok {
		t.Become = become
	}
	if becomeUser, ok := taskMap["become_user"].(string); ok {
		t.BecomeUser = becomeUser
	}
	if becomeMethod, ok := taskMap["become_method"].(string); ok {
		t.BecomeMethod = becomeMethod
	}

	// Handle tags
	if tags, ok := taskMap["tags"]; ok {
		if tagSlice, ok := tags.([]interface{}); ok {
			t.Tags = make([]string, len(tagSlice))
			for i, tag := range tagSlice {
				if tagStr, ok := tag.(string); ok {
					t.Tags[i] = tagStr
				}
			}
		}
	}

	// Handle notify
	if notify, ok := taskMap["notify"]; ok {
		if notifySlice, ok := notify.([]interface{}); ok {
			t.Notify = make([]string, len(notifySlice))
			for i, n := range notifySlice {
				if nStr, ok := n.(string); ok {
					t.Notify[i] = nStr
				}
			}
		}
	}

	// Handle duration fields
	if timeout, ok := taskMap["timeout"]; ok {
		if timeoutStr, ok := timeout.(string); ok {
			if d, err := time.ParseDuration(timeoutStr); err == nil {
				t.Timeout = d
			}
		}
	}
	if delay, ok := taskMap["delay"]; ok {
		if delayStr, ok := delay.(string); ok {
			if d, err := time.ParseDuration(delayStr); err == nil {
				t.Delay = d
			}
		}
	}
	if retryDelay, ok := taskMap["retry_delay"]; ok {
		if retryDelayStr, ok := retryDelay.(string); ok {
			if d, err := time.ParseDuration(retryDelayStr); err == nil {
				t.RetryDelay = d
			}
		}
	}

	// Handle string fields
	if until, ok := taskMap["until"].(string); ok {
		t.Until = until
	}
	if changedWhen, ok := taskMap["changed_when"].(string); ok {
		t.ChangedWhen = changedWhen
	}
	if failedWhen, ok := taskMap["failed_when"].(string); ok {
		t.FailedWhen = failedWhen
	}

	// Handle loop
	if loop, ok := taskMap["loop"]; ok {
		if loopMap, ok := loop.(map[string]interface{}); ok {
			t.Loop = &Loop{}
			if items, ok := loopMap["items"]; ok {
				if itemSlice, ok := items.([]interface{}); ok {
					t.Loop.Items = itemSlice
				}
			}
			if variable, ok := loopMap["variable"].(string); ok {
				t.Loop.Variable = variable
			}
		}
	}

	// Initialize Args map
	t.Args = make(map[string]interface{})

	// Check if there's an explicit "args" field (old syntax)
	if args, ok := taskMap["args"]; ok {
		if argsMap, ok := args.(map[string]interface{}); ok {
			t.Args = argsMap
		} else if argsMapInterface, ok := args.(map[interface{}]interface{}); ok {
			// Convert map[interface{}]interface{} to map[string]interface{}
			t.Args = make(map[string]interface{})
			for k, v := range argsMapInterface {
				if keyStr, ok := k.(string); ok {
					t.Args[keyStr] = v
				}
			}
		}
	} else {
		// Check for nested syntax: parameters under module field
		if module, ok := taskMap["module"]; ok {
			if moduleMap, ok := module.(map[string]interface{}); ok {
				// Nested syntax: module is a map containing module name and parameters
				for key, value := range moduleMap {
					if key == "type" {
						// Module name specified as "type" field
						if moduleStr, ok := value.(string); ok {
							t.Module = moduleStr
						}
					} else {
						// All other fields are module arguments
						t.Args[key] = value
					}
				}
			} else if moduleMapInterface, ok := module.(map[interface{}]interface{}); ok {
				// Nested syntax with interface{} keys (YAML default)
				for k, v := range moduleMapInterface {
					if keyStr, ok := k.(string); ok {
						if keyStr == "type" {
							// Module name specified as "type" field
							if moduleStr, ok := v.(string); ok {
								t.Module = moduleStr
							}
						} else {
							// All other fields are module arguments
							t.Args[keyStr] = v
						}
					}
				}
			} else if moduleStr, ok := module.(string); ok {
				// Standard syntax: module is just a string
				t.Module = moduleStr
				// Collect all non-reserved fields as module arguments
				for key, value := range taskMap {
					if !reservedFields[key] {
						t.Args[key] = value
					}
				}
			}
		} else {
			// New simplified syntax: check for non-reserved fields that are maps
			// These represent module definitions (e.g., package:, user:, template:)
			moduleFound := false
			for key, value := range taskMap {
				if !reservedFields[key] {
					// Check if this field is a map (indicating it's a module definition)
					if moduleMap, ok := value.(map[string]interface{}); ok {
						// This is the new syntax: field name is the module name
						t.Module = key
						t.Args = moduleMap
						moduleFound = true
						break
					} else if moduleMapInterface, ok := value.(map[interface{}]interface{}); ok {
						// Handle map[interface{}]interface{} (YAML default)
						t.Module = key
						t.Args = make(map[string]interface{})
						for k, v := range moduleMapInterface {
							if keyStr, ok := k.(string); ok {
								t.Args[keyStr] = v
							}
						}
						moduleFound = true
						break
					}
				}
			}

			// If no module found with new syntax, fallback to inline args
			if !moduleFound {
				for key, value := range taskMap {
					if !reservedFields[key] {
						t.Args[key] = value
					}
				}
			}
		}
	}

	return nil
}

// MarshalYAML implements custom YAML marshaling for Task
// Formats the task using the new simplified syntax (module name as field)
func (t *Task) MarshalYAML() (interface{}, error) {
	// Create a map to hold the task structure
	result := make(map[string]interface{})

	// Add name if present
	if t.Name != "" {
		result["name"] = t.Name
	}

	// Add module with its arguments using new syntax
	// Instead of "module: { type: package, ... }", use "package: { ... }"
	if t.Module != "" && len(t.Args) > 0 {
		result[t.Module] = t.Args
	}

	// Add task-level fields
	if t.When != "" {
		result["when"] = t.When
	}
	if t.Loop != nil {
		result["loop"] = t.Loop
	}
	if t.Register != "" {
		result["register"] = t.Register
	}
	if t.IgnoreErrors {
		result["ignore_errors"] = t.IgnoreErrors
	}
	if len(t.Tags) > 0 {
		result["tags"] = t.Tags
	}
	if len(t.Notify) > 0 {
		result["notify"] = t.Notify
	}
	if t.Timeout > 0 {
		result["timeout"] = t.Timeout.String()
	}
	if t.Retries > 0 {
		result["retries"] = t.Retries
	}
	if t.Delay > 0 {
		result["delay"] = t.Delay.String()
	}
	if t.Until != "" {
		result["until"] = t.Until
	}
	if t.ChangedWhen != "" {
		result["changed_when"] = t.ChangedWhen
	}
	if t.FailedWhen != "" {
		result["failed_when"] = t.FailedWhen
	}
	if t.Include != "" {
		result["include"] = t.Include
	}
	if t.Serial {
		result["serial"] = t.Serial
	}
	if t.RetryDelay > 0 {
		result["retry_delay"] = t.RetryDelay.String()
	}

	return result, nil
}

// Play represents a set of tasks to execute
type Play struct {
	Name              string                 `yaml:"name"`
	Hosts             string                 `yaml:"hosts"`
	Vars              map[string]interface{} `yaml:"vars,omitempty"`
	Tasks             []Task                 `yaml:"tasks"`
	PreTasks          []Task                 `yaml:"pre_tasks,omitempty"`
	PostTasks         []Task                 `yaml:"post_tasks,omitempty"`
	Handlers          []Task                 `yaml:"handlers,omitempty"`
	Become            bool                   `yaml:"become,omitempty"`
	BecomeUser        string                 `yaml:"become_user,omitempty"`
	BecomeMethod      string                 `yaml:"become_method,omitempty"`
	Tags              []string               `yaml:"tags,omitempty"`
	When              string                 `yaml:"when,omitempty"`
	Serial            interface{}            `yaml:"serial,omitempty"`
	MaxFailPercentage int                    `yaml:"max_fail_percentage,omitempty"`
	AnyErrorsFatal    bool                   `yaml:"any_errors_fatal,omitempty"`
	IgnoreErrors      bool                   `yaml:"ignore_errors,omitempty"`
	GatherFacts       bool                   `yaml:"gather_facts,omitempty"`
	Roles             []RoleReference        `yaml:"roles,omitempty"` // NEW: List of roles to execute
	RoleObjects       []*Role                `yaml:"-" json:"-"`      // NEW: Loaded role objects (internal)
}

// Playbook represents a complete playbook
type Playbook struct {
	Name     string                 `yaml:"name"`
	Plays    []Play                 `yaml:"plays"`
	Vars     map[string]interface{} `yaml:"vars,omitempty"`
	FilePath string                 `yaml:"-" json:"-"`
}

// TaskResult represents the result of task execution
type TaskResult struct {
	TaskName  string                 `json:"task_name"`
	Host      string                 `json:"host"`
	Module    string                 `json:"module"`
	Success   bool                   `json:"success"`
	Failed    bool                   `json:"failed"`
	Changed   bool                   `json:"changed"`
	Skipped   bool                   `json:"skipped"`
	Output    map[string]interface{} `json:"output"`
	Error     string                 `json:"error,omitempty"`
	Duration  time.Duration          `json:"duration"`
	Timestamp time.Time              `json:"timestamp"`
}

// PlayResult represents the result of play execution
type PlayResult struct {
	Name      string        `json:"name"`
	PlayName  string        `json:"play_name"`
	Host      string        `json:"host"`
	Hosts     []HostResult  `json:"hosts"`
	Tasks     []TaskResult  `json:"tasks"`
	Success   bool          `json:"success"`
	Duration  time.Duration `json:"duration"`
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
}

// HostResult represents the result for a specific host
type HostResult struct {
	Host    string       `json:"host"`
	Tasks   []TaskResult `json:"tasks"`
	Success bool         `json:"success"`
	Failed  bool         `json:"failed"`
}

// ExecutionMetadata holds execution context information
type ExecutionMetadata struct {
	User        string                 `json:"user"`        // User who ran the playbook
	Hostname    string                 `json:"hostname"`    // Machine hostname
	WorkingDir  string                 `json:"working_dir"` // Working directory
	Environment map[string]string      `json:"environment"` // Environment variables
	Tags        []string               `json:"tags"`        // Applied tags
	ExtraVars   map[string]interface{} `json:"extra_vars"`  // Extra variables passed
}

// State represents saved state
type State struct {
	Version    int                    `json:"version"` // Schema version for migrations (default: 1)
	LastRun    time.Time              `json:"last_run"`
	Playbook   string                 `json:"playbook"`
	Results    []PlayResult           `json:"results"`
	Variables  map[string]interface{} `json:"variables"`
	Checksums  map[string]string      `json:"checksums"`
	Metadata   *ExecutionMetadata     `json:"metadata,omitempty"`   // Execution context info
	Compressed bool                   `json:"compressed,omitempty"` // Is the state compressed
}

// Module represents a module interface
type Module interface {
	Execute(ctx context.Context, host Host, args map[string]interface{}) (TaskResult, error)
	Validate(args map[string]interface{}) error
	GetName() string
	GetDescription() string
}

// TaskState represents the state of a single task
type TaskState struct {
	TaskID    string                 `json:"task_id"`
	TaskName  string                 `json:"task_name"`
	Host      string                 `json:"host"`
	Module    string                 `json:"module"`
	Args      map[string]interface{} `json:"args"`
	Result    TaskResult             `json:"result"`
	LastRun   time.Time              `json:"last_run"`
	Checksum  string                 `json:"checksum"`
	Variables map[string]interface{} `json:"variables"`
}

// ExecutionContext holds context for task execution
type ExecutionContext struct {
	Variables      map[string]interface{} `json:"variables"`
	Facts          map[string]interface{} `json:"facts"`
	Hostvars       map[string]interface{} `json:"hostvars"`
	GroupVars      map[string]interface{} `json:"group_vars"`
	PlayVars       map[string]interface{} `json:"play_vars"`
	TaskVars       map[string]interface{} `json:"task_vars"`
	RegisteredVars map[string]interface{} `json:"registered_vars"`
	Host           *Host                  `json:"host,omitempty"`
	Task           *Task                  `json:"task,omitempty"`
}

// ProgressInfo represents progress information
type ProgressInfo struct {
	Total       int           `json:"total"`
	Completed   int           `json:"completed"`
	Failed      int           `json:"failed"`
	Skipped     int           `json:"skipped"`
	CurrentTask string        `json:"current_task"`
	CurrentHost string        `json:"current_host"`
	StartTime   time.Time     `json:"start_time"`
	Duration    time.Duration `json:"duration"`
}

// RetryInfo holds retry configuration
type RetryInfo struct {
	Attempts    int           `json:"attempts"`
	MaxAttempts int           `json:"max_attempts"`
	Delay       time.Duration `json:"delay"`
	LastError   string        `json:"last_error,omitempty"`
}

// ConditionResult represents the result of condition evaluation
type ConditionResult struct {
	Condition string `json:"condition"`
	Result    bool   `json:"result"`
	Error     string `json:"error,omitempty"`
}

// TemplateResult represents the result of template rendering
type TemplateResult struct {
	Original string `json:"original"`
	Rendered string `json:"rendered"`
	Error    string `json:"error,omitempty"`
}

// CacheEntry represents a cache entry
type CacheEntry struct {
	Key       string        `json:"key"`
	Value     interface{}   `json:"value"`
	CreatedAt time.Time     `json:"created_at"`
	ExpiresAt time.Time     `json:"expires_at"`
	TTL       time.Duration `json:"ttl"`
}

// AuditEntry represents an audit log entry
type AuditEntry struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Type      string                 `json:"type"` // task_start, task_end, playbook_start, playbook_end
	User      string                 `json:"user"`
	Host      string                 `json:"host"`
	Task      string                 `json:"task,omitempty"`
	Playbook  string                 `json:"playbook,omitempty"`
	Result    *TaskResult            `json:"result,omitempty"`
	Duration  time.Duration          `json:"duration,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// PlaybookResult represents the result of playbook execution
type PlaybookResult struct {
	Name         string                 `json:"name"`
	Success      bool                   `json:"success"`
	Failed       bool                   `json:"failed"`
	PlayResults  []PlayResult           `json:"play_results"`
	Plays        []PlayResult           `json:"plays"`
	TotalTasks   int                    `json:"total_tasks"`
	SuccessTasks int                    `json:"success_tasks"`
	FailedTasks  int                    `json:"failed_tasks"`
	SkippedTasks int                    `json:"skipped_tasks"`
	ChangedTasks int                    `json:"changed_tasks"`
	Duration     time.Duration          `json:"duration"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      time.Time              `json:"end_time"`
	Variables    map[string]interface{} `json:"variables,omitempty"`
	Stats        map[string]interface{} `json:"stats,omitempty"`
	Error        string                 `json:"error,omitempty"`
}

// Loop represents loop configuration for tasks
type Loop struct {
	Items    []interface{} `yaml:"items,omitempty" json:"items,omitempty"`
	Variable string        `yaml:"var,omitempty" json:"var,omitempty"`
	Index    string        `yaml:"index,omitempty" json:"index,omitempty"`
	Range    string        `yaml:"range,omitempty" json:"range,omitempty"`
}

// Group represents a host group
type Group struct {
	Name      string                 `yaml:"name" json:"name"`
	Hosts     map[string]*Host       `yaml:"hosts,omitempty" json:"hosts,omitempty"`
	Children  []string               `yaml:"children,omitempty" json:"children,omitempty"`
	Variables map[string]interface{} `yaml:"vars,omitempty" json:"vars,omitempty"`
	Vars      map[string]interface{} `yaml:"-" json:"-"` // Alias for Variables
}

// Role represents a reusable role
type Role struct {
	Name        string                 `yaml:"name"`
	Path        string                 // Filesystem path
	Description string                 `yaml:"description"`
	Version     string                 `yaml:"version"`
	Author      string                 `yaml:"author"`
	Tasks       []Task                 `yaml:"tasks"`
	Handlers    []Task                 `yaml:"handlers"`
	Defaults    map[string]interface{} `yaml:"defaults"`
	Vars        map[string]interface{} `yaml:"vars"`
	Files       map[string]string      // Built-in files: filename -> content
	Templates   map[string]string      // Built-in templates: filename -> content
	Meta        RoleMeta               `yaml:"meta"`
	PreTasks    []Task                 `yaml:"pre_tasks"`
	PostTasks   []Task                 `yaml:"post_tasks"`
}

// RoleMeta contains role metadata
type RoleMeta struct {
	Dependencies        []RoleDependency        `yaml:"dependencies"`
	MinVersion          string                  `yaml:"min_version"`
	MaxVersion          string                  `yaml:"max_version"`
	Tags                []string                `yaml:"tags"`
	Platforms           []string                `yaml:"platforms"`
	Parameters          map[string]ParameterDef `yaml:"parameters,omitempty"`            // Role parameter schema
	CrossParameterRules []CrossParameterRule    `yaml:"cross_parameter_rules,omitempty"` // Cross-parameter validation rules
	SchemaVersion       int                     `yaml:"schema_version,omitempty"`        // Current schema version (default: 1)
	Migrations          []SchemaMigration       `yaml:"migrations,omitempty"`            // Schema migrations
}

// RoleDependency specifies a role dependency
type RoleDependency struct {
	Name string                 `yaml:"name"`
	Vars map[string]interface{} `yaml:"vars,omitempty"`
}

// RoleReference is used in playbooks to reference roles
type RoleReference struct {
	Name string                 `yaml:"name"`
	Vars map[string]interface{} `yaml:"vars"`
	Path string                 `yaml:"path"`
	Tags []string               `yaml:"tags"`
	When string                 `yaml:"when,omitempty"` // Conditional execution
}

// ConditionalRequirement specifies when a parameter is required
type ConditionalRequirement struct {
	Condition   string `yaml:"condition"`             // Condition expression (e.g., "enable_auth=true")
	Description string `yaml:"description,omitempty"` // Why this parameter is required under condition
	ErrorMsg    string `yaml:"error,omitempty"`       // Custom error message
}

// CustomValidationRule defines a custom validation function for a parameter
type CustomValidationRule struct {
	Name        string      `yaml:"name"`                  // Validator name (e.g., "file_readable", "custom_check")
	Description string      `yaml:"description,omitempty"` // What this validator checks
	ErrorMsg    string      `yaml:"error,omitempty"`       // Custom error message
	Timeout     interface{} `yaml:"timeout,omitempty"`     // Timeout in milliseconds (default: 5000)
	Config      interface{} `yaml:"config,omitempty"`      // Validator-specific configuration
}

// ParameterDef defines a role parameter schema
type ParameterDef struct {
	Type                   string                  `yaml:"type"`                    // Parameter type: string, integer, boolean, array, object
	Required               bool                    `yaml:"required"`                // Whether parameter is always required
	ConditionalRequirement *ConditionalRequirement `yaml:"required_when,omitempty"` // Conditional requirement
	Default                interface{}             `yaml:"default,omitempty"`       // Default value
	Description            string                  `yaml:"description,omitempty"`   // Parameter description
	Constraints            ParameterConstraints    `yaml:"constraints,omitempty"`   // Type-specific constraints
	Validators             []CustomValidationRule  `yaml:"validators,omitempty"`    // Custom validators
}

// ParameterConstraints holds type-specific validation constraints
type ParameterConstraints struct {
	// For string type
	Pattern   string        `yaml:"pattern,omitempty"`    // Regex pattern
	MinLength int           `yaml:"min_length,omitempty"` // Minimum length
	MaxLength int           `yaml:"max_length,omitempty"` // Maximum length
	Enum      []interface{} `yaml:"enum,omitempty"`       // Allowed values

	// For numeric types (integer, float)
	Minimum    interface{} `yaml:"minimum,omitempty"`     // Minimum value
	Maximum    interface{} `yaml:"maximum,omitempty"`     // Maximum value
	MultipleOf interface{} `yaml:"multiple_of,omitempty"` // Value must be multiple of this

	// For array type
	ItemsType   string `yaml:"items_type,omitempty"`   // Type of array items: string, integer, etc.
	MinItems    int    `yaml:"min_items,omitempty"`    // Minimum array length
	MaxItems    int    `yaml:"max_items,omitempty"`    // Maximum array length
	UniqueItems bool   `yaml:"unique_items,omitempty"` // Array items must be unique

	// For object type
	RequiredFields []string `yaml:"required_fields,omitempty"` // Required object fields
}

// ParameterValidationError represents a parameter validation error
type ParameterValidationError struct {
	Parameter string      // Parameter name
	Error     string      // Error message
	Value     interface{} // The invalid value
}

// ValidationResult holds the result of parameter validation
type ValidationResult struct {
	Valid            bool
	Errors           []ParameterValidationError
	CrossParamErrors []CrossParameterValidationError `json:"cross_param_errors,omitempty"`
}

// CrossParameterRule defines a rule that validates multiple parameters together
type CrossParameterRule struct {
	Rule        string `yaml:"rule"`                  // The rule expression (e.g., "port=80 && service=http")
	Description string `yaml:"description,omitempty"` // Human-readable description
	ErrorMsg    string `yaml:"error"`                 // Error message when rule is violated
	Severity    string `yaml:"severity,omitempty"`    // "error" or "warning" (default: "error")
}

// CrossParameterValidationError represents a cross-parameter validation error
type CrossParameterValidationError struct {
	Rule     string                 `json:"rule"`      // The rule that failed
	Error    string                 `json:"error"`     // Error message
	ErrorMsg string                 `json:"error_msg"` // Custom error message from rule
	Details  map[string]interface{} `json:"details"`   // Details about the violation
}

// MigrationRuleType represents the type of migration rule
type MigrationRuleType string

const (
	MigrationRuleTypeRename     MigrationRuleType = "rename"
	MigrationRuleTypeTransform  MigrationRuleType = "transform"
	MigrationRuleTypeDeprecate  MigrationRuleType = "deprecate"
	MigrationRuleTypeRemove     MigrationRuleType = "remove"
	MigrationRuleTypeAddDefault MigrationRuleType = "add_default"
)

// MigrationRule defines a single schema migration rule
type MigrationRule struct {
	Type        MigrationRuleType      `yaml:"type"`        // Type of migration rule
	OldParam    string                 `yaml:"old_param"`   // Old parameter name (for rename)
	NewParam    string                 `yaml:"new_param"`   // New parameter name (for rename/transform)
	FromType    string                 `yaml:"from_type"`   // Original type (for transform)
	ToType      string                 `yaml:"to_type"`     // New type (for transform)
	Description string                 `yaml:"description"` // Migration description
	Default     interface{}            `yaml:"default"`     // Default value for new parameter
	Transformer map[string]interface{} `yaml:"transformer"` // Custom transformation rules
	Reason      string                 `yaml:"reason"`      // Why this migration was needed
}

// SchemaMigration defines migration rules from one version to another
type SchemaMigration struct {
	From  int             `yaml:"from"`            // From schema version
	To    int             `yaml:"to"`              // To schema version
	Rules []MigrationRule `yaml:"rules"`           // Migration rules
	Notes string          `yaml:"notes,omitempty"` // Migration notes
	Date  string          `yaml:"date,omitempty"`  // Migration date
	Error string          `yaml:"error,omitempty"` // Error message if failed
}

// SchemaVersionInfo holds information about schema versioning
type SchemaVersionInfo struct {
	Current    int               `yaml:"current"`              // Current schema version
	Supported  []int             `yaml:"supported"`            // Supported versions
	Migrations []SchemaMigration `yaml:"migrations,omitempty"` // Available migrations
	Deprecated bool              `yaml:"deprecated"`           // Whether current version is deprecated
}

// MigrationError represents a migration error
type MigrationError struct {
	From    int
	To      int
	Error   string
	Details map[string]interface{}
}
