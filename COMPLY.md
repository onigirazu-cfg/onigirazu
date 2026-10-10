# Compliance profiles

`onigirazu comply` checks hosts against a profile — named controls with a severity and a fix,
each built on [verify](VERIFY.md) checks — and scores them per host. The whole profile travels
to a host as one script: 25 controls cost one round trip.

```bash
onigirazu comply list                                   # bundled profiles
onigirazu comply --profile linux-baseline -i hosts.yml
onigirazu comply --profile ssh -i hosts.yml --limit web --format markdown --output ssh.md
onigirazu comply --profile ./company.yml -i hosts.yml --fail-on high --control-tags network,accounts
onigirazu comply show ssh > company.yml                 # start your own from a bundled one
```

```
linux-baseline — Linux server baseline

web1: 21/23 controls pass (91%)
  ✗ [medium] net-1 — IP forwarding is off (unless a router)
      kernel_param net.ipv4.ip_forward: value 1
      fix: sysctl net.ipv4.ip_forward=0 (skip this control on routers and container hosts)
  ✗ [low] log-2 — The journal is persistent
      file /var/log/journal: missing
      fix: mkdir /var/log/journal; systemctl restart systemd-journald

2 host(s), 44/46 controls pass (96%)
```

## Bundled profiles

| Profile | Controls | Scope |
|---|---|---|
| `linux-baseline` | 23 | CIS-style level 1 subset: permissions of account files, world-writable files, `/tmp` mount options, kernel network parameters (forwarding, redirects, source routing, rp_filter, SYN cookies, martians), firewall present, UID 0 and empty-password accounts, umask, root PATH, logger and persistent journal, time sync, pending security updates and unattended upgrades, legacy services, unexpected listeners |
| `ssh` | 12 | sshd as `sshd -T` reports it: root login, password and empty-password authentication, X11, MaxAuthTries, idle timeouts, LoginGraceTime, host-based auth, log level, config and host key permissions, service state |

Both are distribution-neutral (Debian/Ubuntu and RHEL-family) and run with `become` (they read
root-only files). Some controls are policy, not fact: `net-1` fails on routers and Docker hosts
by design, `svc-2` knows only sshd as a legitimate listener — copy the profile and edit.

## Writing a profile

```yaml
name: company            # default: the file name
title: Company baseline
become: true             # default
controls:
  - id: ssh-1            # unique; default: the position
    title: PermitRootLogin is no
    severity: high       # low, medium (default), high, critical
    tags: [ssh, access]
    check: {command: {cmd: "sshd -T | grep -i '^permitrootlogin'", stdout: ["/^permitrootlogin no$/"]}}
    remediation: "sshd_config: PermitRootLogin no"
  - id: net-2
    title: ICMP redirects are not accepted
    checks:              # several checks: all must pass
      - {kernel_param: {name: net.ipv4.conf.all.accept_redirects, value: "0"}}
      - {kernel_param: {name: net.ipv4.conf.default.accept_redirects, value: "0"}}
    remediation: sysctl
```

A check is any [verify check](VERIFY.md#checks). The remediation is text for the reader; the
fix itself is an ordinary playbook.

## Options and output

| Flag | Meaning |
|---|---|
| `--profile` | bundled name or YAML file (required) |
| `--control-tags` | only controls with one of the tags |
| `--min-severity` | only controls of this severity or above |
| `--fail-on` | exit 1 only when a failed control is of this severity or above (default: any failure) |
| `--format` | `text`, `json`, `markdown`, `html`; `--output FILE` |
| `--metrics-file`, `--metrics-push`, `--metrics-label` | `onigirazu_comply_controls{profile,host,result}`, `onigirazu_comply_score{profile,host}`, `onigirazu_comply_last_run_timestamp_seconds{profile}` as for [drift](DRIFT_AND_ROLLBACK.md#metrics) |
| `--limit`, `-e`, `-b` (default on), `--become-user`, `-u`, `--private-key` | as for `apply` |

Exit codes: 0 all controls pass (or none at the `--fail-on` severity failed), 1 a control failed
or a host could not be checked. The JSON report has every control per host (`ok`, `details`,
`remediation`), the host scores and the total.
