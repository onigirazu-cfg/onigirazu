package modules

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/onigirazu-cfg/onigirazu/internal/winrm"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// NewWinOptionalFeatureModule creates win_optional_feature
func NewWinOptionalFeatureModule() types.Module {
	return newWinJSONModule("win_optional_feature", "Enable or disable Windows optional features", winOptionalFeatureScript)
}

func winOptionalFeatureScript(_ context.Context, _ winrm.Runner, args map[string]interface{}) (string, error) {
	if args["name"] == nil {
		return "", fmt.Errorf("win_optional_feature needs name")
	}
	state := getStringArg(args, "state", "present")
	if state != "present" && state != "absent" {
		return "", fmt.Errorf("state must be present or absent")
	}
	return fmt.Sprintf(`$names = %s; $state = %s; $parents = %s; $source = %s
$r = [ordered]@{ changed = $false; reboot_required = $false }
foreach ($n in $names) {
  $f = Get-WindowsOptionalFeature -Online -FeatureName $n
  if (-not $f) { throw "unknown optional feature $n" }
  $on = "$($f.State)" -eq 'Enabled'
  if (($state -eq 'present') -eq $on) { continue }
  $r.changed = $true
  if ($check) { continue }
  if ($state -eq 'present') {
    $o = @{ Online = $true; FeatureName = $n; NoRestart = $true }
    if ($parents) { $o.All = $true }
    if ($source) { $o.Source = $source; $o.LimitAccess = $true }
    $res = Enable-WindowsOptionalFeature @o
  } else {
    $res = Disable-WindowsOptionalFeature -Online -FeatureName $n -NoRestart
  }
  if ($res.RestartNeeded) { $r.reboot_required = $true }
}`, psList(args["name"]), psQuote(state), psBool(getBoolArg(args, "include_parent", false)),
		psQuote(getStringArg(args, "source", ""))), nil
}

// NewWinDiskFactsModule creates win_disk_facts
func NewWinDiskFactsModule() types.Module {
	return newWinJSONModule("win_disk_facts", "Return the disks of a Windows host as ansible_disks", winDiskFactsScript)
}

func winDiskFactsScript(_ context.Context, _ winrm.Runner, _ map[string]interface{}) (string, error) {
	return `$disks = @(Get-Disk | Sort-Object Number | ForEach-Object {
  $d = $_
  $parts = @(Get-Partition -DiskNumber $d.Number -ErrorAction SilentlyContinue)
  [ordered]@{
    number = $d.Number; size = [long]$d.Size; bus_type = "$($d.BusType)"; friendly_name = $d.FriendlyName
    partition_style = "$($d.PartitionStyle)"; partition_count = $parts.Count; operational_status = "$($d.OperationalStatus)"
    read_only = [bool]$d.IsReadOnly; bootable = [bool]$d.IsBoot; system_disk = [bool]$d.IsSystem
    guid = "$($d.Guid)"; unique_id = $d.UniqueId; serial_number = "$($d.SerialNumber)".Trim(); location = $d.Location
    partitions = @($parts | ForEach-Object { [ordered]@{ number = $_.PartitionNumber; size = [long]$_.Size; drive_letter = "$($_.DriveLetter)".Trim([char]0); type = "$($_.Type)"; offset = [long]$_.Offset } })
  }
})
$r = [ordered]@{ changed = $false; ansible_facts = @{ ansible_disks = $disks } }`, nil
}

// NewWinInitializeDiskModule creates win_initialize_disk
func NewWinInitializeDiskModule() types.Module {
	return newWinJSONModule("win_initialize_disk", "Initialize a raw disk with a GPT or MBR partition table", winInitializeDiskScript)
}

// anyArg is an argument as text, numbers included ("" when missing)
func anyArg(args map[string]interface{}, key string) string {
	if v, ok := args[key]; ok && v != nil {
		return strings.TrimSpace(fmt.Sprint(v))
	}
	return ""
}

// diskSelector finds one disk by disk_number, uniqueid or path
func diskSelector(args map[string]interface{}) (string, error) {
	switch {
	case anyArg(args, "disk_number") != "":
		n, err := strconv.Atoi(anyArg(args, "disk_number"))
		if err != nil {
			return "", fmt.Errorf("disk_number must be a number")
		}
		return fmt.Sprintf("Get-Disk -Number %d", n), nil
	case getStringArg(args, "uniqueid", "") != "":
		return "Get-Disk -UniqueId " + psQuote(getStringArg(args, "uniqueid", "")), nil
	case getStringArg(args, "path", "") != "":
		return "Get-Disk -Path " + psQuote(getStringArg(args, "path", "")), nil
	}
	return "", fmt.Errorf("give disk_number, uniqueid or path")
}

