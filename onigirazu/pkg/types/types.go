package types

import (
	"context"
	"fmt"
	"time"
)

// Host represents a target host
type Host struct {
	Name     string                 `yaml:"name"`
	Address  string                 `yaml:"address"`
	Port     int                    `yaml:"port,omitempty"`
	User     string                 `yaml:"user,omitempty"`
	Password string                 `yaml:"password,omitempty"`
	KeyFile  string                 `yaml:"key_file,omitempty"`
	Vars     map[string]interface{} `yaml:"vars,omitempty"`
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
		"name":     true,
		"address":  true,
		"port":     true,
		"user":     true,
		"password": true,
		"key_file": true,
		"vars":     true,
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
		fmt.Printf("[DEBUG YAML] Task '%s' parsed become=%v\n", t.Name, become)
	} else if becomeVal, exists := taskMap["become"]; exists {
		fmt.Printf("[DEBUG YAML] Task '%s' has become field but wrong type: %T = %v\n", t.Name, becomeVal, becomeVal)
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

// State represents saved state
type State struct {
	LastRun   time.Time              `json:"last_run"`
	Playbook  string                 `json:"playbook"`
	Results   []PlayResult           `json:"results"`
	Variables map[string]interface{} `json:"variables"`
	Checksums map[string]string      `json:"checksums"`
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
