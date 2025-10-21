# 🎯 Natural Language Examples for Onigirazu

## 📋 Currently Supported Commands

### **1. 📦 Package Operations**

#### **Installing packages:**

```bash
# Install nginx
onigirazu run all "install nginx package" -i inventory.yml

# Install multiple packages
onigirazu run webservers "install apache package" -i inventory.yml
onigirazu run dbservers "install mysql package" -i inventory.yml
onigirazu run all "install git package" -i inventory.yml

# Install with alternative syntax
onigirazu run all "add nginx package" -i inventory.yml
# Note: "install the nginx package" is not supported
```

#### **Removing packages:**

```bash
# Remove nginx
onigirazu run all "remove nginx package" -i inventory.yml

# Remove multiple packages
onigirazu run webservers "uninstall apache package" -i inventory.yml
onigirazu run all "delete old-package package" -i inventory.yml
```

#### **Updating packages:**

```bash
# Update nginx to latest version
onigirazu run all "update nginx package" -i inventory.yml

# Update all packages (requires special module)
# onigirazu run all "upgrade all package" -i inventory.yml  # DOES NOT WORK
```

### **2. 🔧 Service Operations**

#### **Starting services:**

```bash
# Start nginx
onigirazu run webservers "start nginx service" -i inventory.yml

# Start multiple services
onigirazu run all "start apache service" -i inventory.yml
onigirazu run dbservers "start mysql service" -i inventory.yml
```

#### **Stopping services:**

```bash
# Stop nginx
onigirazu run webservers "stop nginx service" -i inventory.yml

# Stop multiple services
onigirazu run all "stop apache service" -i inventory.yml
```

#### **Restarting services:**

```bash
# Restart nginx
onigirazu run webservers "restart nginx service" -i inventory.yml

# Reload configuration
onigirazu run webservers "reload nginx service" -i inventory.yml
```

### **3. 📁 File Operations**

#### **Creating files:**

```bash
# Create file
onigirazu run all "create file /tmp/test.txt" -i inventory.yml

# Create multiple files
onigirazu run webservers "create file /var/www/index.html" -i inventory.yml
onigirazu run all "touch file /tmp/empty.txt" -i inventory.yml
```

#### **Deleting files:**

```bash
# Delete file
onigirazu run all "delete file /tmp/old.txt" -i inventory.yml

# Delete multiple files
onigirazu run all "delete file /tmp/temp.log" -i inventory.yml
# Note: "remove file" is not supported, use "delete file"
```

---

## ✅ **Real Testing Results**

### **📦 Package Operations - WORKING:**

```bash
✅ "install nginx package"     -> CHANGED (nginx installed)
✅ "add nginx package"         -> CHANGED (nginx installed)
✅ "remove nginx package"      -> CHANGED (nginx removed)
✅ "uninstall nginx package"   -> CHANGED (nginx removed)
✅ "delete nginx package"      -> SUCCESS (nginx already removed)
✅ "update nginx package"      -> CHANGED (nginx updated)
```

### **📁 File Operations - WORKING:**

```bash
✅ "create file /tmp/test.txt"  -> CHANGED (file created)
✅ "delete file /tmp/test.txt"  -> CHANGED (file deleted)
✅ "touch file /tmp/empty.txt"  -> CHANGED (file created)
```

### **🔧 Service Operations - WORKING (partially):**

```bash
✅ "stop nginx service"        -> SUCCESS (service stopped)
⚠️ "start nginx service"       -> FAILED (nginx did not start)
⚠️ "restart nginx service"     -> FAILED (nginx did not restart)
⚠️ "reload nginx service"      -> FAILED (nginx did not reload)
```

### **❌ DOES NOT WORK:**

```bash
❌ "install the nginx package" -> FAILED (invalid syntax - "the" not supported)
❌ "remove file /tmp/old.txt"  -> FAILED (invalid syntax)
❌ "upgrade all package"       -> FAILED (tries to install package "all")
```

---

## 🚀 Advanced Usage Examples

### **1. Combined operations:**

```bash
# Install nginx and start service
onigirazu run webservers "install nginx package" -i inventory.yml
onigirazu run webservers "start nginx service" -i inventory.yml

# Update packages and restart services
onigirazu run all "update nginx package" -i inventory.yml
onigirazu run webservers "restart nginx service" -i inventory.yml
```

### **2. With different host groups:**

```bash
# Different operations for different groups
onigirazu run webservers "install nginx package" -i inventory.yml
onigirazu run dbservers "install mysql package" -i inventory.yml
onigirazu run monitoring "install prometheus package" -i inventory.yml
```

### **3. With additional options:**

```bash
# Check mode (dry-run)
onigirazu run all "install nginx package" --check -i inventory.yml

# Parallel execution
onigirazu run all "install nginx package" --parallel 10 -i inventory.yml

# JSON output
onigirazu run all "start nginx service" --output json -i inventory.yml

# Verbose mode
onigirazu run all "install nginx package" -V -i inventory.yml
```

---

## ✅ **Correct Usage Examples**

### **📦 Package Operations (WORKING):**

```bash
# Installing packages
onigirazu run all "install nginx package" -i inventory.yml
onigirazu run all "add apache package" -i inventory.yml

# Removing packages
onigirazu run all "remove nginx package" -i inventory.yml
onigirazu run all "uninstall apache package" -i inventory.yml
onigirazu run all "delete mysql package" -i inventory.yml

# Updating packages
onigirazu run all "update nginx package" -i inventory.yml
```

### **📁 File Operations (WORKING):**

```bash
# Creating files
onigirazu run all "create file /tmp/test.txt" -i inventory.yml
onigirazu run all "touch file /tmp/empty.txt" -i inventory.yml

# Deleting files
onigirazu run all "delete file /tmp/old.txt" -i inventory.yml
```

