package security

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// SecurityValidator provides security validation capabilities
type SecurityValidator struct {
	config SecurityConfig
	rules  []ValidationRule
}

// SecurityConfig holds security configuration
type SecurityConfig struct {
	AllowedHosts        []string          `json:"allowed_hosts"`
	AllowedPorts        []int             `json:"allowed_ports"`
	AllowedModules      []string          `json:"allowed_modules"`
	BlockedCommands     []string          `json:"blocked_commands"`
	MaxFileSize         int64             `json:"max_file_size"`
	AllowedFileTypes    []string          `json:"allowed_file_types"`
	RequireEncryption   bool              `json:"require_encryption"`
	MaxRetries          int               `json:"max_retries"`
	MaxTimeout          time.Duration     `json:"max_timeout"`
	AllowedDirectories  []string          `json:"allowed_directories"`
	BlockedDirectories  []string          `json:"blocked_directories"`
	RequiredPermissions map[string]string `json:"required_permissions"`
	AuditEnabled        bool              `json:"audit_enabled"`
	LogLevel            string            `json:"log_level"`
}

// ValidationRule represents a security validation rule
type ValidationRule struct {
	Name      string                 `json:"name"`
	Type      RuleType               `json:"type"`
	Pattern   string                 `json:"pattern,omitempty"`
	Validator func(interface{}) bool `json:"-"`
	Message   string                 `json:"message"`
	Severity  Severity               `json:"severity"`
	Enabled   bool                   `json:"enabled"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// RuleType represents the type of validation rule
type RuleType string

const (
	RuleTypeHost    RuleType = "host"
	RuleTypeModule  RuleType = "module"
	RuleTypeCommand RuleType = "command"
	RuleTypeFile    RuleType = "file"
	RuleTypePath    RuleType = "path"
	RuleTypeNetwork RuleType = "network"
	RuleTypeContent RuleType = "content"
	RuleTypeUser    RuleType = "user"
	RuleTypeGroup   RuleType = "group"
	RuleTypeCustom  RuleType = "custom"
)

// Severity represents the severity level
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// ValidationResult represents the result of security validation
type ValidationResult struct {
	Valid      bool                   `json:"valid"`
	Violations []SecurityViolation    `json:"violations"`
	Warnings   []SecurityWarning      `json:"warnings"`
	Score      int                    `json:"score"`
	MaxScore   int                    `json:"max_score"`
	Timestamp  time.Time              `json:"timestamp"`
	Duration   time.Duration          `json:"duration"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// Error implements the error interface for ValidationResult
func (vr ValidationResult) Error() string {
	if vr.Valid {
		return ""
	}
	if len(vr.Violations) > 0 {
		return vr.Violations[0].Message
	}
	return "validation failed"
}

// SecurityViolation represents a security violation
type SecurityViolation struct {
	Rule       string                 `json:"rule"`
	Type       RuleType               `json:"type"`
	Severity   Severity               `json:"severity"`
	Message    string                 `json:"message"`
	Value      interface{}            `json:"value"`
	Suggestion string                 `json:"suggestion,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// SecurityWarning represents a security warning
type SecurityWarning struct {
	Rule       string                 `json:"rule"`
	Message    string                 `json:"message"`
	Value      interface{}            `json:"value"`
	Suggestion string                 `json:"suggestion,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// DefaultSecurityConfig returns a default security configuration
func DefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		AllowedHosts:        []string{"*"}, // Allow all hosts by default
		AllowedPorts:        []int{22, 80, 443, 8080, 9090},
		AllowedModules:      []string{"command", "shell", "file", "copy", "fetch", "get_url", "template", "service", "package", "user", "group", "git", "debug", "set_fact", "stat", "lineinfile", "config", "cron", "systemd", "firewall"},
		BlockedCommands:     []string{"rm -rf /", "dd if=", "mkfs", "fdisk", "format"},
		MaxFileSize:         100 * 1024 * 1024, // 100MB
		AllowedFileTypes:    []string{".txt", ".conf", ".cfg", ".ini", ".yaml", ".yml", ".json", ".xml"},
		RequireEncryption:   false,
		MaxRetries:          3,
		MaxTimeout:          time.Minute * 30,
		AllowedDirectories:  []string{"/tmp", "/var/tmp", "/home", "/opt"},
		BlockedDirectories:  []string{"/etc/passwd", "/etc/shadow", "/boot", "/sys", "/proc"},
		RequiredPermissions: map[string]string{},
		AuditEnabled:        true,
		LogLevel:            "info",
	}
}

