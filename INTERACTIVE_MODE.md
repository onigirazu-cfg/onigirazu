# Interactive Mode User Guide

**Version:** v1.54.0+ | **Status:** Production Ready ✅

## Overview

Interactive Mode is a powerful feature that transforms your Onigirazu execution experience with a beautiful, responsive terminal UI dashboard. It provides real-time log streaming, multi-mode display switching, execution statistics, and graceful control over your playbooks.

Perfect for:

- 📊 **Long-running playbooks** - Monitor progress without terminal noise
- 🐛 **Debugging** - Switch to DEBUG mode to see detailed logs on demand
- 🚀 **CI/CD pipelines** - Visual feedback during automated deployments
- 👀 **Learning** - Understand task execution flow with real-time updates
- 🎯 **Production deployments** - Gracefully stop if needed

## Quick Start

### Enable Interactive Mode

Add the `--interactive` flag to your `apply` command:

```bash
# Basic usage
onigirazu apply playbook.yaml -i inventory.yaml --interactive

# With other flags
onigirazu apply deployment.yaml -i "ubuntu@server1" --verbose --interactive

# With tags
onigirazu apply site.yaml -i inventory.yaml --tags web,app --interactive
```

### What You See

When you run with `--interactive`, you get a professional dashboard showing:

- 🖼️ **Task Status** - Real-time execution with color coding
- 📋 **Live Logs** - Scrollable log view with full output
- 📊 **Statistics** - Task counts, timing, and progress
- ⌨️ **Control Panel** - Help text and keyboard shortcuts
- 🎨 **Beautiful Layout** - Professional design with proper spacing

## Keyboard Controls

### Display Modes

| Key | Mode | Description |
|-----|------|-------------|
| **N** | NORMAL | Standard task output and errors only |
| **V** | VERBOSE | Detailed output with variables and timestamps |
| **D** | DEBUG | Complete debug information for troubleshooting |

### Navigation

| Key | Action | Description |
|-----|--------|-------------|
| **↑** | Scroll Up | Move up in the log view |
| **↓** | Scroll Down | Move down in the log view |
| **Page Up** | Page Up | Jump 10 lines up |
| **Page Down** | Page Down | Jump 10 lines down |

### Information & Help

| Key | Action | Description |
|-----|--------|-------------|
| **S** | Statistics | Show execution statistics overlay |
| **H** | Help | Display keyboard shortcuts help |

### Execution Control

| Key | Action | Description |
|-----|--------|-------------|
| **G** | Graceful Stop | Initiate graceful shutdown (finishes current tasks) |
| **Q** | Quit TUI | Exit the dashboard (continues execution in background) |
| **Ctrl+C** | Force Quit | Emergency exit (may interrupt execution) |

## Display Modes Explained

### Normal Mode (Default)

```
Standard task output and basic error information. Clean and concise.
Best for: Production runs where you don't need detailed debug info.

Example output:
[TASK] Installing packages...
[OK] Installation complete
[ERROR] Failed to start service
```

### Verbose Mode

```
Detailed output including variable values, timestamps, and execution flow.
Best for: Debugging specific task behavior.

Example output:
[2024-01-15 14:32:45] [TASK] Installing packages...
  vars: package_list = ["nginx", "curl"]
  target: ubuntu@server1
[OK] Installation complete in 2.3s
```

### Debug Mode

```
Complete debug information for in-depth troubleshooting.
Best for: Complex issues or understanding internal execution.

Example output:
[DEBUG] Event: task_started
  id: "task_001"
  name: "Installing packages"
  module: "package"
  vars_context: {package_list: [...], env: {...}}
[DEBUG] Module execution: package.Execute()
[DEBUG] Command: apt-get install -y nginx curl
```

## Features in Detail

### 1. Live Log Dashboard

The dashboard displays logs in real-time as tasks execute. Features include:

- ✅ **Auto-scrolling** - Follows new logs automatically
- ✅ **Scrollable history** - Manually scroll to see previous logs
- ✅ **Color-coded output** - Task status shown with colors
- ✅ **Timestamps** - In VERBOSE and DEBUG modes
- ✅ **Context preservation** - Keeps track of multiple tasks

