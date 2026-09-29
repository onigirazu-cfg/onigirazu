# Natural Language Ad-hoc Commands

`onigirazu run` turns a few fixed phrases into a module call (see
[ADHOC_GUIDE.md](../ADHOC_GUIDE.md)). The whole string is lower-cased first, including
names and paths.

## Recognised phrases

| Phrase | Module call |
|--------|-------------|
| `install NAME package`, `add NAME package` | `package name=NAME state=present` |
| `remove NAME package`, `uninstall NAME package`, `delete NAME package` | `package name=NAME state=absent` |
| `update NAME package`, `upgrade NAME package` | `package name=NAME state=latest` |
| `start NAME service` | `service name=NAME state=started` |
| `stop NAME service` | `service name=NAME state=stopped` |
| `restart NAME service` | `service name=NAME state=restarted` |
| `reload NAME service` | `service name=NAME state=reloaded` |
| `create file PATH`, `touch file PATH` | `file path=PATH state=touch` |
| `delete file PATH` | `file path=PATH state=absent` |

```bash
onigirazu run webservers "install nginx package" -b -i inventory.yml
onigirazu run webservers "restart nginx service" -b -i inventory.yml
onigirazu run all "create file /tmp/test.txt" -i inventory.yml
onigirazu run all "install nginx package" --check -i inventory.yml
```

All `run` flags (`--check`, `-b`, `-o json`, `--parallel`, ...) apply.

## Pitfalls

- NAME must come right before `package` / `service`. `install package nginx` installs a
  package named `install`.
- Any phrase starting with `create`, `touch` or `delete` that is not a package or service
  phrase becomes a `file` call on its last word: `create user john` touches a file `john`,
  `delete nginx` removes a file `nginx`.
- `upgrade all package` targets a package named `all`.
- Anything else is run as a plain command through the `command` module, for example
  `install the nginx package`, `install nginx`, `remove file /tmp/x`, `start nginx`.
- One operation per string; `and` is not understood.

Use `-m` for anything beyond these phrases.
