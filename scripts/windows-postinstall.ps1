# Post-installation script for Onigirazu (Windows)
# This script runs after Windows installer/package manager installation
# Requires: PowerShell 5.0+ or PowerShell Core

param(
    [string]$InstallDir = "C:\Program Files\Onigirazu"
)

$ErrorActionPreference = "Stop"

# Define paths
$AppDataPath = [Environment]::GetFolderPath("ApplicationData")
$ProgramDataPath = [Environment]::GetFolderPath("CommonApplicationData")

$UserConfigDir = Join-Path $AppDataPath "Onigirazu"
$SystemConfigDir = Join-Path $ProgramDataPath "Onigirazu"

$UserConfigFile = Join-Path $UserConfigDir "onigirazu.yml"
$SystemConfigFile = Join-Path $SystemConfigDir "onigirazu.yml"

$DefaultConfigTemplate = Join-Path $InstallDir "examples\onigirazu.default.yml"

function Ensure-Directory {
    param([string]$Path)

    if (-not (Test-Path $Path)) {
        New-Item -ItemType Directory -Path $Path -Force | Out-Null
        Write-Host "✓ Created directory: $Path"
    }
}

function Create-DefaultConfig {
    param(
        [string]$ConfigFile,
        [string]$TemplateFile,
        [string]$ConfigType
    )

    if (Test-Path $TemplateFile) {
        if (-not (Test-Path $ConfigFile)) {
            Copy-Item $TemplateFile $ConfigFile -Force
            Write-Host "✓ Created $ConfigType configuration at: $ConfigFile"
        }
        else {
            Write-Host "ℹ $ConfigType configuration already exists: $ConfigFile"
        }
    }
    else {
        Write-Host "⚠ Warning: Configuration template not found at: $TemplateFile"
    }
}

# Main installation logic
Write-Host ""
Write-Host "═══════════════════════════════════════════════════════════"
Write-Host "Onigirazu Post-Installation Setup (Windows)"
Write-Host "═══════════════════════════════════════════════════════════"
Write-Host ""

# Determine which config to create (system requires admin)
$IsAdmin = [Security.Principal.WindowsIdentity]::GetCurrent().Owner.IsWellKnown([Security.Principal.WellKnownSidType]::BuiltinAdministratorsSid)

if ($IsAdmin) {
    Write-Host "✓ Running with administrator privileges"
    Write-Host ""
    Write-Host "Setting up system-wide configuration..."

    Ensure-Directory $SystemConfigDir
    Create-DefaultConfig $SystemConfigFile $DefaultConfigTemplate "System"

    Write-Host ""
    Write-Host "Setting up user configuration..."

    Ensure-Directory $UserConfigDir
    Create-DefaultConfig $UserConfigFile $DefaultConfigTemplate "User"
}
else {
    Write-Host "ℹ Running without administrator privileges"
    Write-Host "Setting up user configuration only (use admin for system-wide)..."
    Write-Host ""

    Ensure-Directory $UserConfigDir
    Create-DefaultConfig $UserConfigFile $DefaultConfigTemplate "User"
}

Write-Host ""
Write-Host "✓ Installation complete!"
Write-Host ""
Write-Host "📋 Configuration Locations:"
Write-Host "  System (requires admin): $SystemConfigFile"
Write-Host "  User (recommended):      $UserConfigFile"
Write-Host ""
Write-Host "📁 Available templates in: $InstallDir\examples\"
Write-Host "  • onigirazu.default.yml     (full-featured)"
Write-Host "  • onigirazu.minimal.yml     (minimal setup)"
Write-Host "  • onigirazu.production.yml  (hardened)"
Write-Host "  • onigirazu.docker.yml      (containers)"
Write-Host ""
Write-Host "🚀 To get started:"
Write-Host "  1. Verify: onigirazu --version"
Write-Host "  2. Edit config file using your editor"
Write-Host "  3. Run: onigirazu --help"
Write-Host ""
Write-Host "📖 Documentation: https://github.com/onigirazu-cfg/onigirazu/docs/"
Write-Host "💬 Support: https://github.com/onigirazu-cfg/onigirazu/issues"
Write-Host ""
Write-Host "═══════════════════════════════════════════════════════════"