// NewSecurityValidator creates a new security validator
func NewSecurityValidator(config SecurityConfig) *SecurityValidator {
	validator := &SecurityValidator{
		config: config,
		rules:  make([]ValidationRule, 0),
	}

	// Add default security rules
	validator.addDefaultRules()

	return validator
}

// ValidateHost validates host configuration
func (sv *SecurityValidator) ValidateHost(host types.Host) ValidationResult {
	startTime := time.Now()
	result := ValidationResult{
		Valid:      true,
		Violations: make([]SecurityViolation, 0),
		Warnings:   make([]SecurityWarning, 0),
		Timestamp:  startTime,
		Metadata:   make(map[string]interface{}),
	}

	// Validate host address
	if !sv.isAllowedHost(host.Address) {
		result.addViolation("host_not_allowed", RuleTypeHost, SeverityHigh,
			fmt.Sprintf("Host %s is not in allowed hosts list", host.Address),
			host.Address, "Add host to allowed_hosts configuration")
	}

	// Validate port
	if host.Port > 0 && !sv.isAllowedPort(host.Port) {
		result.addViolation("port_not_allowed", RuleTypeNetwork, SeverityMedium,
			fmt.Sprintf("Port %d is not in allowed ports list", host.Port),
			host.Port, "Use an allowed port or add port to allowed_ports configuration")
	}

	// Validate IP address format
	if !sv.isValidIPAddress(host.Address) && !sv.isValidHostname(host.Address) {
		result.addViolation("invalid_host_format", RuleTypeHost, SeverityHigh,
			fmt.Sprintf("Host address %s is not a valid IP address or hostname", host.Address),
			host.Address, "Use a valid IP address or hostname")
	}

	// Check for private key file security
	if host.KeyFile != "" {
		if err := sv.validateKeyFile(host.KeyFile); err != nil {
			result.addViolation("insecure_key_file", RuleTypeFile, SeverityCritical,
				fmt.Sprintf("Key file security issue: %v", err),
				host.KeyFile, "Ensure key file has proper permissions (600)")
		}
	}

	// Validate password security (if used)
	if host.Password != "" {
		result.addWarning("password_authentication",
			"Using password authentication is less secure than key-based authentication",
			host.Password, "Consider using SSH key authentication instead")
	}

	result.Duration = time.Since(startTime)
	result.calculateScore()

	return result
}

// ValidateTask validates task configuration
func (sv *SecurityValidator) ValidateTask(task types.Task) ValidationResult {
	startTime := time.Now()
	result := ValidationResult{
		Valid:      true,
		Violations: make([]SecurityViolation, 0),
		Warnings:   make([]SecurityWarning, 0),
		Timestamp:  startTime,
		Metadata:   make(map[string]interface{}),
	}

	// Validate module
	if !sv.isAllowedModule(task.Module) {
		result.addViolation("module_not_allowed", RuleTypeModule, SeverityHigh,
			fmt.Sprintf("Module %s is not in allowed modules list", task.Module),
			task.Module, "Use an allowed module or add module to allowed_modules configuration")
	}

	// Validate timeout
	if task.Timeout > sv.config.MaxTimeout {
		result.addViolation("timeout_too_long", RuleTypeCustom, SeverityMedium,
			fmt.Sprintf("Task timeout %v exceeds maximum allowed timeout %v", task.Timeout, sv.config.MaxTimeout),
			task.Timeout, fmt.Sprintf("Set timeout to less than %v", sv.config.MaxTimeout))
	}

	// Validate retries
	if task.Retries > sv.config.MaxRetries {
		result.addViolation("too_many_retries", RuleTypeCustom, SeverityMedium,
			fmt.Sprintf("Task retries %d exceeds maximum allowed retries %d", task.Retries, sv.config.MaxRetries),
			task.Retries, fmt.Sprintf("Set retries to less than %d", sv.config.MaxRetries))
	}

	// Validate task arguments based on module
	sv.validateTaskArgs(task, &result)

	// Check for dangerous patterns in task name or arguments
	sv.checkDangerousPatterns(task, &result)

	result.Duration = time.Since(startTime)
	result.calculateScore()

	return result
}

