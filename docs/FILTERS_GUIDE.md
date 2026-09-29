# Onigirazu Filters Guide

Using the built-in filters, tests and lookups of Onigirazu expressions, and writing filter plugins.

## 📋 Table of Contents

- [What are Filters?](#what-are-filters)
- [Built-in Filters](#built-in-filters)
  - [String Manipulation](#string-manipulation)
  - [Collection Operations](#collection-operations)
  - [Conditional Operations](#conditional-operations)
  - [Filter Reference](#filter-reference)
  - [Tests](#tests)
  - [Lookups](#lookups)
  - [Expression Syntax](#expression-syntax)
- [Using Filters in Templates](#using-filters-in-templates)
- [Creating Custom Filters](#creating-custom-filters)
- [Advanced Filter Development](#advanced-filter-development)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## What are Filters?

Filters transform values inside `{{ }}` blocks, `when`/`until`/`changed_when`/`failed_when` conditions and loop expressions: `{{ name | upper }}`, `{{ users | map(attribute='name') | join(', ') }}`.

The same expression evaluator handles all of these places, so every built-in filter works in task arguments, conditions and template files alike. Filter plugins are different; see [Creating Custom Filters](#creating-custom-filters).

## Built-in Filters

The most common filters are described below; the [Filter Reference](#filter-reference) lists all of them.

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

  - name: Leave an argument out when the variable is not set
    docker_container:
      name: app
      memory: "{{ app_memory | default(omit) }}"
```

A module argument that renders to `omit` is left out, in nested options and
list items too.

### Filter Reference

| Group | Filters |
|-------|---------|
| Strings | `upper`, `lower`, `title`, `capitalize`, `trim`, `replace(old, new)`, `split(sep)`, `quote` (shell quoting), `basename`, `dirname` |
| Regular expressions | `regex_replace(pattern, repl)` (`\1` back references), `regex_search(pattern)` (match or none), `regex_findall(pattern)`, `regex_escape` |
| Conversion | `int`, `float`, `string`, `bool`, `list`, `abs`, `round` |
| Data formats | `to_json`, `to_nice_json`, `from_json`, `to_yaml`, `to_nice_yaml`, `from_yaml`, `b64encode`, `b64decode` |
| Lists | `length`/`count`, `first`, `last`, `join(sep)`, `unique`, `sort`, `reverse`, `flatten`, `sum`, `min`, `max`, `range(n)`, `zip(other...)`, `product(other...)` |
| Sets | `intersect(other)`, `difference(other)`, `union(other)`, `symmetric_difference(other)` (order of the first list, no duplicates) |
| Selection | `select(test, arg)`, `reject(test, arg)`, `selectattr(attr, test, arg)`, `rejectattr(attr, test, arg)`; without a test an item is kept when it is truthy |
| Mapping | `map(attribute='x')`, `map(attribute='x', default=d)`, `map('filter', args...)`, `map('extract', container, key)` |
| Dictionaries | `keys`, `values`, `dict2items`, `items2dict`, `combine(other, ...)` (shallow merge) |
| Random | `random` (an item of a list, or for a number N one of `start`, `start+step`, ... below N: `60 \| random(seed=inventory_hostname)`), `shuffle`; the same `seed` gives the same result, but not the numbers Ansible would pick |
| Other | `default(value)`, `mandatory` (error when undefined), `ternary(if_true, if_false)`, `password_hash('sha512' or 'sha256', salt)` |

`default` (alias `d`) also replaces a missing list item or key: `result.stdout_lines[0] | default('')`.
Not available: `hash`, `default(value, true)` (the second argument is ignored; `default` always replaces undefined, null and `""`).

```yaml
- debug:
    msg: "{{ users | selectattr('admin') | map(attribute='name') | join(', ') }}"

- debug:
    msg: "{{ groups['web'] | map('extract', hostvars, 'ansible_host') | list }}"

- user:
    name: deploy
    password: "{{ deploy_password | password_hash('sha512') }}"
```

### Tests

Tests work in any expression (`x is test(args)`, `x is not test(args)`) and in `select`, `reject`, `selectattr` and `rejectattr`: `defined`, `undefined`, `none`, `truthy`, `falsy`, `equalto`/`==`/`eq`, `!=`/`ne`, `>`, `<`, `>=`, `<=` (also `gt`, `lt`, `ge`, `le`), `in`, `contains`, `match`/`search`/`regex`, `string`, `number`.

Ansible tests: task results `success`/`succeeded`, `failed`/`failure`, `changed`, `skipped`, `unreachable` (`until: r is success`); `version(other, op)` (loose comparison as Ansible's default: `'22.04' is version('20.04', '>=')`); `mapping`, `sequence`/`iterable`, `boolean`, `integer`, `float`, `lower`, `upper`, `even`, `odd`, `divisibleby(n)`, `subset(list)`, `superset(list)`; paths on the control machine `exists`, `file`, `directory`, `link`, `abs`.

```yaml
when: my_var is defined and my_var | length > 0
when: version is match('^2\\.')
loop: "{{ packages | select('match', '^python3-') | list }}"
```

### Lookups

`lookup(plugin, terms...)` returns one value (plugins can be written `ansible.builtin.env` too) (several are joined with commas); `query(...)`/`q(...)` returns a list. They run on the control machine; relative paths start at the playbook directory.

| Plugin | Returns |
|--------|---------|
| `env` | Environment variable |
| `template` | A template file rendered with the task's variables |
| `file` | File content |
| `pipe` | Output of a shell command |
| `lines` | Output of a command, one item per line |
| `fileglob` | Files matching a pattern |
| `first_found` | First existing path of a list |
| `dict` | `{key, value}` items of a dictionary |
| `items`, `list` | The terms as a list |
| `nested` | Every combination of the lists |
| `together` | The lists zipped by position |
| `subelements` | `[element, subitem]` for a list of dicts and a key (`{skip_missing: true}` as third term) |
| `indexed_items` | `[index, item]` |
| `random_choice` | One of the terms |

```yaml
key: "{{ lookup('file', 'files/id_ed25519.pub') }}"
loop: "{{ query('fileglob', 'files/conf.d/*.conf') }}"
```

### Expression Syntax

- Inline if: `{{ 'big' if n > 3 else 'small' }}`
- Python methods: strings `split`, `rsplit`, `strip`, `lstrip`, `rstrip`, `lower`, `upper`, `title`, `capitalize`, `startswith`, `endswith`, `replace`, `find`, `count`, `join`, `splitlines`, `isdigit`; dicts `get`, `items`, `keys`, `values`; lists `index`, `count` (`{{ opts.split() }}`, `{{ host.split('.')[0] }}`)
- Literal braces: `{{ '{{' }} .Field {{ '}}' }}` gives `{{ .Field }}` (docker `--format`); braces inside quotes and dict literals do not end a `{{ }}` block
- String concatenation: `{{ name ~ '-' ~ version }}`, also inside parentheses and lists, as inline ifs (`('yes' if x else 'no')`) (`when: ('version ' ~ v) not in out.stdout`); `+` adds numbers and concatenates strings and lists (`(a | intersect(b) + ['Other']) | first`)
- `{% set name = expression %}` in templates, for the rest of the template (inside a for loop, for that iteration)
- `True`, `False` and `None` as in Jinja; `and`, `or`, `not`, `in`
- A variable set to `null`/`None` is defined (`x is defined` is true, `x is none` is true) and prints as an empty string, as in Ansible; an undefined name in `{{ }}` fails the task. Unlike Ansible, `default(...)` also replaces `None`
- Dictionary methods `d.keys()` and `d.values()`
- Precedence as in Jinja: a filter applies to the operand right before it
  (`a and b | length > 0` is `a and ((b | length) > 0)`), `not` is weaker than
  comparisons (`not n > 5` is `not (n > 5)`)

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

Current limitation: filters from plugins are registered only in the Go-template engine, not in the expression evaluator. `{{ name | strrev }}` does not find them; call them as functions with Go-template syntax: `{{ strrev .name }}`, `{{ prefix .name "app-" }}`. Plugin filters do not work in conditions. A plugin filter with a built-in's name (such as `reverse`) is shadowed by the built-in.

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
    plugin.AddFilter("strrev", reverseFilter)

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
1. Call plugin filters as Go-template functions (`{{ strrev .name }}`), not with `|`
2. Verify plugin is registered with plugin manager
3. Check filter name matches exactly (case-sensitive)
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
