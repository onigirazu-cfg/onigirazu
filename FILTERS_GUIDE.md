# Onigirazu Filters Guide

Complete guide to using and creating filter plugins in Onigirazu.

## 📋 Table of Contents

- [What are Filters?](#what-are-filters)
- [Built-in Filters](#built-in-filters)
  - [String Manipulation](#string-manipulation)
  - [Collection Operations](#collection-operations)
  - [Conditional Operations](#conditional-operations)
- [Using Filters in Templates](#using-filters-in-templates)
- [Creating Custom Filters](#creating-custom-filters)
- [Advanced Filter Development](#advanced-filter-development)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## What are Filters?

Filters are plugin functions that transform data in Onigirazu templates. They allow you to:

- **Transform text**: Convert case, trim whitespace, replace content
- **Manipulate collections**: Join arrays, split strings, get length
- **Provide defaults**: Supply fallback values for empty/null data
- **Extend functionality**: Create custom filters for domain-specific transformations

### Filter Function Signature

```go
type FilterFunc func(input interface{}, args ...interface{}) (interface{}, error)
```

- **input**: The data to transform (required)
- **args**: Variable arguments for the filter (optional)
- **Return**: Transformed data and error (if any)

### Filter Architecture

```
┌─────────────────────────────────────────┐
│      Onigirazu Template Engine          │
│  ┌───────────────────────────────────┐  │
│  │  Template Parser                  │  │
│  │  {{ variable | filter(args) }}    │  │
│  └──────────────┬────────────────────┘  │
│                 │                        │
│  ┌──────────────▼────────────────────┐  │
│  │  Filter Plugin Manager            │  │
│  │  - BuiltinFiltersPlugin          │  │
│  │  - Custom Filter Plugins         │  │
│  └──────────────┬────────────────────┘  │
│                 │                        │
│  ┌──────────────▼────────────────────┐  │
│  │  Filter Execution                │  │
│  │  Transform Data & Return Result  │  │
│  └───────────────────────────────────┘  │
└─────────────────────────────────────────┘
```

## Built-in Filters

Onigirazu provides 9 built-in filters out of the box:

### String Manipulation

#### 1. `upper` - Convert to Uppercase

Converts a string to uppercase.

**Signature**: `upper`
**Arguments**: None
**Input Type**: String
**Output Type**: String

**Example**:

```yaml
vars:
  app_name: "myapp"

tasks:
  - name: Use upper filter
    debug:
      msg: "{{ app_name | upper }}"
    # Output: MYAPP
```

#### 2. `lower` - Convert to Lowercase

Converts a string to lowercase.

**Signature**: `lower`
**Arguments**: None
**Input Type**: String
**Output Type**: String

**Example**:

```yaml
vars:
  app_name: "MyApp"

tasks:
  - name: Use lower filter
    debug:
      msg: "{{ app_name | lower }}"
    # Output: myapp
```

#### 3. `title` - Convert to Title Case

Converts a string to title case (each word capitalized).

**Signature**: `title`
**Arguments**: None
**Input Type**: String
**Output Type**: String

**Example**:

```yaml
vars:
  description: "hello world from onigirazu"

tasks:
  - name: Use title filter
    debug:
      msg: "{{ description | title }}"
    # Output: Hello World From Onigirazu
```

#### 4. `trim` - Remove Whitespace

Removes leading and trailing whitespace from a string.

**Signature**: `trim`
**Arguments**: None
**Input Type**: String
**Output Type**: String

**Example**:

```yaml
vars:
  user_input: "  john doe  "

tasks:
  - name: Use trim filter
    debug:
      msg: "'{{ user_input | trim }}'"
    # Output: 'john doe'
```

#### 5. `replace` - Replace Substring

Replaces all occurrences of a substring with another substring.

**Signature**: `replace(old, new)`
**Arguments**:

- `old` (string): Substring to find
- `new` (string): Replacement substring
**Input Type**: String
**Output Type**: String

**Example**:

```yaml
vars:
  path: "/home/user/project"

tasks:
  - name: Use replace filter
    debug:
      msg: "{{ path | replace('/home', '/opt') }}"
    # Output: /opt/user/project
```

**Advanced Example**:

```yaml
vars:
  config: "database_host=localhost;database_port=5432"

tasks:
  - name: Replace config values
    debug:
      msg: "{{ config | replace('localhost', 'prod-db.example.com') }}"
    # Output: database_host=prod-db.example.com;database_port=5432
```

### Collection Operations

#### 6. `length` - Get Length

Returns the length of a string, slice, or map.

**Signature**: `length`
**Arguments**: None
**Input Types**: String, Array/Slice, Object/Map
**Output Type**: Integer

**Example**:

```yaml
vars:
  app_name: "myapp"
  users: ["alice", "bob", "charlie"]
  config:
    host: localhost
    port: 5432

tasks:
  - name: String length
    debug:
      msg: "Name length: {{ app_name | length }}"
    # Output: Name length: 5

  - name: Array length
    debug:
      msg: "User count: {{ users | length }}"
    # Output: User count: 3

  - name: Map/Object length
    debug:
      msg: "Config fields: {{ config | length }}"
    # Output: Config fields: 2
```

#### 7. `join` - Join Array Elements

Joins array elements into a single string with a separator.

**Signature**: `join(separator)`
**Arguments**:

- `separator` (string): String to join elements with
**Input Types**: Array/Slice
**Output Type**: String

**Example**:

```yaml
vars:
  users: ["alice", "bob", "charlie"]
  tags: ["production", "critical", "monitored"]

tasks:
  - name: Join users
    debug:
      msg: "Users: {{ users | join(', ') }}"
    # Output: Users: alice, bob, charlie

  - name: Join tags with semicolon
    debug:
      msg: "Tags: {{ tags | join('; ') }}"
    # Output: Tags: production; critical; monitored

  - name: Create CSV
    debug:
      msg: "{{ users | join(',') }}"
    # Output: alice,bob,charlie
```

#### 8. `split` - Split String into Array

Splits a string into an array using a separator.

**Signature**: `split(separator)`
**Arguments**:

- `separator` (string): Delimiter to split by
**Input Type**: String
**Output Type**: Array/Slice

**Example**:

```yaml
vars:
  csv_data: "alice,bob,charlie"
  path_string: "/home/user/projects/myapp"

tasks:
  - name: Split CSV
    debug:
      msg: "{{ csv_data | split(',') }}"
    # Output: ["alice", "bob", "charlie"]

  - name: Split path
    debug:
      msg: "{{ path_string | split('/') }}"
    # Output: ["", "home", "user", "projects", "myapp"]

  - name: Chain filters
    debug:
      msg: "{{ 'a,b,c' | split(',') | join(' -> ') }}"
    # Output: a -> b -> c
```

### Conditional Operations

#### 9. `default` - Provide Default Value

Returns the input if it's not empty/nil, otherwise returns the default value.

**Signature**: `default(default_value)`
**Arguments**:

- `default_value` (any): Value to use if input is empty
**Input Type**: Any
**Output Type**: Same as input or default value type

**Example**:

```yaml
vars:
  user_name: ""
  user_age: null
  user_role: "admin"

tasks:
  - name: Use default for empty string
    debug:
      msg: "Name: {{ user_name | default('anonymous') }}"
    # Output: Name: anonymous

  - name: Use default for null
    debug:
      msg: "Age: {{ user_age | default(0) }}"
    # Output: Age: 0

  - name: Use default for non-empty
    debug:
      msg: "Role: {{ user_role | default('user') }}"
    # Output: Role: admin

  - name: Default with variable
    debug:
      msg: "{{ undefined_var | default(enabled_by_default) }}"
    # Output: Value of enabled_by_default
```

## Using Filters in Templates

### Basic Filter Usage

```yaml
vars:
  app_name: "myapp"

tasks:
  - name: Single filter
    debug:
      msg: "{{ app_name | upper }}"
```

### Chaining Filters

Filters can be chained together:

```yaml
vars:
  user_input: "  hello world  "

tasks:
  - name: Chain multiple filters
    debug:
      msg: "{{ user_input | trim | upper }}"
    # Output: HELLO WORLD

  - name: Complex chain
    debug:
      msg: "{{ 'a,b,c' | split(',') | join(' - ') | upper }}"
    # Output: A - B - C
```

### Filters with Arguments

```yaml
vars:
  config: "host=old;port=5432"
  tags: ["web", "api", "db"]

tasks:
  - name: Filter with arguments
    debug:
      msg: "{{ config | replace('old', 'new') }}"
    # Output: host=new;port=5432

  - name: Filter with multiple arguments
    debug:
      msg: "{{ 'hello/world' | replace('/', '-') }}"
    # Output: hello-world

  - name: Combine chaining and arguments
    debug:
      msg: "{{ tags | join('-') | upper }}"
    # Output: WEB-API-DB
```

### Using Filters in Different Contexts

#### In Variable Definitions

```yaml
vars:
  app_name: "MyApp"
  app_name_lower: "{{ app_name | lower }}"
```

#### In Task Names

```yaml
tasks:
  - name: "{{ app_name | upper }} - Deploy Service"
    debug:
      msg: "Deploying {{ app_name | lower }}..."
```

#### In Conditions

```yaml
tasks:
  - name: Conditional task using filter
    debug:
      msg: "Environment is important"
    when: environment | upper == "PRODUCTION"
```

#### In Task Arguments

```yaml
tasks:
  - name: Use filter in module arguments
    shell:
      cmd: "echo {{ message | upper }}"
```

## Creating Custom Filters

### Basic Custom Filter

```go
package main

import (
    "context"
    "fmt"
    "strings"

    "github.com/onigirazu-cfg/onigirazu/internal/plugins"
)

// ReverseFilterPlugin provides a reverse string filter
type ReverseFilterPlugin struct {
    *plugins.BaseFilterPlugin
}

// NewReverseFilterPlugin creates a new reverse filter plugin
func NewReverseFilterPlugin() *ReverseFilterPlugin {
    plugin := &ReverseFilterPlugin{
        BaseFilterPlugin: plugins.NewBaseFilterPlugin(
            "reverse_filter",
            "1.0.0",
            "Reverses strings and arrays",
        ),
    }

    // Register filter function
    plugin.AddFilter("reverse", reverseFilter)

    return plugin
}

// reverseFilter implementation
func reverseFilter(input interface{}, args ...interface{}) (interface{}, error) {
    switch v := input.(type) {
    case string:
        runes := []rune(v)
        for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
            runes[i], runes[j] = runes[j], runes[i]
        }
        return string(runes), nil

    case []interface{}:
        reversed := make([]interface{}, len(v))
        for i, val := range v {
            reversed[len(v)-1-i] = val
        }
        return reversed, nil

    default:
        return nil, fmt.Errorf("reverse filter expects string or array, got %T", input)
    }
}

// NewPlugin is the entry point for plugin loader
func NewPlugin() plugins.Plugin {
    return NewReverseFilterPlugin()
}
```

### Filter Plugin with Configuration

```go
package main

import (
    "context"
    "fmt"
    "strings"

    "github.com/onigirazu-cfg/onigirazu/internal/plugins"
)

type PrefixFilterPlugin struct {
    *plugins.BaseFilterPlugin
    prefix string
}

func NewPrefixFilterPlugin() *PrefixFilterPlugin {
    plugin := &PrefixFilterPlugin{
        BaseFilterPlugin: plugins.NewBaseFilterPlugin(
            "prefix_filter",
            "1.0.0",
            "Adds prefix to strings",
        ),
    }

    plugin.AddFilter("prefix", plugin.prefixFilter)
    return plugin
}

// Initialize loads configuration
func (p *PrefixFilterPlugin) Initialize(ctx context.Context, config map[string]interface{}) error {
    if prefixVal, ok := config["prefix"].(string); ok {
        p.prefix = prefixVal
    }
    return nil
}

func (p *PrefixFilterPlugin) prefixFilter(input interface{}, args ...interface{}) (interface{}, error) {
    str, ok := input.(string)
    if !ok {
        return nil, fmt.Errorf("prefix filter expects string input")
    }

    prefix := p.prefix
    if len(args) > 0 {
        if argPrefix, ok := args[0].(string); ok {
            prefix = argPrefix
        }
    }

    return prefix + str, nil
}

func NewPlugin() plugins.Plugin {
    return NewPrefixFilterPlugin()
}
```

### Testing Custom Filters

```go
package main

import (
    "testing"
)

func TestReverseFilter(t *testing.T) {
    result, err := reverseFilter("hello", )
    if err != nil {
        t.Fatalf("Unexpected error: %v", err)
    }
    if result != "olleh" {
        t.Errorf("Expected 'olleh', got '%v'", result)
    }
}

func TestReverseFilterArray(t *testing.T) {
    input := []interface{}{"a", "b", "c"}
    result, err := reverseFilter(input)
    if err != nil {
        t.Fatalf("Unexpected error: %v", err)
    }
    arr := result.([]interface{})
    if len(arr) != 3 || arr[0] != "c" || arr[1] != "b" || arr[2] != "a" {
        t.Errorf("Unexpected reverse result: %v", arr)
    }
}
```

## Advanced Filter Development

### Using Context in Filters

```go
func contextAwareFilter(input interface{}, args ...interface{}) (interface{}, error) {
    // Context can be used for:
    // - Caching data
    // - Accessing shared resources
    // - Rate limiting filter calls
    // - Logging operations

    str, ok := input.(string)
    if !ok {
        return nil, fmt.Errorf("expected string")
    }

    // Your filter logic
    return strings.ToUpper(str), nil
}
```

### Error Handling

```go
func safeFilter(input interface{}, args ...interface{}) (interface{}, error) {
    // Validate input
    str, ok := input.(string)
    if !ok {
        return nil, fmt.Errorf(
            "invalid input type: expected string, got %T",
            input,
        )
    }

    // Validate arguments
    if len(args) < 1 {
        return nil, fmt.Errorf("filter requires at least 1 argument")
    }

    arg, ok := args[0].(string)
    if !ok {
        return nil, fmt.Errorf(
            "invalid argument type: expected string, got %T",
            args[0],
        )
    }

    // Perform operation
    return fmt.Sprintf("%s:%s", str, arg), nil
}
```

### Performance Optimization

```go
// Use type assertion caching
func optimizedFilter(input interface{}, args ...interface{}) (interface{}, error) {
    // Fast path for string type
    if str, ok := input.(string); ok {
        return strings.ToUpper(str), nil
    }

    // Handle other types
    switch v := input.(type) {
    case []string:
        // Optimized for pre-allocated slice
        result := make([]string, len(v))
        for i, s := range v {
            result[i] = strings.ToUpper(s)
        }
        return result, nil
    default:
        return nil, fmt.Errorf("unsupported type: %T", input)
    }
}
```

## Best Practices

### 1. **Input Validation**

Always validate filter input types:

```go
func myFilter(input interface{}, args ...interface{}) (interface{}, error) {
    str, ok := input.(string)
    if !ok {
        return nil, fmt.Errorf("expected string, got %T", input)
    }
    // Process string
}
```

### 2. **Clear Error Messages**

Provide descriptive errors:

```go
return nil, fmt.Errorf("replace filter requires 2 arguments (old, new), got %d", len(args))
```

### 3. **Type Safety**

Handle different input types gracefully:

```go
func flexibleFilter(input interface{}, args ...interface{}) (interface{}, error) {
    switch v := input.(type) {
    case string:
        return handleString(v)
    case []interface{}:
        return handleArray(v)
    case map[string]interface{}:
        return handleMap(v)
    default:
        return nil, fmt.Errorf("unsupported type")
    }
}
```

### 4. **Documentation in Code**

Include clear documentation:

```go
// UpperFilter converts string to uppercase
// Input: string
// Args: none
// Returns: uppercase string, error if input is not string
func UpperFilter(input interface{}, args ...interface{}) (interface{}, error) {
    // Implementation
}
```

### 5. **Consider Edge Cases**

```go
func robustFilter(input interface{}, args ...interface{}) (interface{}, error) {
    // Handle nil input
    if input == nil {
        return "", nil
    }

    // Handle empty string
    if str, ok := input.(string); ok && str == "" {
        return input, nil
    }

    // Handle edge cases in arguments
    if len(args) == 0 {
        return input, nil
    }

    // Continue with normal processing
    return processFilter(input, args...)
}
```

## Troubleshooting

### Filter Not Available in Templates

**Problem**: Template engine says filter doesn't exist.

**Solutions**:

1. Verify plugin is registered with plugin manager
2. Check filter name matches exactly (case-sensitive)
3. Ensure template engine has plugin manager set
4. Check plugin initialization succeeded

### Type Mismatch Error

**Problem**: "expected string, got int"

**Solution**: Ensure input type matches filter requirements:

```yaml
# Wrong: numeric values need conversion
msg: "{{ 123 | upper }}"  # ERROR

# Correct: convert to string first
msg: "{{ '123' | upper }}"  # OK
```

### Argument Parsing Issues

**Problem**: Filter arguments not parsed correctly.

**Debug**: Test with explicit string literals:

```yaml
# Test with literals
msg: "{{ 'hello' | replace('h', 'H') }}"  # Works

# If variables fail, check variable type
msg: "{{ text | replace(old_char, new_char) }}"
```

### Performance Problems

**Problem**: Filter is slow with large datasets.

**Solutions**:

1. Cache filter results in variables
2. Minimize filter chaining
3. Use optimized filter implementations
4. Move complex logic to module level

```yaml
# Not optimal: recalculates every task
- debug: msg: "{{ big_data | process }}"
- copy: content: "{{ big_data | process }}"

# Better: cache result
- set_fact:
    processed_data: "{{ big_data | process }}"
- debug: msg: "{{ processed_data }}"
- copy: content: "{{ processed_data }}"
```

## API Reference

### FilterFunc Interface

```go
type FilterFunc func(input interface{}, args ...interface{}) (interface{}, error)
```

### BaseFilterPlugin

```go
type BaseFilterPlugin struct {
    // Unexported fields
}

// Methods
func NewBaseFilterPlugin(name, version, description string) *BaseFilterPlugin
func (p *BaseFilterPlugin) GetName() string
func (p *BaseFilterPlugin) GetType() PluginType
func (p *BaseFilterPlugin) GetVersion() string
func (p *BaseFilterPlugin) GetDescription() string
func (p *BaseFilterPlugin) Initialize(ctx context.Context, config map[string]interface{}) error
func (p *BaseFilterPlugin) Cleanup(ctx context.Context) error
func (p *BaseFilterPlugin) GetFilters() map[string]FilterFunc
func (p *BaseFilterPlugin) AddFilter(name string, fn FilterFunc)
func (p *BaseFilterPlugin) RemoveFilter(name string)
```

## See Also

- [PLUGIN_INTEGRATION.md](PLUGIN_INTEGRATION.md) - General plugin integration guide
- [CALLBACKS_GUIDE.md](CALLBACKS_GUIDE.md) - Callback plugins guide
- [examples/plugins/](../examples/plugins/) - Example plugins