// ValidatePlaybook validates entire playbook
func (sv *SecurityValidator) ValidatePlaybook(playbook types.Playbook) ValidationResult {
	startTime := time.Now()
	result := ValidationResult{
		Valid:      true,
		Violations: make([]SecurityViolation, 0),
		Warnings:   make([]SecurityWarning, 0),
		Timestamp:  startTime,
		Metadata:   make(map[string]interface{}),
	}

	// Validate each play
	for _, play := range playbook.Plays {
		playResult := sv.validatePlay(play)
		result.merge(playResult)
	}

	// Check for privilege escalation patterns
	sv.checkPrivilegeEscalation(playbook, &result)

	// Validate file paths in playbook
	if playbook.FilePath != "" {
		if err := sv.validateFilePath(playbook.FilePath); err != nil {
			result.addViolation("invalid_playbook_path", RuleTypePath, SeverityMedium,
				fmt.Sprintf("Playbook path validation failed: %v", err),
				playbook.FilePath, "Use a path within allowed directories")
		}
	}

	result.Duration = time.Since(startTime)
	result.calculateScore()

	return result
}

// ValidateFile validates file operations
func (sv *SecurityValidator) ValidateFile(path string, operation string) ValidationResult {
	startTime := time.Now()
	result := ValidationResult{
		Valid:      true,
		Violations: make([]SecurityViolation, 0),
		Warnings:   make([]SecurityWarning, 0),
		Timestamp:  startTime,
		Metadata:   make(map[string]interface{}),
	}

	// Validate file path
	if err := sv.validateFilePath(path); err != nil {
		result.addViolation("invalid_file_path", RuleTypePath, SeverityHigh,
			fmt.Sprintf("File path validation failed: %v", err),
			path, "Use a path within allowed directories")
	}

	// Check file size if it exists
	if info, err := os.Stat(path); err == nil {
		if info.Size() > sv.config.MaxFileSize {
			result.addViolation("file_too_large", RuleTypeFile, SeverityMedium,
				fmt.Sprintf("File size %d exceeds maximum allowed size %d", info.Size(), sv.config.MaxFileSize),
				info.Size(), fmt.Sprintf("Use files smaller than %d bytes", sv.config.MaxFileSize))
		}
	}

	// Validate file type
	if !sv.isAllowedFileType(path) {
		result.addViolation("file_type_not_allowed", RuleTypeFile, SeverityMedium,
			fmt.Sprintf("File type for %s is not allowed", path),
			filepath.Ext(path), "Use an allowed file type")
	}

	result.Duration = time.Since(startTime)
	result.calculateScore()

	return result
}

// ValidateVariables validates variable names and values for security issues
func (sv *SecurityValidator) ValidateVariables(variables map[string]interface{}) ValidationResult {
	startTime := time.Now()
	result := ValidationResult{
		Valid:      true,
		Violations: make([]SecurityViolation, 0),
		Warnings:   make([]SecurityWarning, 0),
		Timestamp:  startTime,
		Metadata:   make(map[string]interface{}),
	}

	for name, value := range variables {
		// Check for dangerous variable names
		if strings.Contains(name, "/") || strings.Contains(name, "\\") {
			result.addViolation("dangerous_variable_name", RuleTypeContent, SeverityMedium,
				fmt.Sprintf("Variable name '%s' contains path separators", name),
				name, "Use variable names without path separators")
		}

		// Check for dangerous variable content
		if valueStr, ok := value.(string); ok {
			for _, blocked := range sv.config.BlockedCommands {
				if strings.Contains(valueStr, blocked) {
					result.addViolation("dangerous_variable_content", RuleTypeContent, SeverityHigh,
						fmt.Sprintf("Variable '%s' contains potentially dangerous command: %s", name, blocked),
						valueStr, "Remove dangerous commands from variable values")
				}
			}
		}
	}

	result.Duration = time.Since(startTime)
	result.calculateScore()

	return result
}

