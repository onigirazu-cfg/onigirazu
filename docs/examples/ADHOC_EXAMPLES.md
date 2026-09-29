# Ad-hoc Commands - Examples

Syntax and flags: [ADHOC_GUIDE.md](../ADHOC_GUIDE.md).

The `command` module runs a program without a shell (no pipes, `&&`, `$VAR`); use
`-m shell cmd="…"` for those. Give the command of `command` as `-a "uptime"` or as a plain
string (`run all "uptime"`); `command="uptime"` is not a valid argument.

## Basics

```bash
onigirazu run all -m ping -i inventory.yml
onigirazu run all "uptime" -i inventory.yml
onigirazu run all -m command -a "uname -a" -i inventory.yml
onigirazu run all -m debug msg="Hello World" -i inventory.yml
onigirazu run all -m shell cmd="ps aux | head -20" -i inventory.yml
```

## Packages and services

```bash
onigirazu run webservers -m package name=nginx state=present -b -i inventory.yml
onigirazu run all -m package name=vim state=latest -b -i inventory.yml
onigirazu run webservers -m package name=apache2 state=absent -b -i inventory.yml

onigirazu run webservers -m service name=nginx state=restarted -b -i inventory.yml
onigirazu run webservers -m shell cmd="systemctl status nginx" -i inventory.yml

# Natural language
onigirazu run all "install nginx package" -b -i inventory.yml
onigirazu run all "remove vim package" -b -i inventory.yml
onigirazu run webservers "restart nginx service" -b -i inventory.yml
```

## Files and users

```bash
onigirazu run all -m file path=/tmp/test.txt state=touch -i inventory.yml
onigirazu run all -m file path=/tmp/mydir state=directory mode=0755 -i inventory.yml
onigirazu run all -m file path=/tmp/test.txt state=absent -i inventory.yml
onigirazu run webservers -m copy src=nginx.conf dest=/etc/nginx/nginx.conf -b -i inventory.yml
onigirazu run all -m get_url url=https://example.com/file.txt dest=/tmp/file.txt -i inventory.yml
onigirazu run all -m user name=testuser state=present -b -i inventory.yml

# Natural language
onigirazu run all "create file /tmp/test.txt" -i inventory.yml
onigirazu run all "delete file /tmp/test.txt" -i inventory.yml
```

## Other input forms

```bash
onigirazu run all "package:name=nginx,state=present" -i inventory.yml
onigirazu run all '{"module":"command","args":{"command":"uptime"}}' -i inventory.yml
onigirazu run all 'module: command
args:
  command: uptime' -i inventory.yml
```

## Targets

```bash
onigirazu run all -m ping -i inventory.yml
onigirazu run webservers -m ping -i inventory.yml
onigirazu run server1 -m ping -i inventory.yml
onigirazu run webservers,databases -m ping -i inventory.yml
onigirazu run all -m ping -i web1,web2
```

## Execution options

```bash
# Hosts in parallel (default 10)
onigirazu run all "uptime" --parallel 1 -i inventory.yml

# Check mode: modules without check support are skipped
onigirazu run all -m package name=nginx state=present --check -i inventory.yml

# Per-host timeout (default 30s)
onigirazu run all -m shell cmd="slow_script.sh" --timeout 300s -i inventory.yml

# Variables
onigirazu run all -m debug msg="{{ greeting }}" -e greeting=hi -i inventory.yml

# Detailed results
onigirazu run all "uptime" -V -i inventory.yml
```

## Output

```bash
onigirazu run all -m ping -o json -i inventory.yml | jq '.success'
onigirazu run all -m ping -o yaml -i inventory.yml
onigirazu run all -m ping -o table -i inventory.yml
```

Log lines go to stderr; stdout holds only the results. The exit code is 1 when any host
failed.
