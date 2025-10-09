# Package Module - Quick Reference Guide

**Version:** 2.0
**Last Updated:** 2025-01-28

---

## 🚀 Quick Start

### Import

```go
import "github.com/onigirazu-cfg/onigirazu/internal/modules"
```

### Initialize Module

```go
module := modules.NewUnifiedPackageModule(config)
```

---

## 📦 Core Operations

### Install Package

```go
err := module.Install(ctx, "nginx", "latest")
```

### Remove Package

```go
err := module.Remove(ctx, "nginx")
```

### Update Package

```go
err := module.Update(ctx, "nginx")
```

### Upgrade Package

```go
err := module.Upgrade(ctx, "nginx")
```

### Install Multiple

```go
packages := []string{"nginx", "php-fpm", "mysql-server"}
err := module.InstallMultiple(ctx, packages)
```

### Remove Multiple

```go
packages := []string{"nginx", "php-fpm"}
err := module.RemoveMultiple(ctx, packages)
```

---

## 🔍 Query Operations (NEW)

### Search Packages

```go
packages, err := module.Search(ctx, "python")
for _, pkg := range packages {
    fmt.Printf("%s - %s\n", pkg.Name, pkg.Description)
}
```

### List Installed Packages

```go
packages, err := module.ListInstalled(ctx)
for _, pkg := range packages {
    fmt.Printf("%s: %s\n", pkg.Name, pkg.Version)
}
```

### List Upgradable Packages

```go
packages, err := module.ListUpgradable(ctx)
for _, pkg := range packages {
    fmt.Printf("%s: %s → %s\n", pkg.Name, pkg.Version, pkg.NewVersion)
}
```

---

## 🧹 Maintenance Operations (NEW)

### Clean Cache

```go
err := module.Clean(ctx)
```

### Auto Remove Orphans

```go
orphans, err := module.AutoRemove(ctx)
fmt.Printf("Orphaned packages: %v\n", orphans)
```

### Verify Integrity

```go
err := module.VerifyIntegrity(ctx)
if err != nil {
    fmt.Printf("Integrity issues: %v\n", err)
}
```

---

## 📸 Snapshot Operations (NEW)

### Create Snapshot

```go
snapshot, err := module.CreateSnapshot("Before major upgrade")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Snapshot ID: %s\n", snapshot.ID)
fmt.Printf("Packages: %d\n", len(snapshot.Packages))
fmt.Printf("Checksum: %s\n", snapshot.Checksum)
```

### Restore Snapshot

```go
err := module.RestoreSnapshot(ctx, snapshotID)
if err != nil {
    log.Fatal(err)
}
fmt.Println("System restored successfully")
```

### Safe Update Pattern

```go
// Create snapshot
snapshot, err := module.CreateSnapshot("Before update")
if err != nil {
    return err
}

// Try update
err = module.Upgrade(ctx, "critical-package")
if err != nil {
    // Rollback on failure
    module.RestoreSnapshot(ctx, snapshot.ID)
    return fmt.Errorf("update failed, rolled back: %w", err)
}

fmt.Println("Update successful")
```

---

## 👥 Package Group Operations (NEW)

### Define Package Group

```go
webStack := modules.PackageGroup{
    Name:        "web-server",
    Description: "Complete web server stack",
    Packages:    []string{"nginx", "php-fpm", "mysql-server"},
    Optional:    []string{"redis", "memcached"},
}
```

### Install Package Group

```go
err := module.InstallGroup(ctx, webStack)
if err != nil {
    log.Fatal(err)
}
fmt.Println("Web stack installed")
```

### Remove Package Group

```go
err := module.RemoveGroup(ctx, webStack)
if err != nil {
    log.Fatal(err)
}
fmt.Println("Web stack removed")
```

### Common Package Groups

#### Development Tools

```go
devTools := modules.PackageGroup{
    Name:     "dev-tools",
    Packages: []string{"git", "gcc", "make"},
    Optional: []string{"vim", "tmux"},
}
```

#### Database Stack

```go
dbStack := modules.PackageGroup{
    Name:     "database",
    Packages: []string{"postgresql", "postgresql-contrib"},
    Optional: []string{"pgadmin4"},
}
```

