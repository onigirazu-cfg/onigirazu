# Safe apply: canary, health checks, automatic rollback

A rollout that checks the service after every batch of hosts and, when a batch breaks it,
puts that batch back and stops before the next one.

```yaml
plays:
  - name: web
    hosts: web
    become: true
    serial: [1, "25%"]            # batches; the first one is the canary
    health_check:                 # after every batch, on its hosts
      - name: the app answers
        uri: {url: "http://127.0.0.1:8080/health", status_code: 200}
        retries: 10
        delay: 3
      - wait_for: {port: 443, timeout: 30}
    on_unhealthy: rollback        # default with health_check; or stop, continue
    tasks:
      - template: {src: app.conf.j2, dest: /etc/app.conf}
        notify: restart app
    handlers:
      - name: restart app
        service: {name: app, state: restarted}
```

```bash
onigirazu apply site.yml -i hosts.yml                          # batches from serial
onigirazu apply site.yml -i hosts.yml --canary 1 --canary-pause 5m
onigirazu apply site.yml -i hosts.yml --auto-rollback          # also plays without health_check
```

## What happens per batch

1. The batch's tasks and handlers run.
2. `health_check` runs on the batch's hosts: ordinary tasks (`uri`, `wait_for`, `command` with
   `until`, `assert`, ...), with `retries`/`delay`. A check that fails marks the host unhealthy;
   checks never take hosts out of the run and are not filtered by tags.
3. The batch is unhealthy when a host failed a task or a check.
4. Unhealthy:
   - `rollback`: the changes this run made on the batch are undone, newest first (files back to
     their content, mode and owner, or removed if they were new; packages the run installed
     removed and removed ones reinstalled; services back to running/enabled as before; accounts
     the run created deleted). Changes without a previous state (commands, upgrades of installed
     packages, ...) are listed as kept. The checks run again and the report says whether the
     batch is healthy again. Later batches do not start.
   - `stop`: the rollout stops, nothing is undone.
   - `continue`: the rollout goes on (the failed hosts have left the run).

## Options

| Option | Meaning |
|--------|---------|
| `--canary N` / `--canary P%` | The first batch has this size; the play's `serial` splits the rest (without `serial` the rest is one batch). |
| `--canary-pause DURATION` | After a healthy canary, wait (soak) and run the checks again before going on. `G` in the dashboard or Ctrl+C ends the wait. |
| `--auto-rollback` | Roll back an unhealthy batch in every play, also plays without `health_check` (a failed task makes the batch unhealthy). |
| `--rollback-scope batch\|run` | `batch` (default) undoes the unhealthy batch; `run` undoes every batch of the play so far. |

Without `health_check`, `--canary` or `--auto-rollback` a play runs as before (`serial` batches,
failed hosts leave the run).

Hosts run in inventory order (then by name), so batches and the canary are the same on every run.

## Result

```
Rollout:
  ✓ canary batch 1/3 (web1): healthy
  ✓ batch 2/3 (web2, web3): healthy
  ✗ batch 3/3 (web4, web5): unhealthy, health checks failed: web5
    rolled back: 4 change(s) undone, healthy again
```

- Exit code **5**: a batch was rolled back; 1: failed without a rollback; 0: every batch healthy.
- `-o json`: `rollout` lists the batches (hosts, healthy, reason, undone, irreversible,
  rollback errors, healthy_after_rollback); `rolled_back` is true.
- The run snapshot (`onigirazu rollback --last`) no longer holds the changes the rollout undid.
- In check mode nothing is changed, so nothing is rolled back: an unhealthy batch stops the rollout.
- `--interactive`: the dashboard shows the batches as they go (B for the details); see
  [INTERACTIVE_MODE.md](INTERACTIVE_MODE.md).
