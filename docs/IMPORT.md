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

## Output

```
imported/
  site.yml                 one play per host
  roles/host_web1/
    tasks/main.yml         repositories, accounts, packages, files, timezone, mounts, services
    files/etc/...          the files, by their path on the host
  IMPORT_REPORT.md
```

## Flags

| Flag | Effect |
|---|---|
| `-o DIR` | where to write (must be empty, or `--force`) |
| `--no-verify` | do not plan the new playbook against the hosts |
| `--no-become`, `--become-user` | collect as the login user / another user |
