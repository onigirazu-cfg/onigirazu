# Alphabetical Index of Onigirazu Modules

Complete alphabetical reference of all built-in modules in Onigirazu.

## A

- **apt** - Manage packages on Debian/Ubuntu systems using apt
- **archive** - Creates a compressed archive of one or more files or directories
- **authorized_key** - Manage SSH authorized keys for user accounts

## B

- **blockinfile** - Insert/update/remove a text block in a file using markers

## C

- **command** - Executes commands on remote hosts
- **config** - Manage configuration files with validation and backup
- **copy** - Copy files to remote locations
- **cron** - Manage cron jobs and crontab files

## D

- **debug** - Prints debug messages
- **docker_compose** - Manage Docker Compose applications
- **docker_container** - Manage Docker containers
- **docker_image** - Manage Docker images

## F

- **facts** - Gather system facts and information
- **fail** - Fail playbook execution with a custom message
- **fetch** - Fetch files from remote hosts to local machine
- **file** - Manages files and directories
- **find** - Search for files matching patterns
- **firewall** - Manage firewall rules (UFW, firewalld, iptables)

## G

- **get_url** - Download files from HTTP, HTTPS, or FTP URLs
- **git** - Manages Git repositories
- **group** - Manages system groups

## L

- **lineinfile** - Manages lines in text files

## M

- **mongodb** - Manage MongoDB databases and users
- **mount** - Control active and persistent filesystem mounts
- **mysql_db** - Manage MySQL databases
- **mysql_user** - Manage MySQL users and permissions

## P

- **package** - Unified package management with advanced features
- **pause** - Pause playbook execution for a specified duration or until user input
- **ping** - Tests connectivity to hosts
- **podman** - Manage Podman containers
- **postgresql_db** - Manage PostgreSQL databases
- **postgresql_user** - Manage PostgreSQL users and roles

## R

- **reboot** - Reboot the system, with optional delay and pre-reboot checks

## S

- **script** - Execute a local script on the remote host
- **service** - Manage system services
- **set_fact** - Sets facts (variables) for the current host
- **shell** - Executes commands on remote hosts
- **stat** - Retrieves file or directory status
- **sysctl** - Manage kernel parameters via sysctl
- **systemd** - Manage systemd services, units, and timers

## T

- **template** - Processes Jinja2-like templates with advanced features and creates files

## U

- **uri** - Make HTTP/HTTPS requests to web services and APIs
- **user** - Manages system users

## W

- **wait_for** - Wait for a specific condition to be met before continuing

## Y

- **yum** - Manage packages on RedHat/CentOS/Fedora systems using yum

---

## Quick Navigation

For detailed documentation on each module, see [Core Modules Documentation](README.md).

### By Category

- **System Modules**: [facts](#f), [command](#c), [shell](#s), [script](#s)
- **File System**: [file](#f), [copy](#c), [fetch](#f), [find](#f), [template](#t), [lineinfile](#l), [blockinfile](#b)
- **Configuration**: [config](#c), [lineinfile](#l), [template](#t)
- **Service Management**: [service](#s), [systemd](#s), [reboot](#r)
- **Package Management**: [apt](#a), [yum](#y), [package](#p)
- **Network**: [uri](#u), [get_url](#g), [firewall](#f)
- **Version Control**: [git](#g)
- **Scheduled Jobs**: [cron](#c)
- **Security**: [authorized_key](#a), [firewall](#f), [user](#u), [group](#g)
- **Container Management**: [docker_container](#d), [docker_image](#d), [docker_compose](#d), [podman](#p)
- **Database Management**: [mysql_db](#m), [mysql_user](#m), [postgresql_db](#p), [postgresql_user](#p), [mongodb](#m)
- **System Control**: [sysctl](#s), [mount](#m)
- **Utilities**: [debug](#d), [pause](#p), [fail](#f), [set_fact](#s), [stat](#s), [wait_for](#w)

### Total Modules: 45

---

*Last updated: Auto-generated from module registry*