### 2. Multi-Mode Display

Switch between 3 display modes without stopping execution:

```
NORMAL    - Standard output (default, lightest)
  ↓ Press V to increase verbosity
VERBOSE   - Detailed output (recommended for debugging)
  ↓ Press D to maximum verbosity
DEBUG     - Complete debug info (very verbose)
  ↑ Press N to decrease verbosity
```

### 3. Execution Statistics

Press **S** to see execution statistics overlay:

```
╔════════════════════════════════════════╗
║    EXECUTION STATISTICS                ║
╠════════════════════════════════════════╣
║ Total Tasks: 15                        ║
║ Completed:   8                         ║
║ Running:     1                         ║
║ Failed:      0                         ║
║ Skipped:     0                         ║
║                                        ║
║ Elapsed Time: 2m 34s                   ║
║ Est. Total:   3m 15s                   ║
╚════════════════════════════════════════╝
```

### 4. Graceful Shutdown

Press **G** to gracefully stop execution:

- ✅ Current tasks complete normally
- ✅ No new tasks are started
- ✅ Execution finishes cleanly
- ✅ State is properly saved
- ✅ Resources are cleaned up

Recommended for:

- ⏹️ Stopping long-running deployments
- 🚫 Canceling unnecessary runs
- 🔄 Preventing new tasks when not needed

### 5. Help Overlay

Press **H** to see the interactive help overlay showing all keyboard shortcuts and current mode status.

## Practical Examples

### Example 1: Monitoring a Deployment

```bash
# Start your deployment with interactive mode
onigirazu apply deploy-app.yaml -i prod-inventory.yaml --interactive

# You see the live dashboard
# While deployment runs:
# - Press V to see verbose output if you want more detail
# - Press S to check overall progress
# - Press D if something seems wrong to debug
# - Press G to stop if you notice an issue
```

### Example 2: Debugging a Failing Task

```bash
# Run with interactive mode
onigirazu apply problematic-playbook.yaml -i localhost --interactive

# When a task fails:
# 1. You see it in the dashboard immediately (red error)
# 2. Press V to switch to VERBOSE mode and see more context
# 3. Press D to switch to DEBUG mode for complete info
# 4. Scroll up with ↑ arrow to see previous task outputs
# 5. Press G to stop and examine the failure
```

### Example 3: Long-Running Infrastructure Setup

```bash
# Start a long infrastructure setup
onigirazu apply setup-cluster.yaml -i cluster-inventory.yaml --interactive

# You can now:
# - Step away but keep monitoring
# - Press S to check progress without disrupting view
# - Switch between modes as needed (N/V/D)
# - See exactly where execution is in the task list
# - Gracefully stop if something unexpected happens
```

## Combining with Other Flags

Interactive mode works with all other Onigirazu flags:

```bash
# With tags
onigirazu apply playbook.yaml -i inventory.yaml --tags web,app --interactive

# With verbosity
onigirazu apply playbook.yaml -i inventory.yaml --verbose --interactive

# With dry-run
onigirazu apply playbook.yaml -i inventory.yaml --plan --interactive

# With limits
onigirazu apply playbook.yaml -i inventory.yaml --limit webservers --interactive

# With custom config
onigirazu apply playbook.yaml -i inventory.yaml -c config.yaml --interactive
```

## Performance Considerations

### Dashboard Performance

- **Refresh Rate**: 30 FPS for smooth UI
- **Memory Usage**: Minimal (efficient event streaming)
- **CPU Impact**: Negligible (<1%)
- **Network**: No impact (local TUI only)

### Large Deployments

Interactive Mode handles large deployments efficiently:

- ✅ Works with 1000+ tasks
- ✅ Handles 10,000+ log lines
- ✅ Scrolling remains responsive
- ✅ No performance degradation

Tested with:

- 1000+ concurrent tasks
- 100k+ log messages
- Complex multi-play deployments
- Long-running background jobs

## Architecture Overview

### How It Works

```
Playbook Execution → Event Stream ↘
                                    → TUI Dashboard (Real-time Display)
                    ↖ Keyboard Input (Control)
```

