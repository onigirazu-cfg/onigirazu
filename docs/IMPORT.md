# import: a playbook from running hosts

```
onigirazu import web1 -i hosts.yml -o imported/
```

`import` connects to the hosts (read-only, with sudo) and writes a playbook that recreates what
makes each of them differ from a fresh install. Then it plans the new playbook against the same
hosts: a faithful import has nothing to change, and the command exits 0; otherwise it lists the
tasks that would still change something and exits 2.

## What is taken

| What | How it is found | Task |
|---|---|---|
| packages | installed by hand: `apt-mark showmanual`, `dnf repoquery --userinstalled` | `apt` / `package` |
| repositories | files under /etc/apt, /usr/share/keyrings, /etc/yum.repos.d, /etc/pki/rpm-gpg, and local `file:` repositories they use | `copy`, before the packages |
| accounts | uid/gid 1000-59999 with their groups | `group`, `user` |
| files | no package owns them (/etc, /opt, /srv, /usr/local, user crontabs), or a package's configuration file that was edited | `copy`, `file` (directories, links), with mode and owner |
| services | enabled or disabled against the vendor preset (running ones also started) | `service` |
| timezone, mounts | /etc/localtime, /etc/fstab (not /, /boot, swap) | `timezone`, `mount` |

Left out, and listed in `IMPORT_REPORT.md`: files over 1 MiB, directories under /opt, /srv and
/usr/local with more than 300 files (applications, not configuration), masked units, services of
a host without systemd, and host identity and generated files (ssh host keys, machine-id,
hostname, resolv.conf, CA bundles, systemd enablement links, cloud-init files, ...).

Private keys and credential files are never written; the report lists them to be provided from a
vault.

## Many hosts: shared roles

```
onigirazu import web1 web2 db1 -i hosts.yml -o imported/
onigirazu import all -i hosts.yml -o imported/
```

What the hosts have in common goes into shared roles, the rest into a role per host:

| Role | Holds |
|---|---|
| `common` | what every imported host has, the same |
| one per inventory group (`web`) | what all imported hosts of the group share (the biggest group first) |
| `<service>_hosts` | hosts in no group, clustered when they share at least half of their packages and services; named after a service, a server package or the distribution |
| `host_<name>` | what is left for one host |

A file with the same path, mode and owner everywhere joins the shared role even when its content
differs:

- when it differs only by the hosts' own names and addresses, it becomes a template;
  `host_vars/<host>.yml` holds `import_hostname`, `import_fqdn`, `import_ip`. Rendering gives back
  each host's file byte for byte; files that already contain Jinja syntax are not templated.
- otherwise every host gets its own copy: `files/<host>/<path>`, `src: "{{ inventory_hostname }}/..."`.

An account with another uid on some hosts stays in the host roles.

## Output

```
imported/
  site.yml                 one play per stage over all hosts: repositories, accounts,
                           packages, files, system (timezone, mounts), services
  roles/<role>/
    tasks/<stage>.yml      the role's tasks of a stage; main.yml imports them all
    files/, templates/
  host_vars/<host>.yml     template variables
  IMPORT_REPORT.md         roles and their hosts, what was left out, secrets, the check
```

Each stage runs on all hosts before the next, so a file of a shared role can belong to an
account of a host role. Every play runs the stage of each role with `include_role` and a `when`
that picks the role's hosts (`'web' in group_names`, `inventory_hostname == 'db1'`).

## Flags

| Flag | Effect |
|---|---|
| `-o DIR` | where to write (must be empty, or `--force`) |
| `--no-verify` | do not plan the new playbook against the hosts |
| `--no-become`, `--become-user` | collect as the login user / another user |