// AddRule adds a custom validation rule
func (sv *SecurityValidator) AddRule(rule ValidationRule) {
	sv.rules = append(sv.rules, rule)
}

// RemoveRule removes a validation rule by name
func (sv *SecurityValidator) RemoveRule(name string) {
	for i, rule := range sv.rules {
		if rule.Name == name {
			sv.rules = append(sv.rules[:i], sv.rules[i+1:]...)
			break
		}
	}
}

// GetRules returns all validation rules
func (sv *SecurityValidator) GetRules() []ValidationRule {
	return sv.rules
}

// addDefaultRules adds default security validation rules
func (sv *SecurityValidator) addDefaultRules() {
	// Host validation rules
	sv.AddRule(ValidationRule{
		Name:     "no_localhost",
		Type:     RuleTypeHost,
		Pattern:  `^(localhost|127\.0\.0\.1|::1)$`,
		Message:  "Localhost connections should be avoided in production",
		Severity: SeverityMedium,
		Enabled:  true,
	})

	// Command injection prevention
	sv.AddRule(ValidationRule{
		Name:     "command_injection",
		Type:     RuleTypeCommand,
		Pattern:  `[;&|><$` + "`" + `]`,
		Message:  "Potential command injection detected",
		Severity: SeverityCritical,
		Enabled:  true,
	})

	// Path traversal prevention
	sv.AddRule(ValidationRule{
		Name:     "path_traversal",
		Type:     RuleTypePath,
		Pattern:  `\.\.\/|\.\.\\`,
		Message:  "Path traversal attempt detected",
		Severity: SeverityHigh,
		Enabled:  true,
	})

	// Sensitive file access
	sv.AddRule(ValidationRule{
		Name:     "sensitive_files",
		Type:     RuleTypeFile,
		Pattern:  `\/(etc\/passwd|etc\/shadow|etc\/hosts|root\/\.ssh)`,
		Message:  "Access to sensitive system files detected",
		Severity: SeverityHigh,
		Enabled:  true,
	})
}

// Helper methods for validation

func (sv *SecurityValidator) isAllowedHost(host string) bool {
	if len(sv.config.AllowedHosts) == 0 {
		return true // No restrictions
	}

	for _, allowed := range sv.config.AllowedHosts {
		if host == allowed {
			return true
		}
		// Support wildcard matching
		if matched, _ := filepath.Match(allowed, host); matched {
			return true
		}
	}

	return false
}

func (sv *SecurityValidator) isAllowedPort(port int) bool {
	if len(sv.config.AllowedPorts) == 0 {
		return true // No restrictions
	}

	for _, allowed := range sv.config.AllowedPorts {
		if port == allowed {
			return true
		}
	}

	return false
}

func (sv *SecurityValidator) isAllowedModule(module string) bool {
	if len(sv.config.AllowedModules) == 0 {
		return true // No restrictions
	}

	for _, allowed := range sv.config.AllowedModules {
		if module == allowed {
			return true
		}
	}

	return false
}

func (sv *SecurityValidator) isValidIPAddress(ip string) bool {
	return net.ParseIP(ip) != nil
}

