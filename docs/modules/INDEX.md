# Alphabetical Index of Onigirazu Modules

Complete alphabetical reference of all built-in modules in Onigirazu. Click on any module name to view its full documentation.

## A

- **[apt](README.md#apt)** - Manage packages on Debian/Ubuntu systems using apt
- **[archive](README.md#archive)** - Creates a compressed archive of one or more files or directories
- **[authorized_key](README.md#authorized_key)** - Manage SSH authorized keys for user accounts

## B

- **[blockinfile](README.md#blockinfile)** - Insert/update/remove a text block in a file using markers

## C

- **[command](README.md#command)** - Executes commands on remote hosts
- **[config](README.md#config)** - Manage configuration files with validation and backup
- **[copy](README.md#copy)** - Copy files to remote locations
- **[cron](README.md#cron)** - Manage cron jobs and crontab files

## D

- **[debug](README.md#debug)** - Prints debug messages
- **[docker_compose](README.md#docker_compose)** - Manage Docker Compose applications
- **[docker_container](README.md#docker_container)** - Manage Docker containers
- **[docker_image](README.md#docker_image)** - Manage Docker images

## F

- **[facts](README.md#facts)** - Gather system facts and information
- **[fail](README.md#fail)** - Fail playbook execution with a custom message
- **[fetch](README.md#fetch)** - Fetch files from remote hosts to local machine
- **[file](README.md#file)** - Manages files and directories
- **[find](README.md#find)** - Search for files matching patterns
- **[firewall](README.md#firewall)** - Manage firewall rules (UFW, firewalld, iptables)

## G

- **[get_url](README.md#get_url)** - Download files from HTTP, HTTPS, or FTP URLs
- **[git](README.md#git)** - Manages Git repositories
- **[group](README.md#group)** - Manages system groups

## L

- **[lineinfile](README.md#lineinfile)** - Manages lines in text files

## M

- **[mongodb](README.md#mongodb)** - Manage MongoDB databases and users
- **[mount](README.md#mount)** - Control active and persistent filesystem mounts
- **[mysql_db](README.md#mysql_db)** - Manage MySQL databases
- **[mysql_user](README.md#mysql_user)** - Manage MySQL users and permissions

## P

- **[package](README.md#package)** - Unified package management with advanced features
- **[pause](README.md#pause)** - Pause playbook execution for a specified duration or until user input
- **[ping](README.md#ping)** - Tests connectivity to hosts
- **[podman](README.md#podman)** - Manage Podman containers
- **[postgresql_db](README.md#postgresql_db)** - Manage PostgreSQL databases
- **[postgresql_user](README.md#postgresql_user)** - Manage PostgreSQL users and roles

## R

- **[reboot](README.md#reboot)** - Reboot the system, with optional delay and pre-reboot checks

## S

- **[script](README.md#script)** - Execute a local script on the remote host
- **[service](README.md#service)** - Manage system services
- **[set_fact](README.md#set_fact)** - Sets facts (variables) for the current host
- **[shell](README.md#shell)** - Executes commands on remote hosts
- **[stat](README.md#stat)** - Retrieves file or directory status
- **[sysctl](README.md#sysctl)** - Manage kernel parameters via sysctl
- **[systemd](README.md#systemd)** - Manage systemd services, units, and timers

## T

- **[template](README.md#template)** - Processes Jinja2-like templates with advanced features and creates files

## U

- **[uri](README.md#uri)** - Make HTTP/HTTPS requests to web services and APIs
- **[user](README.md#user)** - Manages system users

## W

- **[wait_for](README.md#wait_for)** - Wait for a specific condition to be met before continuing

## Y

- **[yum](README.md#yum)** - Manage packages on RedHat/CentOS/Fedora systems using yum

---

## Quick Navigation

For detailed documentation on each module, see [Core Modules Documentation](README.md).

### By Category

- **System Modules**: [facts](README.md#facts), [command](README.md#command), [shell](README.md#shell), [script](README.md#script)
- **File System**: [file](README.md#file), [copy](README.md#copy), [fetch](README.md#fetch), [find](README.md#find), [template](README.md#template), [lineinfile](README.md#lineinfile), [blockinfile](README.md#blockinfile)
- **Configuration**: [config](README.md#config), [lineinfile](README.md#lineinfile), [template](README.md#template)
- **Service Management**: [service](README.md#service), [systemd](README.md#systemd), [reboot](README.md#reboot)
- **Package Management**: [apt](README.md#apt), [yum](README.md#yum), [package](README.md#package)
- **Network**: [uri](README.md#uri), [get_url](README.md#get_url), [firewall](README.md#firewall)
- **Version Control**: [git](README.md#git)
- **Scheduled Jobs**: [cron](README.md#cron)
- **Security**: [authorized_key](README.md#authorized_key), [firewall](README.md#firewall), [user](README.md#user), [group](README.md#group)
- **Container Management**: [docker_container](README.md#docker_container), [docker_image](README.md#docker_image), [docker_compose](README.md#docker_compose), [podman](README.md#podman)
- **Database Management**: [mysql_db](README.md#mysql_db), [mysql_user](README.md#mysql_user), [postgresql_db](README.md#postgresql_db), [postgresql_user](README.md#postgresql_user), [mongodb](README.md#mongodb)
- **System Control**: [sysctl](README.md#sysctl), [mount](README.md#mount)
- **Utilities**: [debug](README.md#debug), [pause](README.md#pause), [fail](README.md#fail), [set_fact](README.md#set_fact), [stat](README.md#stat), [wait_for](README.md#wait_for), [ping](README.md#ping), [archive](README.md#archive)

### Total Modules: 45

---

*Last updated: Auto-generated from module registry*
