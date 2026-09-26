package types

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
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

	// Privilege escalation of the task being run on this host; set per task by
	// the module registry, never read from inventory
	Become     bool   `yaml:"-" json:"-"`
	BecomeUser string `yaml:"-" json:"-"`
	// Environment of the current task, set by the registry for the executor
	Environment  map[string]string `yaml:"-" json:"-"`
	BecomeMethod string            `yaml:"-" json:"-"`
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
	Listen       string                 `yaml:"listen,omitempty"`
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
	// BecomeSet tells become: false (run without escalation even in a play
	// with become: true) from no become at all
	BecomeSet    bool   `yaml:"-" json:"-"`
	BecomeUser   string `yaml:"become_user,omitempty"`
	BecomeMethod string `yaml:"become_method,omitempty"`
	RunOnce      bool   `yaml:"run_once,omitempty"`
	DelegateTo   string `yaml:"delegate_to,omitempty"`
	// block / rescue / always (a task with a block has no module)
	Block  []Task `yaml:"block,omitempty"`
	Rescue []Task `yaml:"rescue,omitempty"`
	Always []Task `yaml:"always,omitempty"`
	// set by the engine on tasks inside a block with a rescue section: their
	// failure is handled by the rescue and does not fail the play by itself
	Rescuable bool `yaml:"-"`
	// CheckMode overrides --check for this task (check_mode: true/false)
	CheckMode *bool `yaml:"check_mode,omitempty"`
	// Environment variables for the commands the task runs
	Environment map[string]interface{} `yaml:"environment,omitempty"`
	// Vars are variables of this task only
	Vars map[string]interface{} `yaml:"vars,omitempty"`
	// NoLog hides the task's arguments, output and errors from logs and state
	NoLog bool `yaml:"no_log,omitempty"`
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
		"listen":        true,
		"timeout":       true,
		"retries":       true,
		"delay":         true,
		"until":         true,
		"changed_when":  true,
		"failed_when":   true,
		"include":       true,
		"include_tasks": true,
		"import_tasks":  true,
		"serial":        true,
		"retry_delay":   true,
		"become":        true,
		"become_user":   true,
		"become_method": true,
		"run_once":      true,
		"delegate_to":   true,
		"block":         true,
		"rescue":        true,
		"always":        true,
		"check_mode":    true,
		"environment":   true,
		"vars":          true,
		"no_log":        true,
		"loop_control":  true,
		"with_items":    true,
		"with_list":     true,
		"with_dict":     true,
	}

	if env, ok := taskMap["environment"].(map[string]interface{}); ok {
		t.Environment = env
	}
	if vars, ok := taskMap["vars"].(map[string]interface{}); ok {
		t.Vars = vars
	}
	if noLog, ok := yamlBool(taskMap["no_log"]); ok {
		t.NoLog = noLog
	}
	// with_items / with_list: the old spelling of loop
	for _, key := range []string{"with_items", "with_list"} {
		switch items := taskMap[key].(type) {
		case []interface{}:
			t.Loop = &Loop{Items: items}
		case string:
			t.Loop = &Loop{Expr: items}
		}
	}
	// with_dict: loop over {key, value} items of a dictionary
	switch d := taskMap["with_dict"].(type) {
	case map[string]interface{}:
		keys := make([]string, 0, len(d))
		for k := range d {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		items := make([]interface{}, 0, len(keys))
		for _, k := range keys {
			items = append(items, map[string]interface{}{"key": k, "value": d[k]})
		}
		t.Loop = &Loop{Items: items}
	case string:
		inner := strings.TrimSpace(d)
		if strings.HasPrefix(inner, "{{") && strings.HasSuffix(inner, "}}") {
			inner = strings.TrimSpace(inner[2 : len(inner)-2])
		}
		t.Loop = &Loop{Expr: "(" + inner + ") | dict2items"}
	}

	// check_mode: false runs the task for real in a check run; true checks
	// it in a normal run
	if check, ok := yamlBool(taskMap["check_mode"]); ok {
		t.CheckMode = &check
	}

	// block / rescue / always: nested task lists
	for key, target := range map[string]*[]Task{"block": &t.Block, "rescue": &t.Rescue, "always": &t.Always} {
		raw, ok := taskMap[key]
		if !ok {
			continue
		}
		data, err := yaml.Marshal(raw)
		if err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
		if err := yaml.Unmarshal(data, target); err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
	}

	// Extract basic fields
	if name, ok := taskMap["name"].(string); ok {
		t.Name = name
	}
	// Extract module field first (will be overridden later if it's a map)
	if module, ok := taskMap["module"].(string); ok {
		t.Module = module
	}
	t.When = conditionValue(taskMap["when"])
	if register, ok := taskMap["register"].(string); ok {
		t.Register = register
	}
	if ignoreErrors, ok := yamlBool(taskMap["ignore_errors"]); ok {
		t.IgnoreErrors = ignoreErrors
	}
	// include, include_tasks and import_tasks are expanded by the parser
	for _, key := range []string{"include", "include_tasks", "import_tasks"} {
		if include, ok := taskMap[key].(string); ok {
			t.Include = include
		}
	}
	if serial, ok := yamlBool(taskMap["serial"]); ok {
		t.Serial = serial
	}
	if retries, ok := taskMap["retries"].(int); ok {
		t.Retries = retries
	}
	if become, ok := yamlBool(taskMap["become"]); ok {
		t.Become, t.BecomeSet = become, true
	}
	if becomeUser, ok := taskMap["become_user"].(string); ok {
		t.BecomeUser = becomeUser
	}
	if becomeMethod, ok := taskMap["become_method"].(string); ok {
		t.BecomeMethod = becomeMethod
	}
	if runOnce, ok := yamlBool(taskMap["run_once"]); ok {
		t.RunOnce = runOnce
	}
	if delegateTo, ok := taskMap["delegate_to"].(string); ok {
		t.DelegateTo = delegateTo
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
	// notify: a handler name or a list of them
	switch notify := taskMap["notify"].(type) {
	case string:
		t.Notify = []string{notify}
	case []interface{}:
		for _, n := range notify {
			if nStr, ok := n.(string); ok {
				t.Notify = append(t.Notify, nStr)
			}
		}
	}

	// Handle listen
	if listen, ok := taskMap["listen"].(string); ok {
		t.Listen = listen
	}

	// Handle duration fields
	if d, ok := durationValue(taskMap["timeout"]); ok {
		t.Timeout = d
	}
	if d, ok := durationValue(taskMap["delay"]); ok {
		t.Delay = d
	}
	if d, ok := durationValue(taskMap["retry_delay"]); ok {
		t.RetryDelay = d
	}

	// Handle string fields
	t.Until = conditionValue(taskMap["until"])
	t.ChangedWhen = conditionValue(taskMap["changed_when"])
	t.FailedWhen = conditionValue(taskMap["failed_when"])

	// Handle loop
	if loop, ok := taskMap["loop"]; ok {
		var loopMap map[string]interface{}

		// Try to extract loop as map[string]interface{} (standard YAML parsing)
		if lm, ok := loop.(map[string]interface{}); ok {
			loopMap = lm
		} else if lmInterface, ok := loop.(map[interface{}]interface{}); ok {
			// Convert map[interface{}]interface{} to map[string]interface{}
			// This is the default YAML parser behavior
			loopMap = make(map[string]interface{})
			for k, v := range lmInterface {
				if keyStr, ok := k.(string); ok {
					loopMap[keyStr] = v
				}
			}
		}

		switch l := loop.(type) {
		case []interface{}: // loop: [a, b]
			t.Loop = &Loop{Items: l}
		case string: // loop: "{{ list_var }}"
			t.Loop = &Loop{Expr: l}
		}

		if loopMap != nil {
			t.Loop = &Loop{}

			// Extract items field: a list, or an expression that yields one
			switch items := loopMap["items"].(type) {
			case []interface{}:
				t.Loop.Items = items
			case string:
				t.Loop.Expr = items
			}

			// Extract variable field (YAML tag says "var" but we accept both)
			if variable, ok := loopMap["var"].(string); ok {
				t.Loop.Variable = variable
			} else if variable, ok := loopMap["variable"].(string); ok {
				t.Loop.Variable = variable
			}

			// Extract range field
			if loopRange, ok := loopMap["range"].(string); ok {
				t.Loop.Range = loopRange
			}

			// Extract index field
			if index, ok := loopMap["index"].(string); ok {
				t.Loop.Index = index
			}
		}
	}

	// loop_control: loop_var / index_var name the item and index variables
	if lc, ok := taskMap["loop_control"].(map[string]interface{}); ok && t.Loop != nil {
		if v, ok := lc["loop_var"].(string); ok {
			t.Loop.Variable = v
		}
		if v, ok := lc["index_var"].(string); ok {
			t.Loop.Index = v
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

	return t.applyShortForm(taskMap, reservedFields)
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
	if t.Listen != "" {
		result["listen"] = t.Listen
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

	for key, list := range map[string][]Task{"block": t.Block, "rescue": t.Rescue, "always": t.Always} {
		if len(list) > 0 {
			result[key] = list
		}
	}

	return result, nil
}

// Play represents a set of tasks to execute
type Play struct {
	Name              string                 `yaml:"name"`
	Hosts             string                 `yaml:"hosts"`
	Vars              map[string]interface{} `yaml:"vars,omitempty"`
	VarsFiles         []string               `yaml:"vars_files,omitempty"`
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
	// Environment of every task of the play; a task's own environment wins
	Environment map[string]interface{} `yaml:"environment,omitempty"`
	Roles       []RoleReference        `yaml:"roles,omitempty"` // NEW: List of roles to execute
	RoleObjects []*Role                `yaml:"-" json:"-"`      // NEW: Loaded role objects (internal)
}

// Playbook represents a complete playbook
type Playbook struct {
	Name     string                 `yaml:"name"`
	Plays    []Play                 `yaml:"plays"`
	Vars     map[string]interface{} `yaml:"vars,omitempty"`
	FilePath string                 `yaml:"-" json:"-"`
}

// UnmarshalYAML implements custom YAML unmarshaling for Playbook
// Supports both formats:
// 1. Structured: name: ..., plays: [...]
// 2. Direct list: [- hosts: ..., name: ..., tasks: ...]
func (pb *Playbook) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Try to unmarshal as a list of plays (Ansible format)
	var plays []Play
	if err := unmarshal(&plays); err == nil && len(plays) > 0 {
		// Successfully unmarshaled as list - generate playbook name from first play
		pb.Plays = plays
		if plays[0].Name != "" {
			pb.Name = plays[0].Name + " (playbook)"
		} else {
			pb.Name = "Generated Playbook"
		}
		return nil
	}

	// Fall back to structured format
	type playbookAlias Playbook
	if err := unmarshal((*playbookAlias)(pb)); err != nil {
		return err
	}

	return nil
}

// TaskResult represents the result of task execution
type TaskResult struct {
	TaskName string `json:"task_name"`
	Host     string `json:"host"`
	Module   string `json:"module"`
	Success  bool   `json:"success"`
	Failed   bool   `json:"failed"`
	Changed  bool   `json:"changed"`
	Skipped  bool   `json:"skipped"`
	// Ignored: the task failed, but ignore_errors or a rescue section handled it
	Ignored   bool                   `json:"ignored,omitempty"`
	Output    map[string]interface{} `json:"output"`
	Error     string                 `json:"error,omitempty"`
	Notify    []string               `json:"notify,omitempty"`
	Duration  time.Duration          `json:"duration_ms"` // Store as milliseconds for JSON compatibility
	Timestamp time.Time              `json:"timestamp"`
}

// MarshalJSON implements custom JSON marshaling for TaskResult
func (tr TaskResult) MarshalJSON() ([]byte, error) {
	type Alias TaskResult
	return json.Marshal(&struct {
		Duration int64 `json:"duration_ms"`
		*Alias
	}{
		Duration: tr.Duration.Milliseconds(),
		Alias:    (*Alias)(&tr),
	})
}

// UnmarshalJSON implements custom JSON unmarshaling for TaskResult
func (tr *TaskResult) UnmarshalJSON(data []byte) error {
	type Alias TaskResult
	aux := &struct {
		Duration int64 `json:"duration_ms"`
		*Alias
	}{
		Alias: (*Alias)(tr),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	tr.Duration = time.Duration(aux.Duration) * time.Millisecond
	return nil
}

// PlayResult represents the result of play execution
type PlayResult struct {
	Name      string        `json:"name"`
	PlayName  string        `json:"play_name"`
	Host      string        `json:"host"`
	Hosts     []HostResult  `json:"hosts"`
	Tasks     []TaskResult  `json:"tasks"`
	Success   bool          `json:"success"`
	Duration  time.Duration `json:"duration_ms"` // Store as milliseconds for JSON compatibility
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
}

// MarshalJSON implements custom JSON marshaling for PlayResult
func (pr PlayResult) MarshalJSON() ([]byte, error) {
	type Alias PlayResult
	return json.Marshal(&struct {
		Duration int64 `json:"duration_ms"`
		*Alias
	}{
		Duration: pr.Duration.Milliseconds(),
		Alias:    (*Alias)(&pr),
	})
}

// UnmarshalJSON implements custom JSON unmarshaling for PlayResult
func (pr *PlayResult) UnmarshalJSON(data []byte) error {
	type Alias PlayResult
	aux := &struct {
		Duration int64 `json:"duration_ms"`
		*Alias
	}{
		Alias: (*Alias)(pr),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	pr.Duration = time.Duration(aux.Duration) * time.Millisecond
	return nil
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
	TTL       time.Duration `json:"ttl_ms"` // Store as milliseconds for JSON compatibility
}

// MarshalJSON implements custom JSON marshaling for CacheEntry
func (ce CacheEntry) MarshalJSON() ([]byte, error) {
	type Alias CacheEntry
	return json.Marshal(&struct {
		TTL int64 `json:"ttl_ms"`
		*Alias
	}{
		TTL:   ce.TTL.Milliseconds(),
		Alias: (*Alias)(&ce),
	})
}

// UnmarshalJSON implements custom JSON unmarshaling for CacheEntry
func (ce *CacheEntry) UnmarshalJSON(data []byte) error {
	type Alias CacheEntry
	aux := &struct {
		TTL int64 `json:"ttl_ms"`
		*Alias
	}{
		Alias: (*Alias)(ce),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	ce.TTL = time.Duration(aux.TTL) * time.Millisecond
	return nil
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
	Duration  time.Duration          `json:"duration_ms,omitempty"` // Store as milliseconds for JSON compatibility
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// MarshalJSON implements custom JSON marshaling for AuditEntry
func (ae AuditEntry) MarshalJSON() ([]byte, error) {
	type Alias AuditEntry
	durationMS := int64(0)
	if ae.Duration != 0 {
		durationMS = ae.Duration.Milliseconds()
	}
	return json.Marshal(&struct {
		Duration int64 `json:"duration_ms,omitempty"`
		*Alias
	}{
		Duration: durationMS,
		Alias:    (*Alias)(&ae),
	})
}

// UnmarshalJSON implements custom JSON unmarshaling for AuditEntry
func (ae *AuditEntry) UnmarshalJSON(data []byte) error {
	type Alias AuditEntry
	aux := &struct {
		Duration int64 `json:"duration_ms,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(ae),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.Duration != 0 {
		ae.Duration = time.Duration(aux.Duration) * time.Millisecond
	}
	return nil
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
	Duration     time.Duration          `json:"duration_ms"` // Store as milliseconds for JSON compatibility
	StartTime    time.Time              `json:"start_time"`
	EndTime      time.Time              `json:"end_time"`
	Variables    map[string]interface{} `json:"variables,omitempty"`
	Stats        map[string]interface{} `json:"stats,omitempty"`
	Error        string                 `json:"error,omitempty"`
}

// MarshalJSON implements custom JSON marshaling for PlaybookResult
func (pbr PlaybookResult) MarshalJSON() ([]byte, error) {
	type Alias PlaybookResult
	return json.Marshal(&struct {
		Duration int64 `json:"duration_ms"`
		*Alias
	}{
		Duration: pbr.Duration.Milliseconds(),
		Alias:    (*Alias)(&pbr),
	})
}

// UnmarshalJSON implements custom JSON unmarshaling for PlaybookResult
func (pbr *PlaybookResult) UnmarshalJSON(data []byte) error {
	type Alias PlaybookResult
	aux := &struct {
		Duration int64 `json:"duration_ms"`
		*Alias
	}{
		Alias: (*Alias)(pbr),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	pbr.Duration = time.Duration(aux.Duration) * time.Millisecond
	return nil
}

// Loop represents loop configuration for tasks
type Loop struct {
	Items    []interface{} `yaml:"items,omitempty" json:"items,omitempty"`
	Variable string        `yaml:"var,omitempty" json:"var,omitempty"`
	Index    string        `yaml:"index,omitempty" json:"index,omitempty"`
	Range    string        `yaml:"range,omitempty" json:"range,omitempty"`
	// Expr is an expression that yields the items, evaluated per host:
	// loop: "{{ result.stdout_lines }}"
	Expr string `yaml:"expr,omitempty" json:"expr,omitempty"`
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
	// Params are the variables given with the role in the play
	// ("- role: web" + "port: 80"); they win over the role's own vars
	Params map[string]interface{} `yaml:"-" json:"-"`
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

// UnmarshalYAML accepts hosts as a pattern string or as a list of patterns
// (joined with ",", which the inventory reads as a union)
func (p *Play) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(value.Content); i += 2 {
			key, val := value.Content[i], value.Content[i+1]
			if key.Value != "hosts" || val.Kind != yaml.SequenceNode {
				continue
			}
			parts := make([]string, 0, len(val.Content))
			for _, item := range val.Content {
				parts = append(parts, item.Value)
			}
			value.Content[i+1] = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: strings.Join(parts, ",")}
		}
	}
	// Ansible gathers facts unless the play says gather_facts: false
	p.GatherFacts = true
	type plain Play
	return value.Decode((*plain)(p))
}

// UnmarshalYAML accepts the Ansible forms of a role reference: a bare name
// ("- common"), "role:" as an alias of "name:", and role variables given
// directly next to the name
func (r *RoleReference) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		r.Name = value.Value
		return nil
	}
	var raw map[string]interface{}
	if err := value.Decode(&raw); err != nil {
		return err
	}
	type plain RoleReference
	var p plain
	if err := value.Decode(&p); err != nil {
		return err
	}
	*r = RoleReference(p)
	if r.Name == "" {
		r.Name, _ = raw["role"].(string)
	}
	for k, v := range raw {
		switch k {
		case "name", "role", "vars", "path", "tags", "when":
			continue
		}
		if r.Vars == nil {
			r.Vars = make(map[string]interface{})
		}
		r.Vars[k] = v
	}
	return nil
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

// conditionValue reads when/until/changed_when/failed_when: a string, a
// boolean, or a list whose items must all hold
func conditionValue(v interface{}) string {
	switch c := v.(type) {
	case string:
		return c
	case bool:
		return strconv.FormatBool(c)
	case []interface{}:
		parts := make([]string, 0, len(c))
		for _, item := range c {
			if cond := conditionValue(item); cond != "" {
				parts = append(parts, "("+cond+")")
			}
		}
		return strings.Join(parts, " and ")
	}
	return ""
}

// durationValue reads a duration written as "5s" or as a number of seconds
func durationValue(v interface{}) (time.Duration, bool) {
	switch d := v.(type) {
	case string:
		if parsed, err := time.ParseDuration(d); err == nil {
			return parsed, true
		}
		if secs, err := strconv.ParseFloat(strings.TrimSpace(d), 64); err == nil {
			return time.Duration(secs * float64(time.Second)), true
		}
	case int:
		return time.Duration(d) * time.Second, true
	case float64:
		return time.Duration(d * float64(time.Second)), true
	}
	return 0, false
}

// yamlBool reads a boolean task keyword: YAML true/false, or the yes/no,
// on/off strings Ansible playbooks use (YAML 1.2 keeps them as strings)
func yamlBool(v interface{}) (bool, bool) {
	switch b := v.(type) {
	case bool:
		return b, true
	case string:
		switch strings.ToLower(b) {
		case "yes", "true", "on", "y":
			return true, true
		case "no", "false", "off", "n":
			return false, true
		}
	}
	return false, false
}