#### Monitoring Stack

```go
monitoring := modules.PackageGroup{
    Name:     "monitoring",
    Packages: []string{"prometheus", "grafana"},
    Optional: []string{"alertmanager"},
}
```

---

## 🏥 Health Check Operations (NEW)

### Perform Health Check

```go
result, err := module.PerformHealthCheck(ctx)
if err != nil {
    log.Fatal(err)
}

// Check overall health
if !result.Healthy {
    fmt.Println("⚠️  System is unhealthy")
}

// Display issues
if len(result.Issues) > 0 {
    fmt.Println("Issues:")
    for _, issue := range result.Issues {
        fmt.Printf("  - %s\n", issue)
    }
}

// Display warnings
if len(result.Warnings) > 0 {
    fmt.Println("Warnings:")
    for _, warning := range result.Warnings {
        fmt.Printf("  - %s\n", warning)
    }
}

// Display recommendations
if len(result.Recommendations) > 0 {
    fmt.Println("Recommendations:")
    for _, rec := range result.Recommendations {
        fmt.Printf("  - %s\n", rec)
    }
}

// Display orphan packages
if len(result.OrphanPackages) > 0 {
    fmt.Printf("Orphan packages: %d\n", len(result.OrphanPackages))
}
```

### Automated Health Monitoring

```go
func monitorHealth(ctx context.Context, module *modules.UnifiedPackageModule) {
    ticker := time.NewTicker(1 * time.Hour)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            result, err := module.PerformHealthCheck(ctx)
            if err != nil {
                log.Printf("Health check failed: %v", err)
                continue
            }

            if !result.Healthy {
                // Send alert
                sendAlert("Package system unhealthy", result.Issues)
            }

        case <-ctx.Done():
            return
        }
    }
}
```

---

## 📝 Audit Logging (NEW)

### Log Audit Entry

```go
entry := modules.AuditEntry{
    Timestamp: time.Now(),
    Operation: "install",
    User:      "admin",
    Packages:  []string{"nginx"},
    Success:   true,
    Metadata: map[string]interface{}{
        "version": "1.18.0",
        "source":  "apt",
    },
}

err := module.LogAuditEntry(entry)
```

### Audit Pattern

```go
func auditedInstall(ctx context.Context, module *modules.UnifiedPackageModule, pkg string) error {
    // Create audit entry
    entry := modules.AuditEntry{
        Timestamp: time.Now(),
        Operation: "install",
        User:      getCurrentUser(),
        Packages:  []string{pkg},
    }

    // Perform operation
    err := module.Install(ctx, pkg, "latest")

    // Update audit entry
    entry.Success = (err == nil)
    if err != nil {
        entry.Error = err.Error()
    }

    // Log audit
    module.LogAuditEntry(entry)

    return err
}
```

---

## ℹ️ Information Operations

### Check if Installed

```go
installed, err := module.IsInstalled(ctx, "nginx")
if installed {
    fmt.Println("nginx is installed")
}
```

### Get Package Version

```go
version, err := module.GetVersion(ctx, "nginx")
fmt.Printf("nginx version: %s\n", version)
```

### Get Package Info

```go
info, err := module.GetInfo(ctx, "nginx")
fmt.Printf("Name: %s\n", info.Name)
fmt.Printf("Version: %s\n", info.Version)
fmt.Printf("Description: %s\n", info.Description)
fmt.Printf("Size: %d bytes\n", info.Size)
```

### List Available Versions

```go
versions, err := module.ListVersions(ctx, "nginx")
for _, version := range versions {
    fmt.Println(version)
}
```

### Get Dependencies

```go
deps, err := module.GetDependencies(ctx, "nginx")
fmt.Printf("Dependencies: %v\n", deps)
```

### Verify Checksum

```go
valid, err := module.VerifyChecksum(ctx, "nginx", "1.18.0")
if valid {
    fmt.Println("Checksum verified")
}
```

---

## 🎯 Common Patterns

### Pattern 1: Safe System Upgrade

