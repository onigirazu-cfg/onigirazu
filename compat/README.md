# Compatibility check against Ansible

`compat/run.sh` runs each playbook in `cases/` with `ansible-playbook` and with onigirazu, each on its
own fresh container from the rig image, twice (the second run shows idempotence), and compares:

- per task: status (`ok`, `changed`, `skipped`, `failed`, `ignored`) and message (debug output, the
  `fail` module's message; other error texts differ between the tools and are not compared);
- the files the playbook left under `/root/compat` (checksums).

Loop items, which onigirazu reports one by one, are folded into one task first.

```bash
make docker-up RIG=ubuntu2404       # the rig image and key
compat/run.sh                       # all cases
compat/run.sh compat/cases/loops.yml
```

Needs docker, `ansible-playbook` (ansible-core, e.g. `uv tool install ansible-core`) and go. Runs on
the docker-test-lab VM. A case is a plain Ansible playbook for `hosts: all`; write results under
`/root/compat` or print them with `debug`.

## Known differences

- A `when:` that is not a boolean (`when: some_string`): ansible-core 2.19+ fails the task,
  onigirazu treats it as truthy/falsy.
- A debug `msg` that is a whole boolean expression: Ansible keeps the boolean (`true`), onigirazu
  prints text (`True`); compared as equal.
- Dicts print with sorted keys; Ansible keeps the playbook's order.
