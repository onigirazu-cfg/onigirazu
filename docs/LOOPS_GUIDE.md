# Loops in Onigirazu

## Table of Contents

1. [Introduction](#introduction)
2. [Ansible-Style Loops](#ansible-style-loops)
3. [Onigirazu Loop Map: items and range](#onigirazu-loop-map-items-and-range)
4. [Loop Variables](#loop-variables)
5. [Loop Execution](#loop-execution)
6. [Playbook Examples](#playbook-examples)
7. [Best Practices](#best-practices)
8. [Troubleshooting](#troubleshooting)
9. [Reference](#reference)

---

## Introduction

A loop runs the same task once per item. Onigirazu supports the Ansible loop keywords (`loop`, `with_items`, `with_list`, `with_dict`, `with_sequence`, the `with_<lookup>` forms, `loop_control`) and its own map form with `items` and `range`.

---

## Ansible-Style Loops

### loop with a list

```yaml
- name: Install packages
  package:
    name: "{{ item }}"
    state: present
  loop:
    - nginx
    - curl
    - git
```

Items can be dictionaries:

```yaml
- name: Create users
  user:
    name: "{{ item.name }}"
    shell: "{{ item.shell }}"
    groups: "{{ item.groups }}"
  loop:
    - { name: alice, shell: /bin/bash, groups: sudo }
    - { name: bob, shell: /bin/sh, groups: developers }
```

String items that contain `{{ }}` are rendered per host before the loop starts.

### loop with an expression

A string is evaluated per host and must yield a list:

```yaml
- name: Find log files
  find:
    path: /var/log
    pattern: "*.log"
  register: log_files

- name: Compress each log file
  command: gzip -9 {{ item.path }}
  loop: "{{ log_files.files }}"

- name: Create numbered directories
  file:
    path: /tmp/dir-{{ item }}
    state: directory
  loop: "{{ range(1, 4) | list }}"     # 1, 2, 3
```

### with_items and with_list

The older spelling of `loop`; they take a list or an expression. As in Ansible, `with_items` expands items that are lists, one level (`[[1, 2], 3]` loops over 1, 2, 3); `with_list` and `loop` do not.

```yaml
- name: Install packages
  package: name={{ item }} state=present
  with_items:
    - nginx
    - redis
```

### with_dict and dict2items

`with_dict` loops over the `key`/`value` pairs of a dictionary, sorted by key. `loop` with the `dict2items` filter does the same:

```yaml
vars:
  sysctl_settings:
    net.core.somaxconn: 65536
    vm.swappiness: 10

tasks:
  - name: Apply sysctl settings
    sysctl:
      name: "{{ item.key }}"
      value: "{{ item.value }}"
    with_dict: "{{ sysctl_settings }}"

  - name: Same with loop
    sysctl:
      name: "{{ item.key }}"
      value: "{{ item.value }}"
    loop: "{{ sysctl_settings | dict2items }}"
```

`items2dict` converts back.

### with_sequence

`start`, `end`, `stride`, `count` and `format` (printf style). Items are strings. Templated bounds are not supported; use `loop: "{{ range(a, b) | list }}"` instead.

```yaml
- name: Create web directories
  file:
    path: /srv/{{ item }}
    state: directory
  with_sequence: start=1 end=3 format=web%02d    # web01, web02, web03
```

### with_nested, with_subelements and other lookups

`with_<name>` takes its items from the lookup of that name, as in Ansible; the terms may use
variables (`"{{ users }}"` stays a list). File lookups run on the control machine, relative to
the playbook directory. An unknown `with_*` is an error.

| Keyword | Items |
|---------|-------|
| `with_nested: [[a, b], [1, 2]]` | every combination: `[a, 1]`, `[a, 2]`, `[b, 1]`, `[b, 2]` |
| `with_together: [[a, b], [1, 2]]` | pairs by position: `[a, 1]`, `[b, 2]` (a shorter list gives none) |
| `with_subelements: ["{{ users }}", keys]` | `[user, key]` for every key of every user; a third term `{skip_missing: true}` skips users without `keys` |
| `with_indexed_items: [x, y]` | `[0, x]`, `[1, y]` |
| `with_random_choice: [a, b]` | one of them |
| `with_file`, `with_fileglob`, `with_first_found`, `with_lines`, `with_pipe`, `with_env`, `with_template` | what the lookup returns |

```yaml
- name: Authorized keys of every user
  authorized_key:
    user: "{{ item[0].name }}"
    key: "{{ item[1] }}"
  with_subelements:
    - "{{ users }}"
    - keys
    - {skip_missing: true}
```

### loop_control

| Key | Effect |
|-----|--------|
| `loop_var` | Name of the item variable (default `item`) |
| `index_var` | Name of the 0-based index variable (default `item_index`) |

`label` and other `loop_control` keys are ignored.

```yaml
- name: Create users
  user:
    name: "{{ user.name }}"
  loop: "{{ users }}"
  loop_control:
    loop_var: user
    index_var: idx
```

---

## Onigirazu Loop Map: items and range

`loop` can also be a map:

| Key | Meaning |
|-----|---------|
| `items` | A list, or an expression string that yields a list |
| `range` | A numeric or character range (see below) |
| `var` (or `variable`) | Name of the item variable |
| `index` | Name of the index variable |

```yaml
- name: Install packages
  package:
    name: "{{ pkg }}"
    state: present
  loop:
    items: [nginx, postgresql, redis]
    var: pkg
```

### Numeric ranges

`range` is `start-end` or `start-end:step`. Both ends are inclusive; items are integers. `range` is not templated.

| Range | Items |
|-------|-------|
| `"1-10"` | 1 … 10 |
| `"1-20:2"` | 1, 3, 5, … 19 |
| `"0-5"` | 0 … 5 |
| `"10-1"` | 10, 9, … 1 |
| `"20-0:2"` | 20, 18, … 0 |

```yaml
- name: Create numbered directories
  file:
    path: /tmp/dir-{{ item }}
    state: directory
  loop:
    range: "1-10"
```

### Character ranges

Single letters: `"a-z"`, `"A-Z"`, `"a-z:3"` (a, d, g, …), `"z-a"`. Items are strings. Letters and non-letters cannot be mixed.

The whole range is expanded into a list before the loop starts.

---

## Loop Variables

| Variable | Value |
|----------|-------|
| `item` | Current item (renamed by `loop_var` or `var`) |
| `item_index` | 0-based index (renamed by `index_var` or `index`) |
| `loop.index` | 1-based index |
| `loop.index0` | 0-based index |
| `loop.length` | Number of items |
| `loop.first` | `true` on the first item |
| `loop.last` | `true` on the last item |

```yaml
- name: Show position
  command: echo "{{ loop.index }}/{{ loop.length }} {{ item }}"
  loop: [apple, banana, cherry]
```

---

## Loop Execution

- Hosts run in parallel; on each host the items run one after another.
- `when` is evaluated for each item, with the item variables available.
- A failing item stops the loop on that host and fails the task, unless `ignore_errors: true` is set, in which case the remaining items still run.
- Each item that reports `changed` notifies the task's handlers. The handler still runs once per host at the next flush (see [HANDLERS_GUIDE.md](HANDLERS_GUIDE.md)).

### Registering a loop

`register` on a looped task stores:

```yaml
results:            # one entry per item: that item's result plus the item itself
  - { changed: true, rc: 0, stdout: "...", item: nginx, ... }
changed: true       # any item changed
failed: false       # any item failed
skipped: false      # true only when the list was empty
```

The item is stored under the item variable's name (`item`, or the `loop_var` name).

```yaml
- name: Check services
  command: systemctl is-active {{ item }}
  loop: [nginx, redis]
  register: svc
  ignore_errors: true

- name: Show inactive services
  debug:
    msg: "{{ r.item }} is not active"
  loop: "{{ svc.results }}"
  loop_control:
    loop_var: r
  when: r.failed
```

### Conditions per item

```yaml
- name: Install packages selectively
  package:
    name: "{{ item.package }}"
    state: present
  loop:
    - { package: nginx, install: true }
    - { package: apache2, install: false }
    - { package: haproxy, install: true }
  when: item.install
```

Only nginx and haproxy are installed.

---

## Playbook Examples

### Directory structure

```yaml
- name: Create application directories
  hosts: app_servers
  tasks:
    - name: Create directories
      file:
        path: /opt/app/{{ item }}
        state: directory
        mode: "0755"
      loop: [bin, lib, conf, data, logs, tmp]
```

### Tiered directories with the index

```yaml
- name: Create tiered storage
  file:
    path: /storage/tier{{ loop.index }}/{{ item }}
    state: directory
  loop: [cache, working, archive]
```

Creates `/storage/tier1/cache`, `/storage/tier2/working`, `/storage/tier3/archive`.

### Find and clean up

```yaml
- name: Clean up temporary files
  hosts: servers
  tasks:
    - name: Find temporary files
      find:
        path: /tmp
        pattern: "*.tmp"
        type: file
      register: tmp_files

    - name: Remove them
      file:
        path: "{{ item.path }}"
        state: absent
      loop: "{{ tmp_files.files }}"
```

`find` returns `files` (each with `path` and other attributes) and `matched`. With an empty list the task does nothing.

### Remote-to-remote copies

```yaml
- name: Copy binaries on the remote host
  copy:
    src: /tmp/extracted/{{ item }}
    dest: /usr/local/bin/{{ item }}
    mode: "0755"
    remote_src: true
  loop: [binary1, binary2, binary3]
```

---

## Best Practices

1. **Prefer `loop:`** with a list or expression; use the `items`/`range` map only when you need a range.
2. **Name the item variable** with `loop_control.loop_var` when items are dictionaries or when a later loop iterates over a registered `results` list.
3. **Use `loop.index`** instead of `item_index + 1` for 1-based numbering.
4. **Use `ignore_errors: true`** when later items should run even if one fails, and inspect `results` afterwards.

---

## Troubleshooting

### "loop must specify either items or range"

A `loop:` map has neither `items` nor `range`. Give one, or use `loop:` with a list.

### "loop needs a list"

The `loop` expression did not yield a list (for example a dictionary). Use `dict2items` for dictionaries.

### Template variable undefined

The item variable is `item` unless renamed with `loop_control.loop_var`, `var` or `variable`.

### Invalid range

Valid: `"1-10"`, `"1-10:2"`, `"10-1"`, `"a-z"`, `"a-z:2"`.

Errors:

- `"1-10:0"` or `"1-10:-2"`: step must be positive; use a reverse range (`"10-1:2"`) instead
- `"a-9"`: letters and digits cannot be mixed
- `"1-"`: start and end must not be empty
- negative numbers are not supported; use `loop: "{{ range(-5, 5) | list }}"`

### `$(...)` in module arguments

Only `shell` runs through a shell. In `command` (which quotes each word), `copy`, `file` and similar modules, `$(date)` is taken literally; use a variable such as `{{ ansible_date_time.date }}`.

---

## Reference

| Task keyword | Accepts |
|--------------|---------|
| `loop` | list, expression string, or map (`items`, `range`, `var`/`variable`, `index`) |
| `with_items`, `with_list` | list or expression string |
| `with_dict` | dictionary or expression string |
| `with_sequence` | `start= end= stride= count= format=` |
| `with_<lookup>` | the lookup's terms: a list, or one value |
| `loop_control` | `loop_var`, `index_var` |

## Related Documentation

- [Handlers Guide](HANDLERS_GUIDE.md)
- [Filters Guide](FILTERS_GUIDE.md)
- [Variables Cheatsheet](VARIABLES_CHEATSHEET.md)