```go
func safeSystemUpgrade(ctx context.Context, module *modules.UnifiedPackageModule) error {
    // 1. Health check before
    before, err := module.PerformHealthCheck(ctx)
    if err != nil {
        return err
    }
    if !before.Healthy {
        return fmt.Errorf("system unhealthy before upgrade")
    }

    // 2. Create snapshot
    snapshot, err := module.CreateSnapshot("Before system upgrade")
    if err != nil {
        return err
    }

    // 3. Get upgradable packages
    packages, err := module.ListUpgradable(ctx)
    if err != nil {
        return err
    }

    // 4. Upgrade all packages
    for _, pkg := range packages {
        err = module.Upgrade(ctx, pkg.Name)
        if err != nil {
            // Rollback on error
            module.RestoreSnapshot(ctx, snapshot.ID)
            return fmt.Errorf("upgrade failed, rolled back: %w", err)
        }
    }

    // 5. Health check after
    after, err := module.PerformHealthCheck(ctx)
    if err != nil || !after.Healthy {
        // Rollback if unhealthy
        module.RestoreSnapshot(ctx, snapshot.ID)
        return fmt.Errorf("system unhealthy after upgrade, rolled back")
    }

    // 6. Clean up
    module.Clean(ctx)
    module.AutoRemove(ctx)

    return nil
}
```

### Pattern 2: Conditional Package Installation

```go
func ensurePackages(ctx context.Context, module *modules.UnifiedPackageModule, packages []string) error {
    for _, pkg := range packages {
        installed, err := module.IsInstalled(ctx, pkg)
        if err != nil {
            return err
        }

        if !installed {
            fmt.Printf("Installing %s...\n", pkg)
            err = module.Install(ctx, pkg, "latest")
            if err != nil {
                return err
            }
        } else {
            fmt.Printf("%s already installed\n", pkg)
        }
    }
    return nil
}
```

### Pattern 3: Package Search and Install

```go
func searchAndInstall(ctx context.Context, module *modules.UnifiedPackageModule, query string) error {
    // Search for packages
    packages, err := module.Search(ctx, query)
    if err != nil {
        return err
    }

    if len(packages) == 0 {
        return fmt.Errorf("no packages found for: %s", query)
    }

    // Display options
    fmt.Println("Found packages:")
    for i, pkg := range packages {
        fmt.Printf("%d. %s - %s\n", i+1, pkg.Name, pkg.Description)
    }

    // Install first match (or let user choose)
    return module.Install(ctx, packages[0].Name, "latest")
}
```

### Pattern 4: Maintenance Routine

```go
func maintenanceRoutine(ctx context.Context, module *modules.UnifiedPackageModule) error {
    fmt.Println("Starting maintenance routine...")

    // 1. Health check
    result, err := module.PerformHealthCheck(ctx)
    if err != nil {
        return err
    }
    fmt.Printf("Health: %v\n", result.Healthy)

    // 2. Clean cache
    fmt.Println("Cleaning cache...")
    err = module.Clean(ctx)
    if err != nil {
        return err
    }

    // 3. Remove orphans
    fmt.Println("Removing orphans...")
    orphans, err := module.AutoRemove(ctx)
    if err != nil {
        return err
    }
    fmt.Printf("Removed %d orphan packages\n", len(orphans))

    // 4. Verify integrity
    fmt.Println("Verifying integrity...")
    err = module.VerifyIntegrity(ctx)
    if err != nil {
        return err
    }

    fmt.Println("Maintenance complete")
    return nil
}
```

### Pattern 5: Upgrade with Notification

