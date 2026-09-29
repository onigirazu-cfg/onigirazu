# Testing roles: onigirazu test

`onigirazu test` runs a role's Molecule scenarios with onigirazu instead of ansible-playbook. It is
the command plugin `onigirazu-test`, shipped next to `onigirazu` in the release archives (or
`make build`); see [PLUGIN_INTEGRATION.md](PLUGIN_INTEGRATION.md). Molecule itself is not needed,
nor Python.

```sh
cd roles/users
onigirazu test                      # the default scenario: the whole test sequence
onigirazu test -s all               # every scenario
onigirazu test --destroy never      # keep the instances after the run
onigirazu test converge             # one step: create first, then converge, verify, ...
onigirazu test login                # a shell in the instance
onigirazu test destroy
```

Steps: `dependency`, `cleanup`, `destroy`, `syntax`, `create`, `prepare`, `converge`, `idempotence`,
`side_effect`, `verify`; `test` runs the scenario's `test_sequence` (Molecule's by default) and
destroys the instances at the end, after a failure too, unless `--destroy never`.

What is read from `molecule.yml` (environment variables are expanded as Molecule does:
`${VAR}`, `${VAR:-default}`):

| Key | Support |
|-----|---------|
| `driver.name` | `docker` or `podman` |
| `platforms` | `name`, `image`, `hostname`, `privileged`, `cgroupns_mode`, `override_command`, `command`, `volumes`, `tmpfs`, `capabilities`, `env`, `networks`, `published_ports`, `groups`; `pre_build_image: false` (building an image) is not supported |
| `provisioner.env` | passed to the runs; relative `*_PATH` values start at the scenario directory |
| `provisioner.playbooks` | other file names for `converge`, `prepare`, `verify`, `side_effect`, `cleanup` |
| `provisioner.inventory` | `group_vars`, `host_vars` |
| `dependency` | `galaxy` with `requirements-file`: `onigirazu galaxy install` into the scenario's cache directory (`--no-deps` skips it) |
| `verifier` | `ansible` (`verify.yml`); others are skipped |
| `scenario.test_sequence` | the steps of `test` |

The instances use the container connection (`ansible_connection: docker`), so images need no SSH
server. Container names are `molecule-<role>-<scenario>-<platform>`, so the scenarios of different
roles can run at the same time. The role's parent directory is in `ANSIBLE_ROLES_PATH`, as in
Molecule. The inventory, the state of the runs and the installed dependencies live in the user
cache directory (`~/.cache/onigirazu-test/`), not in the role.

`idempotence` runs converge again and fails, listing the tasks and hosts, when any task changes.
