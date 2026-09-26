# Interactive Mode User Guide

## Overview

`apply --interactive` shows a terminal dashboard while the playbook runs: a live log view, a statistics panel and keyboard shortcuts for display modes, filtering and search. It needs an interactive terminal (TTY); do not use it in CI.

## Quick Start

```bash
# Basic usage
onigirazu apply playbook.yaml -i inventory.yaml --interactive

# With tags
onigirazu apply site.yaml -i inventory.yaml --tags web,app --interactive
```

When the run finishes, the dashboard stays open until you press **Q**. The end-of-run summary that `apply` normally prints is not shown in interactive mode.

## Keyboard Controls

Keys are case-insensitive.

### Display Modes

| Key | Action |
|-----|--------|
| **V** | Toggle VERBOSE (back to NORMAL when pressed again) |
| **D** | Toggle DEBUG: also show debug events (back to NORMAL when pressed again) |
| **N** | Back to NORMAL |

Debug events are shown only in DEBUG mode. VERBOSE currently shows the same events as NORMAL.

### Navigation

| Key | Action |
|-----|--------|
| **↑** / **↓** | Scroll the log one line |
| **Page Up** / **Page Down** | Scroll the log half a screen |

### Filtering and Search

| Key | Action |
|-----|--------|
| **F** | Toggle filter mode |
| **E** / **W** / **T** | In filter mode: cycle the error / warning / task filters |
| **/** | Start a search; type the text, **Enter** to finish, **Backspace** to delete |
| **C** | In filter or search mode: clear filters |

### Information

| Key | Action |
|-----|--------|
| **S** | Toggle the statistics overlay (speed, fastest and slowest tasks, tasks by duration) |
| **H** | Toggle the help overlay |

In an overlay, press the same key again or **Q** to close it.

### Execution Control

| Key | Action |
|-----|--------|
| **Q** / **Ctrl+C** | Close the dashboard. The run is not interrupted; it continues without a display. |
| **P** | Pause/resume the elapsed-time counter in the dashboard |
| **G** | Ask "Stop execution gracefully?" (**Y** / **N**) |

**Current limitation**: **P** does not pause the run, and confirming **G** marks the dashboard as "stopping" but does not stop the run. To stop a run, close the dashboard with **Q** and interrupt the process (Ctrl+C in the terminal, or send SIGINT/SIGTERM).

## Practical Examples

### Monitoring a deployment

```bash
onigirazu apply deploy-app.yaml -i prod-inventory.yaml --interactive
# S - check progress and timings
# D - show debug events if something looks wrong
# F then E - show only errors
```

### Finding a failure

```bash
onigirazu apply problematic-playbook.yaml -i inventory.yaml --interactive
# / - search for the host or task name, Enter to keep the filter
# ↑ / Page Up - scroll back to earlier output
# C - clear the search
```

## Combining with Other Flags

```bash
# Tags
onigirazu apply playbook.yaml -i inventory.yaml --tags web,app --interactive

# Check mode
onigirazu apply playbook.yaml -i inventory.yaml --check --interactive

# Limit hosts
onigirazu apply playbook.yaml -i inventory.yaml --limit webservers --interactive

# Config file
onigirazu apply playbook.yaml -i inventory.yaml -c config.yaml --interactive
```

## How It Works

The execution engine sends task and log events to the dashboard (a Bubble Tea program), which redraws on each event and on a 500 ms timer. Logs are written into the dashboard instead of the terminal.

## Troubleshooting

### Dashboard seems frozen

1. Press **S** to see whether tasks are still completing.
2. A task may be waiting (for example `pause`, `wait_for` or a slow command).
3. To stop the run, press **Q** and then Ctrl+C.

### Cannot see all output

1. Scroll with ↑/↓ and Page Up/Page Down.
2. Clear filters with **C** (in filter or search mode).
3. Enlarge the terminal window.

### Dashboard stays open after the run

This is expected: press **Q** to exit.

## Best Practices

- Use `--interactive` for runs you watch; use plain output or `--output json` for CI and logs.
- Use **F**/**/** to narrow the log instead of scrolling.
- Do not rely on **G** or **P** to control the run.

## Getting Help

- `onigirazu apply --help` for command line options
- **H** in the dashboard for keyboard shortcuts
- Bug reports: include `onigirazu version`, your terminal (`echo $TERM`) and steps to reproduce
