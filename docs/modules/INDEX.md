# Module Index

All 61 built-in modules, alphabetically. `include_role`/`import_role` and `setup`/`gather_facts` are one module each under two names. Full reference: [Core Modules Documentation](README.md).

## A

- **[apt](README.md#apt)** - Manage packages on Debian/Ubuntu systems using apt
- **[apt_key](README.md#apt_key)** - Add or remove an APT signing key
- **[apt_repository](README.md#apt_repository)** - Add or remove an APT repository
- **[archive](README.md#archive)** - Create a compressed archive of files or directories
- **[assert](README.md#assert)** - Fail when a condition does not hold
- **[async_status](README.md#async_status)** - Status of a task started with async and poll: 0
- **[authorized_key](README.md#authorized_key)** - Manage SSH authorized keys for user accounts

## B

- **[blockinfile](README.md#blockinfile)** - Insert, update or remove a text block in a file using markers

## C

- **[command](README.md#command)** - Run a command on the host, without a shell
- **[config](README.md#config)** - Manage configuration files with validation and backup
- **[copy](README.md#copy)** - Copy files or content to the host
- **[cron](README.md#cron)** - Manage cron jobs and crontab files

## D

- **[debug](README.md#debug)** - Print a message or a variable
- **[docker_compose](README.md#docker_compose)** - Manage Docker Compose applications
- **[docker_container](README.md#docker_container)** - Manage Docker containers
- **[docker_host_info](README.md#docker_host_info)** - Information about the Docker host
- **[docker_image](README.md#docker_image)** - Manage Docker images

## F

- **[fail](README.md#fail)** - Fail the play with a custom message
- **[fetch](README.md#fetch)** - Fetch files from the host to the control machine
- **[file](README.md#file)** - Manage files, directories and links
- **[find](README.md#find)** - Search for files matching patterns
- **[firewall](README.md#firewall)** - Manage firewall rules (UFW, firewalld, iptables)

## G

- **[get_url](README.md#get_url)** - Download files from HTTP, HTTPS or FTP URLs
- **[getent](README.md#getent)** - Read a getent database (passwd, group, hosts, ...)
- **[git](README.md#git)** - Manage Git repositories
- **[group](README.md#group)** - Manage system groups

## H

- **[hostname](README.md#hostname)** - Set the host name

## I

- **[include_role / import_role](README.md#include_role--import_role)** - Run a role's tasks at this point
- **[include_vars](README.md#include_vars)** - Load variables from YAML files on the control machine
- **[ini_file](README.md#ini_file)** - Manage one option of an INI file

## L

- **[lineinfile](README.md#lineinfile)** - Manage lines in text files

## M

- **[meta](README.md#meta)** - Engine actions: flush_handlers, end_host, end_play, noop
- **[mongodb](README.md#mongodb)** - Manage MongoDB databases and users
- **[mount](README.md#mount)** - Control active and persistent filesystem mounts
- **[mysql_db](README.md#mysql_db)** - Manage MySQL databases
- **[mysql_user](README.md#mysql_user)** - Manage MySQL users and permissions

## P

- **[package](README.md#package)** - Install, remove or update packages with the host's package manager
- **[pause](README.md#pause)** - Pause for a duration or until user input
- **[ping](README.md#ping)** - Test connectivity to hosts
- **[pip](README.md#pip)** - Manage Python packages with pip
- **[podman](README.md#podman)** - Manage Podman containers
- **[postgresql_db](README.md#postgresql_db)** - Manage PostgreSQL databases
- **[postgresql_user](README.md#postgresql_user)** - Manage PostgreSQL users and roles

## R

- **[reboot](README.md#reboot)** - Reboot the host, with optional delay and pre-reboot checks
- **[replace](README.md#replace)** - Replace every match of a regular expression in a file

## S

- **[script](README.md#script)** - Copy a local script to the host and run it
- **[service](README.md#service)** - Manage system services
- **[set_fact](README.md#set_fact)** - Set variables for the current host
- **[setup / gather_facts](README.md#setup)** - Gather the host's facts again (filter, fact_path)
- **[shell](README.md#shell)** - Run a command on the host through a shell
- **[slurp](README.md#slurp)** - Read a file from the host (base64)
- **[stat](README.md#stat)** - Retrieve file or directory status
- **[sysctl](README.md#sysctl)** - Manage kernel parameters via sysctl
- **[systemd](README.md#systemd)** - Manage systemd services, units and timers

## T

- **[template](README.md#template)** - Render a Jinja2 template to a file on the host
- **[timezone](README.md#timezone)** - Set the system time zone

## U

- **[ufw](README.md#ufw)** - Manage ufw firewall rules and state
- **[unarchive](README.md#unarchive)** - Extract a tar or zip archive on the host
- **[uri](README.md#uri)** - Make HTTP/HTTPS requests to web services and APIs
- **[user](README.md#user)** - Manage system users

## W

- **[wait_for](README.md#wait_for)** - Wait for a port or file condition before continuing

## Y

- **[yum](README.md#yum)** - Manage packages on RedHat/CentOS/Fedora systems using yum

- **[win_ping / win_command / win_shell](README.md#win_ping-win_command-win_shell)** - Windows hosts over WinRM
- **[win_powershell](README.md#win_powershell)** - PowerShell script with $Ansible.Result and all output streams
- **[win_regedit](README.md#win_regedit)** - Registry keys and values
- **[win_file / win_copy / win_service / win_timezone](README.md#win_file-win_copy-win_service-win_timezone)** - Files, services and time zone on Windows
- **[win_firewall_rule / win_firewall / win_group_membership / win_feature / win_reboot](README.md#win_firewall_rule-win_firewall-win_group_membership-win_feature-win_reboot)** - Firewall, groups, features, reboots on Windows
- **[win_scheduled_task / win_chocolatey](README.md#win_scheduled_task-win_chocolatey)** - Scheduled tasks and Chocolatey packages
- **[win_optional_feature and the disk modules](README.md#win_optional_feature-and-the-disk-modules)** - Optional features, disks, partitions, volumes

## By Category

- **Commands**: [command](README.md#command), [shell](README.md#shell), [script](README.md#script)
- **Files**: [file](README.md#file), [copy](README.md#copy), [fetch](README.md#fetch), [find](README.md#find), [stat](README.md#stat), [template](README.md#template), [lineinfile](README.md#lineinfile), [blockinfile](README.md#blockinfile), [replace](README.md#replace), [ini_file](README.md#ini_file), [config](README.md#config), [archive](README.md#archive), [unarchive](README.md#unarchive), [slurp](README.md#slurp)
- **Packages**: [package](README.md#package), [apt](README.md#apt), [yum](README.md#yum), [pip](README.md#pip), [apt_repository](README.md#apt_repository), [apt_key](README.md#apt_key)
- **Services and system**: [service](README.md#service), [systemd](README.md#systemd), [sysctl](README.md#sysctl), [mount](README.md#mount), [reboot](README.md#reboot), [hostname](README.md#hostname), [timezone](README.md#timezone), [cron](README.md#cron)
- **Users and security**: [user](README.md#user), [group](README.md#group), [authorized_key](README.md#authorized_key), [getent](README.md#getent), [firewall](README.md#firewall), [ufw](README.md#ufw)
- **Network**: [uri](README.md#uri), [get_url](README.md#get_url), [wait_for](README.md#wait_for)
- **Source control**: [git](README.md#git)
- **Containers**: [docker_container](README.md#docker_container), [docker_image](README.md#docker_image), [docker_compose](README.md#docker_compose), [docker_host_info](README.md#docker_host_info), [podman](README.md#podman)
- **Databases**: [mysql_db](README.md#mysql_db), [mysql_user](README.md#mysql_user), [postgresql_db](README.md#postgresql_db), [postgresql_user](README.md#postgresql_user), [mongodb](README.md#mongodb)
- **Playbook control**: [debug](README.md#debug), [set_fact](README.md#set_fact), [assert](README.md#assert), [fail](README.md#fail), [pause](README.md#pause), [ping](README.md#ping), [include_vars](README.md#include_vars), [include_role / import_role](README.md#include_role--import_role), [setup / gather_facts](README.md#setup), [meta](README.md#meta), [async_status](README.md#async_status)