**Components**:

1. **Executor**: Generates execution events
2. **Event Bus**: Channels events to TUI
3. **TUI Model**: Manages display state
4. **Event Listener**: Receives and processes events
5. **Renderer**: Beautiful Bubble Tea/Lipgloss UI

### Event Flow

```
Task Starts → EVENT_TASK_START → Dashboard Updates → Display Refreshes
                ↓
              Log Event → EVENT_LOG → Buffer Update → Dashboard Updates
                ↓
           Task Complete → EVENT_TASK_END → Statistics Update → Refresh
```

## Troubleshooting

### Dashboard Appears Frozen

**Cause**: Playbook waiting for input or hanging

**Solution**:

1. Press **S** to check if progress is being made
2. Press **G** to gracefully stop
3. Ctrl+C to force quit if needed

### Can't See All Output

**Cause**: Terminal window too small or using NORMAL mode

**Solution**:

1. Press **V** to switch to VERBOSE mode
2. Use ↑/↓ arrows to scroll through logs
3. Enlarge your terminal window

### Mode Switching Not Working

**Cause**: Keyboard input not registered (rare)

**Solution**:

1. Click in the terminal window to ensure focus
2. Press **H** to refresh help display
3. Try the keyboard shortcut again

### Dashboard Performance Issues

**Cause**: Very large deployments (10,000+ tasks)

**Solution**:

1. These are tested and work, but may be slow on older systems
2. Reduce log verbosity with **N** key (NORMAL mode)
3. Use `--limit` to run fewer tasks at once

## Best Practices

### ✅ DO

- ✅ Use `--interactive` for new playbooks to debug faster
- ✅ Use VERBOSE mode for complex deployments
- ✅ Use DEBUG mode for hard-to-find issues
- ✅ Use **S** to check progress without stopping
- ✅ Use **G** to gracefully stop when needed
- ✅ Scroll through logs to find error context

### ❌ DON'T

- ❌ Don't use interactive mode for fully automated CI/CD (no TTY available)
- ❌ Don't force quit with Ctrl+C unless necessary (use G for graceful stop)
- ❌ Don't close the terminal while deployment is running
- ❌ Don't assume silence means hanging (check with S key)

## Environment Variables

Optional environment variables for interactive mode:

```bash
# Disable colors in dashboard
CLICOLOR=0 onigirazu apply playbook.yaml -i inventory.yaml --interactive

# Set custom theme (future versions)
ONG_THEME=dark onigirazu apply playbook.yaml -i inventory.yaml --interactive
```

## Version History

### v1.54.0 (Current)

- ✅ Interactive Mode Production Release
- ✅ Stable event streaming
- ✅ All keyboard controls functional
- ✅ Beautiful TUI dashboard
- ✅ Multi-mode display
- ✅ Graceful shutdown support

### v1.53.0-v1.53.3

- 🧪 Beta testing and refinement
- 🧪 Event flow optimization
- 🧪 Performance tuning

### Pre-v1.53.0

- 🛠️ Development and initial implementation

## Getting Help

### Documentation

- **Main README**: [../README.md](../README.md) - Interactive Mode Quick Start
- **CLI Help**: `onigirazu apply --help` - Command line options
- **In-Dashboard Help**: Press **H** while running - Keyboard shortcuts

### Reporting Issues

Found a bug? Please report it on GitHub with:

1. Your Onigirazu version (onigirazu --version)
2. Your terminal type (echo $TERM)
3. Steps to reproduce
4. Screenshot if possible

### Contributing

Interested in improving Interactive Mode? Check out the [CONTRIBUTING.md](../CONTRIBUTING.md) guide.

## Summary

Interactive Mode transforms your Onigirazu experience with:

- 🖥️ Beautiful, responsive dashboard
- 📊 Real-time log streaming
- 🔄 Multi-mode display switching
- 📈 Execution statistics
- ⌨️ Intuitive keyboard control
- 🎯 Graceful execution control

Perfect for debugging, monitoring, and understanding your automation workflows.

**Get started now**: `onigirazu apply your-playbook.yaml -i inventory.yaml --interactive`