func winInitializeDiskScript(_ context.Context, _ winrm.Runner, args map[string]interface{}) (string, error) {
	sel, err := diskSelector(args)
	if err != nil {
		return "", err
	}
	style := strings.ToUpper(getStringArg(args, "style", "gpt"))
	if style != "GPT" && style != "MBR" {
		return "", fmt.Errorf("style must be gpt or mbr")
	}
	// like community.windows: an initialized disk is left alone unless
	// force, and force never wipes a disk that has partitions
	return fmt.Sprintf(`$disk = %s; $style = %s; $online = %s; $force = %s
$r = [ordered]@{ changed = $false }
if ($disk.IsOffline -and $online) { $r.changed = $true; if (-not $check) { Set-Disk -Number $disk.Number -IsOffline $false } }
if ($disk.IsReadOnly) { $r.changed = $true; if (-not $check) { Set-Disk -Number $disk.Number -IsReadOnly $false } }
if ("$($disk.PartitionStyle)" -eq 'RAW') {
  $r.changed = $true; if (-not $check) { Initialize-Disk -Number $disk.Number -PartitionStyle $style }
} elseif ("$($disk.PartitionStyle)" -ne $style) {
  if (-not $force) { throw "disk $($disk.Number) is already initialized as $($disk.PartitionStyle); force: true converts it" }
  if (@(Get-Partition -DiskNumber $disk.Number -ErrorAction SilentlyContinue).Count) { throw "disk $($disk.Number) has partitions: not converting it" }
  $r.changed = $true; if (-not $check) { Set-Disk -Number $disk.Number -PartitionStyle $style }
}`, sel, psQuote(style), psBool(getBoolArg(args, "online", true)), psBool(getBoolArg(args, "force", false))), nil
}

// NewWinPartitionModule creates win_partition
func NewWinPartitionModule() types.Module {
	return newWinJSONModule("win_partition", "Create, resize or remove a partition", winPartitionScript)
}

var sizeRe = regexp.MustCompile(`^\s*(\d+(?:\.\d+)?)\s*([KMGT]i?B|B)?\s*$`)

// partitionBytes reads partition_size: -1 (all free space), bytes, or a
// number with KB/MB/GB/TB (powers of 1000) or KiB/MiB/GiB/TiB
func partitionBytes(v interface{}) (int64, error) {
	s := strings.TrimSpace(fmt.Sprint(v))
	if s == "-1" {
		return -1, nil
	}
	m := sizeRe.FindStringSubmatch(s)
	if m == nil {
		return 0, fmt.Errorf("partition_size %q: -1, bytes, or a number with KB/MB/GB/TB or KiB/MiB/GiB/TiB", s)
	}
	n, _ := strconv.ParseFloat(m[1], 64)
	mult := map[string]float64{"": 1, "B": 1, "KB": 1e3, "MB": 1e6, "GB": 1e9, "TB": 1e12,
		"KiB": 1 << 10, "MiB": 1 << 20, "GiB": 1 << 30, "TiB": 1 << 40}[m[2]]
	return int64(n * mult), nil
}

