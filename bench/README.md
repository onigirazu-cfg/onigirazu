# Benchmark against Ansible

`bench/run.sh` runs `site.yml` with onigirazu and with ansible-playbook on fresh disposable VMs
(the Ubuntu 24.04 e2e base) for each host count, and records per tool: converge, a second run that
must change nothing, and check mode; wall time, CPU time and peak memory of the control side
(the tool and its children, ssh included).

- Workflow **Bench** (Actions → Bench → Run workflow): `hosts` (default `1 5 10`), `vars` (JSON, the
  volume: `bench_files` 200, `bench_users` 20, `bench_lines` 100). Results: the run summary and the
  `bench-results` artifact (`results.tsv`, `summary.md`, the logs). A pull request that changes the
  benchmark runs it on one host.
- Fair settings: Ansible gets `forks` = host count, pipelining and ControlPersist; onigirazu the same
  parallelism. Each tool gets its own fresh VMs; the order alternates between host counts.
- The workload uses about 40 modules (files, templates, packages, users, cron, sysctl, mount, services,
  ufw, git, archives, downloads, docker, MariaDB, PostgreSQL), the same playbook for both tools.