func (sv *SecurityValidator) isValidHostname(hostname string) bool {
	// Basic hostname validation
	if len(hostname) == 0 || len(hostname) > 253 {
		return false
	}

	// Check for valid hostname pattern
	hostnameRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`)
	return hostnameRegex.MatchString(hostname)
}

func (sv *SecurityValidator) validateKeyFile(keyFile string) error {
	info, err := os.Stat(keyFile)
	if err != nil {
		return fmt.Errorf("key file not found: %v", err)
	}

	// Check file permissions (should be 600)
	mode := info.Mode()
	if mode.Perm() != 0600 {
		return fmt.Errorf("key file has insecure permissions: %o (should be 600)", mode.Perm())
	}

	return nil
}

func (sv *SecurityValidator) validateFilePath(path string) error {
	// Check for path traversal patterns in original path
	if strings.Contains(path, "..") {
		return fmt.Errorf("path traversal detected")
	}

	// Clean the path to resolve any traversal attempts
	cleanPath := filepath.Clean(path)

	// Check cleaned path against blocked directories
	for _, blocked := range sv.config.BlockedDirectories {
		if strings.HasPrefix(cleanPath, blocked) {
			return fmt.Errorf("path is in blocked directory: %s", blocked)
		}
	}

	// Check for allowed directories (if specified)
	if len(sv.config.AllowedDirectories) > 0 {
		allowed := false
		for _, allowedDir := range sv.config.AllowedDirectories {
			if strings.HasPrefix(cleanPath, allowedDir) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("path is not in allowed directories")
		}
	}

	return nil
}

func (sv *SecurityValidator) isAllowedFileType(path string) bool {
	if len(sv.config.AllowedFileTypes) == 0 {
		return true // No restrictions
	}

	ext := strings.ToLower(filepath.Ext(path))
	for _, allowed := range sv.config.AllowedFileTypes {
		if ext == strings.ToLower(allowed) {
			return true
		}
	}

	return false
}

func (sv *SecurityValidator) validateTaskArgs(task types.Task, result *ValidationResult) {
	// Module-specific argument validation
	switch task.Module {
	case "shell", "command":
		// Check both "cmd" and "command" arguments
		var cmdStr string
		if cmd, exists := task.Args["cmd"]; exists {
			if str, ok := cmd.(string); ok {
				cmdStr = str
			}
		}
		if cmd, exists := task.Args["command"]; exists {
			if str, ok := cmd.(string); ok {
				cmdStr = str
			}
		}
		if cmdStr != "" {
			sv.validateCommand(cmdStr, result)
		}
	case "copy", "template":
		if dest, exists := task.Args["dest"]; exists {
			if destStr, ok := dest.(string); ok {
				if err := sv.validateFilePath(destStr); err != nil {
					result.addViolation("invalid_dest_path", RuleTypePath, SeverityMedium,
						fmt.Sprintf("Destination path validation failed: %v", err),
						destStr, "Use a valid destination path")
				}
			}
		}
	case "file":
		// Validate file path
		if path, exists := task.Args["path"]; exists {
			if pathStr, ok := path.(string); ok {
				if err := sv.validateFilePath(pathStr); err != nil {
					result.addViolation("invalid_file_path", RuleTypePath, SeverityHigh,
						fmt.Sprintf("File path validation failed: %v", err),
						pathStr, "Use a path within allowed directories")
				}
			}
		}
		// Validate content size
		if content, exists := task.Args["content"]; exists {
			if contentStr, ok := content.(string); ok {
				contentSize := int64(len(contentStr))
				if contentSize > sv.config.MaxFileSize {
					result.addViolation("content_too_large", RuleTypeFile, SeverityHigh,
						fmt.Sprintf("Content size %d bytes exceeds maximum allowed size %d bytes", contentSize, sv.config.MaxFileSize),
						contentSize, fmt.Sprintf("Reduce content size to less than %d bytes", sv.config.MaxFileSize))
				}
			}
		}
	case "user":
		// Check for system user modifications
		if name, exists := task.Args["name"]; exists {
			if nameStr, ok := name.(string); ok {
				systemUsers := []string{"root", "daemon", "bin", "sys", "sync", "games", "man", "lp", "mail", "news", "uucp", "proxy", "www-data", "backup", "list", "irc", "gnats", "nobody"}
				for _, sysUser := range systemUsers {
					if nameStr == sysUser {
						result.addViolation("system_user_modification", RuleTypeUser, SeverityCritical,
							fmt.Sprintf("Attempting to modify system user: %s", nameStr),
							nameStr, "Do not modify system users")
					}
				}
			}
		}
		// Check for UID 0 (root equivalent)
		if uid, exists := task.Args["uid"]; exists {
			var uidInt int
			switch v := uid.(type) {
			case int:
				uidInt = v
			case float64:
				uidInt = int(v)
			case string:
				_, _ = fmt.Sscanf(v, "%d", &uidInt)
			}
			if uidInt == 0 {
				result.addViolation("root_uid_assignment", RuleTypeUser, SeverityCritical,
					"Attempting to create user with UID 0 (root equivalent)",
					uid, "Use a non-zero UID for non-root users")
			}
		}
	case "group":
		// Check for system group modifications
		if name, exists := task.Args["name"]; exists {
			if nameStr, ok := name.(string); ok {
				systemGroups := []string{"root", "daemon", "bin", "sys", "adm", "tty", "disk", "lp", "mail", "news", "uucp", "man", "proxy", "kmem", "dialout", "fax", "voice", "cdrom", "floppy", "tape", "sudo", "audio", "dip", "www-data", "backup"}
				for _, sysGroup := range systemGroups {
					if nameStr == sysGroup {
						result.addViolation("system_group_modification", RuleTypeGroup, SeverityCritical,
							fmt.Sprintf("Attempting to modify system group: %s", nameStr),
							nameStr, "Do not modify system groups")
					}
				}
			}
		}
		// Check for GID 0 (root equivalent)
		if gid, exists := task.Args["gid"]; exists {
			var gidInt int
			switch v := gid.(type) {
			case int:
				gidInt = v
			case float64:
				gidInt = int(v)
			case string:
				_, _ = fmt.Sscanf(v, "%d", &gidInt)
			}
			if gidInt == 0 {
				result.addViolation("root_gid_assignment", RuleTypeGroup, SeverityCritical,
					"Attempting to create group with GID 0 (root equivalent)",
					gid, "Use a non-zero GID for non-root groups")
			}
		}
	}
}

func (sv *SecurityValidator) validateCommand(command string, result *ValidationResult) {
	// Check for blocked commands
	for _, blocked := range sv.config.BlockedCommands {
		if strings.Contains(command, blocked) {
			result.addViolation("blocked_command", RuleTypeCommand, SeverityHigh,
				fmt.Sprintf("Command contains blocked pattern: %s", blocked),
				command, "Remove blocked command pattern")
		}
	}

	// Check for command substitution patterns
	if strings.Contains(command, "$(") || strings.Contains(command, "`") {
		result.addViolation("command_substitution", RuleTypeCommand, SeverityCritical,
			"Command contains command substitution pattern",
			command, "Remove command substitution or use separate tasks")
		return
	}

	// Check for pipe to shell interpreters
	pipeToShellPatterns := []string{"| sh", "| bash", "| zsh", "| ksh", "| csh", "| tcsh", "|sh", "|bash", "|zsh"}
	for _, pattern := range pipeToShellPatterns {
		if strings.Contains(command, pattern) {
			result.addViolation("pipe_to_shell", RuleTypeCommand, SeverityCritical,
				"Command pipes output to shell interpreter",
				command, "Avoid piping to shell interpreters")
			return
		}
	}

	// Count command chaining operators to detect complex chaining
	semicolonCount := strings.Count(command, ";")
	andCount := strings.Count(command, "&&")
	orCount := strings.Count(command, "||")
	pipeCount := strings.Count(command, "|")

	// Calculate total operator count (excluding simple cases)
	totalOperators := semicolonCount + andCount + orCount

	// Complex chaining: multiple different operators or semicolons (always dangerous)
	if semicolonCount > 0 {
		result.addViolation("command_chaining_semicolon", RuleTypeCommand, SeverityCritical,
			"Command uses semicolon for command chaining",
			command, "Use separate tasks instead of semicolon chaining")
		return
	}

	// Complex chaining: mixing && and || operators, or more than 2 operators total
	hasAnd := andCount > 0
	hasOr := orCount > 0
	hasPipe := pipeCount > 0

	operatorTypes := 0
	if hasAnd {
		operatorTypes++
	}
	if hasOr {
		operatorTypes++
	}
	if hasPipe {
		operatorTypes++
	}

	// Allow simple chaining (single operator type, max 2 operators)
	// Block complex chaining (multiple operator types or more than 2 operators)
	if operatorTypes > 1 || totalOperators > 2 {
		result.addViolation("complex_command_chaining", RuleTypeCommand, SeverityHigh,
			"Command uses complex chaining with multiple operator types or too many operators",
			command, "Simplify command or use separate tasks")
		return
	}
}