func winPartitionScript(_ context.Context, _ winrm.Runner, args map[string]interface{}) (string, error) {
	state := getStringArg(args, "state", "present")
	if state != "present" && state != "absent" {
		return "", fmt.Errorf("state must be present or absent")
	}
	letter := strings.TrimSuffix(strings.ToUpper(getStringArg(args, "drive_letter", "")), ":")
	if letter != "" && (len(letter) != 1 || letter[0] < 'A' || letter[0] > 'Z') {
		return "", fmt.Errorf("drive_letter must be one letter")
	}
	number := anyArg(args, "partition_number")
	disk := anyArg(args, "disk_number")
	if letter == "" && (disk == "" || (number == "" && state == "absent")) {
		return "", fmt.Errorf("give drive_letter, or disk_number (and partition_number to remove one)")
	}
	size := int64(-1)
	if v, ok := args["partition_size"]; ok && v != nil {
		var err error
		if size, err = partitionBytes(v); err != nil {
			return "", err
		}
	}
	return fmt.Sprintf(`$letter = %s; $disk = %s; $number = %s; $state = %s; $size = [long]%d
$r = [ordered]@{ changed = $false }
$part = $null
if ($letter) { $part = Get-Partition -DriveLetter $letter -ErrorAction SilentlyContinue }
elseif ($number) { $part = Get-Partition -DiskNumber ([int]$disk) -PartitionNumber ([int]$number) -ErrorAction SilentlyContinue }
if ($state -eq 'absent') {
  if ($part) { $r.changed = $true; if (-not $check) { $part | Remove-Partition -Confirm:$false } }
} elseif (-not $part) {
  if (-not $disk) { throw "no partition $letter and no disk_number to create it on" }
  $r.changed = $true
  if (-not $check) {
    $o = @{ DiskNumber = [int]$disk }
    if ($size -lt 0) { $o.UseMaximumSize = $true } else { $o.Size = $size }
    if ($letter) { $o.DriveLetter = $letter }
    $part = New-Partition @o
  }
} elseif ($size -ge 0 -and [Math]::Abs($part.Size - $size) -gt 1MB) {
  $r.changed = $true
  if (-not $check) { $part | Resize-Partition -Size $size }
}
if ($part) { $r.partition_number = $part.PartitionNumber; $r.disk_number = $part.DiskNumber; $r.drive_letter = "$($part.DriveLetter)".Trim([char]0); $r.size = [long]$part.Size }`,
		psQuote(letter), psQuote(disk), psQuote(number), psQuote(state), size), nil
}

// NewWinFormatModule creates win_format
func NewWinFormatModule() types.Module {
	return newWinJSONModule("win_format", "Format a volume", winFormatScript)
}

func winFormatScript(_ context.Context, _ winrm.Runner, args map[string]interface{}) (string, error) {
	letter := strings.TrimSuffix(strings.ToUpper(getStringArg(args, "drive_letter", "")), ":")
	path := getStringArg(args, "path", "")
	label := getStringArg(args, "label", "")
	if letter == "" && path == "" && label == "" {
		return "", fmt.Errorf("give drive_letter, path or label")
	}
	fs := strings.ToUpper(getStringArg(args, "file_system", "ntfs"))
	switch fs {
	case "NTFS", "REFS", "EXFAT", "FAT32", "FAT":
	default:
		return "", fmt.Errorf("file_system must be ntfs, refs, exfat, fat32 or fat")
	}
	if fs == "REFS" {
		fs = "ReFS"
	} else if fs == "EXFAT" {
		fs = "exFAT"
	}
	unit := anyArg(args, "allocation_unit_size")
	if unit != "" {
		if _, err := strconv.Atoi(unit); err != nil {
			return "", fmt.Errorf("allocation_unit_size must be a number of bytes")
		}
	}
	// as community.windows: a volume that already has a file system is
	// formatted again only with force
	return fmt.Sprintf(`$letter = %s; $path = %s; $label = %s; $fs = %s; $newLabel = %s; $unit = %s; $force = %s; $full = %s
$vol = if ($letter) { Get-Volume -DriveLetter $letter -ErrorAction SilentlyContinue } elseif ($path) { Get-Volume -Path $path -ErrorAction SilentlyContinue } else { Get-Volume -FileSystemLabel $label -ErrorAction SilentlyContinue }
if (-not $vol) { throw "no volume $letter$path$label" }
$r = [ordered]@{ changed = $false }
$have = "$($vol.FileSystem)"
if ($have -and -not $force) {
  if ($have -ne $fs) { throw "volume has file system $have, not $fs; force: true reformats it" }
  if ($newLabel -and $vol.FileSystemLabel -ne $newLabel) { $r.changed = $true; if (-not $check) { $vol | Set-Volume -NewFileSystemLabel $newLabel } }
} else {
  $r.changed = $true
  if (-not $check) {
    $o = @{ FileSystem = $fs; Force = $true; Confirm = $false }
    if ($newLabel) { $o.NewFileSystemLabel = $newLabel }
    if ($unit) { $o.AllocationUnitSize = [int]$unit }
    if ($full) { $o.Full = $true }
    $vol | Format-Volume @o | Out-Null
  }
}`, psQuote(letter), psQuote(path), psQuote(label), psQuote(fs), psQuote(getStringArg(args, "new_label", "")),
		psQuote(unit), psBool(getBoolArg(args, "force", false)), psBool(getBoolArg(args, "full", false))), nil
}
