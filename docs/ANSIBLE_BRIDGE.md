# Ansible bridge

Modules onigirazu does not have run through an installed ansible-core, when `onigirazu.yml` allows
them. The list is explicit: a module that is neither built in nor allowed is still an error before the
run (a typo does not silently call Ansible).

```yaml
# onigirazu.yml
ansible_bridge:
  modules:
    - community.general.*     # patterns match the name as written in the task
    - win_*
    - known_hosts
  ansible_playbook: ~/.local/bin/ansible-playbook   # default: ansible-playbook from PATH
```

A relative `ansible_playbook` path is resolved against the directory containing `onigirazu.yml`;
a leading `~` expands to the home directory. A bare name is looked up in `PATH`.

The bridge passes collection search paths to both `ansible-playbook` and `ansible-doc` through
`ANSIBLE_COLLECTIONS_PATH`: `collections_path` from `onigirazu.yml`, then `ANSIBLE_COLLECTIONS_PATH`
(and `ANSIBLE_COLLECTIONS_PATHS`) from the environment, then `ansible.cfg`, followed by the default
user and system collection directories. Relative `ansible.cfg` entries are resolved against that
file’s directory. Duplicate paths are removed, preserving their first occurrence.

For each bridged task and host onigirazu runs `ansible-playbook` once, with:

- a one-host inventory: the host's `ansible_*` variables (WinRM, connection plugins and so on), its
  address, port, user, key, password and become password;
- a one-task playbook with the rendered arguments, `become`, `environment`, `no_log`; every string is
  `!unsafe`, so text such as `{{x}}` reaches the module as it is;
- `--check` and `--diff` as the run has them; a module without check mode is reported skipped;
- `StrictHostKeyChecking=no` hosts get `ANSIBLE_HOST_KEY_CHECKING=False`.

The files are 0600 in a temporary directory removed after the task; secrets never appear on a command
line. The result comes back through a callback plugin: `changed`, `failed`, `skipped`, `msg` and the
module's return values are registered as usual, plus `via: ansible`. Unreachable hosts and missing
ansible-playbook are task failures with Ansible's message.

Collections the bridged modules need are installed the Ansible way (`ansible-galaxy collection
install`), and connections such as WinRM need their Python packages (`pywinrm`).

## Checking bridged tasks before the run

`onigirazu lint` checks the arguments of bridged tasks against the module's spec from
`ansible-doc -j`: unknown names (with the closest known one), missing required arguments, and literal
values outside the module's choices; templated values are left to run time. Specs are cached per
ansible-core version and collection search paths in the user cache directory (`onigirazu/ansible-doc/`), so later runs need no
ansible-doc. `apply`, `validate` and `lint` read `ansible_bridge` from `onigirazu.yml` the same way.

```text
$ onigirazu lint site.yml
  ✗ [module-args] [web → add host key] module known_hosts has no argument "nme" (did you mean "name"?)
  ✗ [module-args] [web → add host key] module known_hosts: state is gone, not one of [absent present]
```

`onigirazu doc MODULE` shows the arguments of a built-in module, or of any module ansible-doc knows,
with types, defaults, choices and aliases, and says whether it is allowed through the bridge.

