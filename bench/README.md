# Benchmark against Ansible

`bench/run.sh` runs `site.yml` with onigirazu and with ansible-playbook on fresh disposable VMs
(the Ubuntu 24.04 e2e base) for each host count, and records per tool: converge, a second run that
must change nothing, and check mode; wall time, CPU time and peak memory of the control side
(the tool and its children, ssh included).

- Workflow **Bench** (Actions → Bench → Run workflow): `hosts` (default `1 5 10`), `vars` (JSON, the
  volume: `bench_files` 200, `bench_users` 20, `bench_lines` 100), `server` (`auto`, `python` or `sh`:
  the command server onigirazu uses on the hosts). Results: the run summary and the
  `bench-results` artifact (`results.tsv`, `summary.md`, the logs). A pull request that changes the
  benchmark runs it on one host.
- Fair settings: Ansible gets `forks` = host count, pipelining and ControlPersist; onigirazu the same
  parallelism. Each tool gets its own fresh VMs; the order alternates between host counts.
- The workload uses about 40 modules (files, templates, packages, users, cron, sysctl, mount, services,
  ufw, git, archives, downloads, docker, MariaDB, PostgreSQL), the same playbook for both tools.

## State checks

After the converge and the second run of each tool, [goss](https://github.com/goss-org/goss)
(pinned version and checksum in `run.sh`) validates every host against `goss.py`'s description of the
state `site.yml` leaves: files with their modes and contents, packages, users, cron, sysctl, mount,
services, ufw, the container and its HTTP answer, the databases. Any failed check fails the bench; the
summary lists the checks per tool, host count and run.


## Scale

`bench/scale.sh BIN N "CONCURRENCIES"` (defaults: N 300, concurrencies `50 100 N`; `SERVER=sh|python|auto`,
`KEEP=1` leaves the containers up) starts N small Alpine sshd containers on one docker host,
runs a short playbook (facts, a directory, five files in a loop, lineinfile, command, assert) at each
concurrency, a converge and a second run, and prints wall time, CPU time and peak memory of onigirazu.
It cleans up its containers (`KEEP=1` leaves them and the work directory); `SERVER` picks the command
server. Run it on a lab docker host, not on a laptop.

500 hosts on a 16-core lab VM (2026-10-09):

| concurrency | converge | second run | CPU (second run) | memory |
|---|---|---|---|---|
| 50  | 7.4 s | 5.8 s | 6.2 s | 233 MB |
| 100 | 7.3 s | 5.6 s | 6.1 s | 243 MB |
| 500 | 7.8 s | 5.5 s | 6.0 s | 284 MB |

With `SERVER=sh` (the POSIX shell server, no agent upload) at concurrency 100: converge 8.7 s, second run
7.8 s, CPU 5.4 s, 215 MB.