```go
func upgradeWithNotification(ctx context.Context, module *modules.UnifiedPackageModule) error {
    // Get upgradable packages
    packages, err := module.ListUpgradable(ctx)
    if err != nil {
        return err
    }

    if len(packages) == 0 {
        fmt.Println("All packages up to date")
        return nil
    }

    // Notify about available upgrades
    fmt.Printf("Available upgrades: %d\n", len(packages))
    for _, pkg := range packages {
        fmt.Printf("  %s: %s → %s\n", pkg.Name, pkg.Version, pkg.NewVersion)
    }

    // Create snapshot before upgrade
    snapshot, err := module.CreateSnapshot("Before batch upgrade")
    if err != nil {
        return err
    }

    // Upgrade all
    failed := []string{}
    for _, pkg := range packages {
        err = module.Upgrade(ctx, pkg.Name)
        if err != nil {
            failed = append(failed, pkg.Name)
        }
    }

    // Check results
    if len(failed) > 0 {
        fmt.Printf("Failed to upgrade: %v\n", failed)
        fmt.Println("Rolling back...")
        module.RestoreSnapshot(ctx, snapshot.ID)
        return fmt.Errorf("upgrade failed for %d packages", len(failed))
    }

    fmt.Println("All packages upgraded successfully")
    return nil
}
```

---

## 🔧 Configuration

### Module Configuration

```go
config := modules.PackageModuleConfig{
    Type:         "apt",           // Package manager type
    CacheTTL:     5 * time.Minute, // Cache TTL
    LockFile:     "/var/lock/pkg", // Lock file path
    MaxRetries:   3,               // Max retry attempts
    RetryDelay:   1 * time.Second, // Retry delay
}

module := modules.NewUnifiedPackageModule(config)
```

---

## 🐛 Error Handling

### Basic Error Handling

```go
err := module.Install(ctx, "nginx", "latest")
if err != nil {
    log.Printf("Installation failed: %v", err)
    return err
}
```

### Advanced Error Handling

```go
err := module.Install(ctx, "nginx", "latest")
if err != nil {
    switch {
    case errors.Is(err, context.Canceled):
        log.Println("Operation canceled")
    case errors.Is(err, context.DeadlineExceeded):
        log.Println("Operation timed out")
    default:
        log.Printf("Installation failed: %v", err)
    }
    return err
}
```

---

## ⏱️ Context Usage

### With Timeout

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()

err := module.Install(ctx, "nginx", "latest")
```

### With Cancellation

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// Cancel on signal
go func() {
    <-sigChan
    cancel()
}()

err := module.Install(ctx, "nginx", "latest")
```

---

## 📊 Monitoring

### Basic Monitoring

```go
// Get cache statistics
hits, misses := module.GetCacheStats()
hitRate := float64(hits) / float64(hits+misses) * 100
fmt.Printf("Cache hit rate: %.1f%%\n", hitRate)
```

### Health Monitoring

```go
result, _ := module.PerformHealthCheck(ctx)
fmt.Printf("Healthy: %v\n", result.Healthy)
fmt.Printf("Issues: %d\n", len(result.Issues))
fmt.Printf("Warnings: %d\n", len(result.Warnings))
```

---

## 🎓 Best Practices

### 1. Always Use Context

```go
// Good
err := module.Install(ctx, "nginx", "latest")

// Bad
err := module.Install(context.Background(), "nginx", "latest")
```

### 2. Create Snapshots Before Major Changes

```go
snapshot, _ := module.CreateSnapshot("Before change")
// ... make changes ...
// Rollback if needed: module.RestoreSnapshot(ctx, snapshot.ID)
```

### 3. Regular Health Checks

```go
// Run health checks periodically
ticker := time.NewTicker(1 * time.Hour)
go func() {
    for range ticker.C {
        module.PerformHealthCheck(ctx)
    }
}()
```

### 4. Use Package Groups for Related Packages

```go
// Good
webStack := PackageGroup{...}
module.InstallGroup(ctx, webStack)

// Less efficient
module.Install(ctx, "nginx", "latest")
module.Install(ctx, "php-fpm", "latest")
module.Install(ctx, "mysql-server", "latest")
```

### 5. Clean Up Regularly

```go
// Periodic cleanup
module.Clean(ctx)
module.AutoRemove(ctx)
```

---

## 📚 Additional Resources

- **Full Documentation:** `PACKAGE_ENHANCEMENT_README.md`
- **Architecture:** `PACKAGE_ARCHITECTURE.md`
- **Detailed Report:** `PACKAGE_MODULE_ENHANCEMENT_REPORT.md`
- **Examples:** See patterns section above

---

**Version:** 2.0
**Last Updated:** 2025-01-28
**Status:** Production Ready