func (sv *SecurityValidator) checkDangerousPatterns(task types.Task, result *ValidationResult) {
	// Check task name and arguments for dangerous patterns
	criticalPatterns := []string{"rm -rf", "dd if=", ":(){ :|:& };:", "mkfs", "fdisk"}
	suspiciousPatterns := []string{"curl", "wget"}

	taskStr := fmt.Sprintf("%s %v", task.Name, task.Args)
	taskStrLower := strings.ToLower(taskStr)

	// Critical patterns should be violations
	for _, pattern := range criticalPatterns {
		if strings.Contains(taskStrLower, strings.ToLower(pattern)) {
			result.addViolation("dangerous_command_pattern", RuleTypeCommand, SeverityCritical,
				fmt.Sprintf("Task contains dangerous pattern: %s", pattern),
				pattern, "Remove dangerous command or use safer alternatives")
		}
	}

	// Suspicious patterns should be warnings
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(taskStrLower, strings.ToLower(pattern)) {
			result.addWarning("suspicious_pattern",
				fmt.Sprintf("Task contains potentially suspicious pattern: %s", pattern),
				pattern, "Review task for security implications")
		}
	}
}

func (sv *SecurityValidator) validatePlay(play types.Play) ValidationResult {
	result := ValidationResult{
		Valid:      true,
		Violations: make([]SecurityViolation, 0),
		Warnings:   make([]SecurityWarning, 0),
		Timestamp:  time.Now(),
		Metadata:   make(map[string]interface{}),
	}

	// Validate each task in the play
	for _, task := range play.Tasks {
		taskResult := sv.ValidateTask(task)
		result.merge(taskResult)
	}

	// Check for privilege escalation
	if play.Become {
		result.addWarning("privilege_escalation",
			"Play uses privilege escalation (become: true)",
			play.Become, "Ensure privilege escalation is necessary and properly secured")
	}

	return result
}

