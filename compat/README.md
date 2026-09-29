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

A case that starts with `# compat-hosts: N` runs on N hosts (`h1`..`hN`, in groups `odd` and `even`); each host's
tasks and files are compared on their own, since hosts run in parallel.

Needs docker, `ansible-playbook` (ansible-core, e.g. `uv tool install ansible-core`) and go. Runs on
the docker-test-lab VM. A case is a plain Ansible playbook for `hosts: all`; write results under
`/root/compat` or print them with `debug`.

## Known differences

- A `when:` that is not a boolean (`when: some_string`): ansible-core 2.19+ fails the task,
  onigirazu treats it as truthy/falsy.
- A debug `msg` that is a whole boolean expression: Ansible keeps the boolean (`true`), onigirazu
  prints text (`True`); compared as equal.
- Dicts print with sorted keys; Ansible keeps the playbook's order.
- Presentation, normalized away: Ansible prefixes role tasks with `role : ` and reports
  include_tasks/include_role as tasks of their own.
- A loop over `include_tasks` runs each included task over the items in turn (Ansible runs the whole
  file per item).
- A template that renders to a Python literal, such as `content: "{{ port }}\n"`, becomes that value
  in Ansible (the number 80, written as `80` without the newline); onigirazu keeps the text.
- `file: state=touch` on an existing file: Ansible reports changed every run (times move); onigirazu
  reports ok, so a second run and drift checks stay clean. The case uses `preserve` for both times.