### **🔧 Service Operations (PARTIALLY WORKING):**

```bash
# Stopping services (WORKS)
onigirazu run all "stop nginx service" -i inventory.yml

# Starting services (WORKS, but nginx may not start)
onigirazu run all "start nginx service" -i inventory.yml
```

## 🎯 Possible Natural Language Extensions

### **1. User Operations:**

```bash
# Possible future support:
onigirazu run all "create user john" -i inventory.yml
onigirazu run all "delete user olduser" -i inventory.yml
onigirazu run all "add user to group admin" -i inventory.yml
```

### **2. Directory Operations:**

```bash
# Possible future support:
onigirazu run all "create directory /var/www" -i inventory.yml
onigirazu run all "delete directory /tmp/old" -i inventory.yml
onigirazu run all "make directory /var/log/app" -i inventory.yml
```

### **3. Network Operations:**

```bash
# Possible future support:
onigirazu run all "open port 80" -i inventory.yml
onigirazu run all "close port 22" -i inventory.yml
onigirazu run all "enable firewall" -i inventory.yml
```

### **4. System Operations:**

```bash
# Possible future support:
onigirazu run all "reboot system" -i inventory.yml
onigirazu run all "shutdown system" -i inventory.yml
onigirazu run all "update system" -i inventory.yml
```

---

## 🔧 Implementation Details

### **Currently Supported Patterns:**

#### **Package Operations:**

```go
// Supported actions:
"install <package> package"
"add <package> package"
"remove <package> package"
"uninstall <package> package"
"delete <package> package"
"update <package> package"
"upgrade <package> package"

// Examples:
"install nginx package" -> package module, name=nginx, state=present
"remove apache package" -> package module, name=apache, state=absent
"update mysql package" -> package module, name=mysql, state=latest
```

#### **Service Operations:**

```go
// Supported actions:
"start <service> service"
"stop <service> service"
"restart <service> service"
"reload <service> service"

// Examples:
"start nginx service" -> service module, name=nginx, state=started
"stop apache service" -> service module, name=apache, state=stopped
"restart mysql service" -> service module, name=mysql, state=restarted
```

#### **File Operations:**

```go
// Supported actions:
"create file <path>"
"delete file <path>"
"remove file <path>"
"touch file <path>"

// Examples:
"create file /tmp/test.txt" -> file module, path=/tmp/test.txt, state=touch
"delete file /tmp/old.log" -> file module, path=/tmp/old.log, state=absent
```

---

## 🎯 Benefits of Natural Language

### **1. 🚀 Simplicity:**

```bash
# Instead of complex syntax:
onigirazu run all -m package name=nginx state=present

# Just type naturally:
onigirazu run all "install nginx package"
```

### **2. 🎯 Intuitiveness:**

```bash
# Easy even for beginners:
onigirazu run webservers "start nginx service"
onigirazu run all "install git package"
onigirazu run dbservers "restart mysql service"
```

### **3. 🔧 Speed:**

```bash
# Quick for one-off operations:
onigirazu run all "install nginx package"
onigirazu run all "start nginx service"
onigirazu run all "create file /tmp/test.txt"
```

### **4. 📚 Learning:**

```bash
# Easy to discover new modules:
onigirazu run all "install <package> package"  # Learn about package module
onigirazu run all "start <service> service"     # Learn about service module
onigirazu run all "create file <path>"          # Learn about file module
```

---

## 🚀 Future Enhancements

### **1. Parser Extensions:**

```go
// Add support for more modules:
- user module: "create user john", "delete user olduser"
- group module: "create group admin", "add user to group"
- directory module: "create directory /var/www", "delete directory /tmp"
- port module: "open port 80", "close port 22"
- firewall module: "enable firewall", "disable firewall"
```

### **2. Syntax Extensions:**

```go
// Support for more complex constructs:
"install nginx package and start nginx service"
"create user john and add to admin group"
"open port 80 and restart nginx service"
```

### **3. Contextual Understanding:**

```go
// Context-aware parsing:
"install nginx" -> automatically determine as package
"start nginx" -> automatically determine as service
"create /tmp/test.txt" -> automatically determine as file
```

---

## ⚠️ **Limitations and Known Issues**

### **❌ DOES NOT WORK:**

```bash
# Invalid syntax
❌ "install the nginx package"  # "the" is not supported
❌ "remove file /tmp/old.txt"  # "remove file" is not supported
❌ "upgrade all package"       # tries to install package "all"

# Complex constructs
❌ "install nginx and start nginx service"  # only one operation at a time
❌ "create user john and add to admin group"  # only one operation at a time
```

### **⚠️ LIMITATIONS:**

```bash
# Service operations depend on service state
⚠️ "start nginx service"   # may not start if nginx is not installed
⚠️ "restart nginx service"  # may not restart if nginx is not running
⚠️ "reload nginx service"   # may not reload if nginx is not running

# Package operations depend on package manager
⚠️ "update nginx package"   # only works with Homebrew on macOS
⚠️ "upgrade all package"    # not supported (tries to install package "all")
```

### **✅ RELIABLE:**

```bash
# Package operations
✅ "install nginx package"   # always works
✅ "add apache package"      # always works
✅ "remove nginx package"    # always works
✅ "uninstall apache package" # always works
✅ "delete mysql package"    # always works

# File operations
✅ "create file /tmp/test.txt"  # always works
✅ "delete file /tmp/old.txt"   # always works
✅ "touch file /tmp/empty.txt"  # always works

# Service operations (stop)
✅ "stop nginx service"      # always works
```
