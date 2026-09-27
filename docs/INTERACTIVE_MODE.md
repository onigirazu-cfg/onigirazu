# Interactive Mode User Guide

## Overview

`apply --interactive` (or `interactive_mode: true` in the config) shows a terminal dashboard
while the playbook runs: a live log, statistics with a host list, a browser for task results,
and keys to pause or stop the run. It needs a terminal: without one (CI, a pipe, `-o json`)
`apply` says so and runs with its normal output.

When the dashboard closes, the usual end-of-run summary is printed.

```bash
onigirazu apply site.yml -i inventory.yml --interactive
onigirazu apply site.yml -i inventory.yml --tags web --check --interactive
```

## The screen

- **Header**: playbook, status (INITIALIZING, RUNNING, PAUSED, STOPPING, COMPLETED, FAILED,
  STOPPED), detail level, elapsed time, and task counts.
- **Log** (left): one line per finished task and host (`✓ web1: Install nginx [changed]`),
  warnings and errors. More detail with **V** / **D**.
- **Statistics** (right): play and task progress, speed, the task running now, and every host
  with its ok / changed / failed counts; hosts that failed (they have left the run) come first,
  in red.

## Keys

Keys are case-insensitive. **H** shows them all.

### Run

| Key | Action |
|-----|--------|
| **P** | Pause / resume. Tasks already running finish; the next task waits. |
| **G** | Stop gracefully (asks first): running tasks finish, no new task starts. The dashboard stays open. |
| **Q**, **Ctrl+C** | Close. While the run goes on it asks first, and **Y** stops the run and closes. A second **Ctrl+C** at the question does the same. After the run, **Q** closes at once. |
| **X** | After the run: run the playbook again on the hosts that failed (asks first). The dashboard closes, and the same command runs again with `--limit` set to those hosts, in a new dashboard. |

A stopped run ends with a non-zero exit code and "run stopped by user"; state, audit and the
rollback snapshot are saved as for any run.

### Results

| Key | Action |
|-----|--------|
| **R** | Task results, newest last: host, task, status |
| **↑** / **↓**, **PgUp** / **PgDn**, **Home** / **End** | Move |
| **Enter** | Everything about one result: module, duration, error, message, stdout, stderr, and the diff of a changed file (as with `--diff`); **↑↓ PgUp PgDn Home End** scroll |
| **F** | Failed results only (toggle) |
| **Esc** | Back |

### Timeline

| Key | Action |
|-----|--------|
| **L** | Every host as a row of task marks in run order (✓ ok, ⟳ changed, ✗ failed, ⊘ skipped), failed hosts first |
| **↑** / **↓**, **Enter** | Select a host and show its tasks as bars on the run's time axis: when each started and how long it took |
| **Esc** | Back |

### Rollout

With health checks, `--canary` or `--auto-rollback` (see [SAFE_APPLY.md](SAFE_APPLY.md)) the
statistics panel shows the batches of the play as marks (✓ healthy, ◌ running, ◎ checking health,
⏳ canary soak, ✗ unhealthy, ↺ rolled back), the log reports every batch, and the header says
ROLLING BACK while a batch is being put back.

| Key | Action |
|-----|--------|
| **B** | Every batch: hosts, state, reason, unhealthy hosts, changes undone, changes kept, rollback errors, health after the rollback |

### Log

| Key | Action |
|-----|--------|
| **N** | Normal: task results, warnings, errors |
| **V** | Verbose: also the task's message and first output lines, tasks as they start, every log line |
| **D** | Debug: also debug lines |
| **↑** / **↓** (**k** / **j**), **PgUp** / **PgDn**, **Home** / **End** | Scroll; new lines keep coming at the bottom, **End** follows them again |
| **F** | Filter mode: **E** errors, **W** warnings, **T** tasks, **C** clear |
| **/** | Search: type, **Enter** keeps the filter, **Esc** leaves |
| **S** | Statistics overlay: speed, fastest and slowest tasks |

## Troubleshooting

- **Every command starts about 5 seconds late**: the terminal does not answer the colour query
  the terminal library sends at start. Use another terminal, or `TERM=dumb` for such a session.
- **"Interactive mode needs a terminal"**: stdin or stderr is not a terminal, or `-o json|yaml`
  is set; the run goes on with the normal output.
- **"Terminal too small"**: the dashboard needs at least 80x24.