func (sv *SecurityValidator) checkPrivilegeEscalation(playbook types.Playbook, result *ValidationResult) {
	for _, play := range playbook.Plays {
		if play.Become {
			result.addWarning("playbook_privilege_escalation",
				fmt.Sprintf("Play '%s' uses privilege escalation", play.Name),
				play.Name, "Review privilege escalation requirements")
		}
	}
}

// Helper methods for ValidationResult

func (vr *ValidationResult) addViolation(rule string, ruleType RuleType, severity Severity, message string, value interface{}, suggestion string) {
	vr.Valid = false
	vr.Violations = append(vr.Violations, SecurityViolation{
		Rule:       rule,
		Type:       ruleType,
		Severity:   severity,
		Message:    message,
		Value:      value,
		Suggestion: suggestion,
		Timestamp:  time.Now(),
	})
}

func (vr *ValidationResult) addWarning(rule, message string, value interface{}, suggestion string) {
	vr.Warnings = append(vr.Warnings, SecurityWarning{
		Rule:       rule,
		Message:    message,
		Value:      value,
		Suggestion: suggestion,
		Timestamp:  time.Now(),
	})
}

func (vr *ValidationResult) merge(other ValidationResult) {
	if !other.Valid {
		vr.Valid = false
	}
	vr.Violations = append(vr.Violations, other.Violations...)
	vr.Warnings = append(vr.Warnings, other.Warnings...)
}

func (vr *ValidationResult) calculateScore() {
	vr.MaxScore = 100
	vr.Score = vr.MaxScore

	// Deduct points based on violations
	for _, violation := range vr.Violations {
		switch violation.Severity {
		case SeverityCritical:
			vr.Score -= 25
		case SeverityHigh:
			vr.Score -= 15
		case SeverityMedium:
			vr.Score -= 10
		case SeverityLow:
			vr.Score -= 5
		}
	}

	// Deduct points for warnings (less severe)
	vr.Score -= len(vr.Warnings) * 2

	if vr.Score < 0 {
		vr.Score = 0
	}
}

// SecurityAuditor provides security auditing capabilities
type SecurityAuditor struct {
	validator *SecurityValidator
	auditLog  []AuditEntry
	mutex     sync.RWMutex
}

// AuditEntry represents a security audit entry
type AuditEntry struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Type      string                 `json:"type"`
	Action    string                 `json:"action"`
	Resource  string                 `json:"resource"`
	User      string                 `json:"user"`
	Result    ValidationResult       `json:"result"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// NewSecurityAuditor creates a new security auditor
func NewSecurityAuditor(validator *SecurityValidator) *SecurityAuditor {
	return &SecurityAuditor{
		validator: validator,
		auditLog:  make([]AuditEntry, 0),
	}
}

// AuditHost audits host access
func (sa *SecurityAuditor) AuditHost(host types.Host, user string) AuditEntry {
	result := sa.validator.ValidateHost(host)

	entry := AuditEntry{
		ID:        generateAuditID(),
		Timestamp: time.Now(),
		Type:      "host_access",
		Action:    "validate",
		Resource:  host.Address,
		User:      user,
		Result:    result,
		Metadata: map[string]interface{}{
			"host_name": host.Name,
			"port":      host.Port,
		},
	}

	sa.mutex.Lock()
	sa.auditLog = append(sa.auditLog, entry)
	sa.mutex.Unlock()

	return entry
}

// AuditTask audits task execution
func (sa *SecurityAuditor) AuditTask(task types.Task, user string) AuditEntry {
	result := sa.validator.ValidateTask(task)

	entry := AuditEntry{
		ID:        generateAuditID(),
		Timestamp: time.Now(),
		Type:      "task_execution",
		Action:    "validate",
		Resource:  task.Name,
		User:      user,
		Result:    result,
		Metadata: map[string]interface{}{
			"module": task.Module,
			"args":   task.Args,
		},
	}

	sa.mutex.Lock()
	sa.auditLog = append(sa.auditLog, entry)
	sa.mutex.Unlock()

	return entry
}

// GetAuditLog returns the audit log
func (sa *SecurityAuditor) GetAuditLog() []AuditEntry {
	sa.mutex.RLock()
	defer sa.mutex.RUnlock()

	log := make([]AuditEntry, len(sa.auditLog))
	copy(log, sa.auditLog)
	return log
}

// generateAuditID generates a unique audit ID using cryptographically secure random bytes
func generateAuditID() string {
	// Use crypto/rand for secure random ID generation
	b := make([]byte, 8) // 8 bytes = 16 hex characters
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID if random generation fails
		return fmt.Sprintf("%016x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// SetMaxFileSize sets the maximum file size
func (sv *SecurityValidator) SetMaxFileSize(size int64) {
	sv.config.MaxFileSize = size
}

// AddBlockedPath adds a path to the blocked directories list
func (sv *SecurityValidator) AddBlockedPath(path string) {
	sv.config.BlockedDirectories = append(sv.config.BlockedDirectories, path)
}

// AddDangerousPattern adds a dangerous command pattern
func (sv *SecurityValidator) AddDangerousPattern(pattern string) error {
	// Validate the regex pattern
	_, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex pattern: %v", err)
	}

	sv.config.BlockedCommands = append(sv.config.BlockedCommands, pattern)
	return nil
}
