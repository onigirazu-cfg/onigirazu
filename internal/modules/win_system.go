package modules

import (
	"context"
	"crypto/sha1" // #nosec G505 -- Ansible's checksum format, not a security boundary
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/winrm"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// winJSONModule is a Windows module whose work is one PowerShell script
// that prints its result with winPrintResult
type winJSONModule struct {
	*BaseModule
	description string
	// script builds the PowerShell, or fails on bad arguments; c is for
	// modules that upload first
	script func(ctx context.Context, c winrm.Runner, args map[string]interface{}) (string, error)
}

func (m *winJSONModule) GetDescription() string { return m.description }

func (m *winJSONModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	start := time.Now()
	res := types.TaskResult{TaskName: taskName(args), Host: host.Name, Module: m.name, Timestamp: start,
		Output: map[string]interface{}{}}
	fail := func(err error) (types.TaskResult, error) {
		res.Failed, res.Error = true, err.Error()
		res.Duration = time.Since(start)
		return res, nil
	}
	c, err := winClient(host, args)
	if err != nil {
		return fail(err)
	}
	script, err := m.script(ctx, c, args)
	if err != nil {
		return fail(err)
	}
	check := "$false"
	if inCheckMode(args) {
		check = "$true"
	}
	out, _, err := runWinJSON(ctx, c, "$ErrorActionPreference = 'Stop'\n$check = "+check+"\n"+script+"\n"+winPrintResult(4))
	if err != nil {
		return fail(err)
	}
	for k, v := range out {
		if k != "changed" && k != "failed" && k != "msg" {
			res.Output[k] = v
		}
	}
	res.Changed, _ = out["changed"].(bool)
	if failed, _ := out["failed"].(bool); failed {
		res.Failed, res.Error = true, fmt.Sprint(out["msg"])
	} else {
		res.Success = true
		if msg, ok := out["msg"]; ok {
			res.Output["msg"] = msg
		}
	}
	res.Duration = time.Since(start)
	return res, nil
}

func newWinJSONModule(name, description string, script func(context.Context, winrm.Runner, map[string]interface{}) (string, error)) *winJSONModule {
	return &winJSONModule{BaseModule: NewBaseModule(name), description: description, script: script}
}

// NewWinFileModule creates win_file
func NewWinFileModule() types.Module {
	return newWinJSONModule("win_file", "Create or remove files and directories on a Windows host", winFileScript)
}

func winFileScript(_ context.Context, _ winrm.Runner, args map[string]interface{}) (string, error) {
	path := getStringArg(args, "path", getStringArg(args, "dest", ""))
	if path == "" {
		return "", fmt.Errorf("win_file needs path")
	}
	state := getStringArg(args, "state", "")
	switch state {
	case "", "file", "directory", "absent", "touch":
	default:
		return "", fmt.Errorf("state must be file, directory, absent or touch")
	}
	return fmt.Sprintf(`$path = %s; $state = %s
$item = Get-Item -LiteralPath $path -Force -ErrorAction SilentlyContinue
$r = [ordered]@{ changed = $false; path = $path }
if ($state -eq '') { $state = if ($item -and $item.PSIsContainer) { 'directory' } else { 'file' } }
switch ($state) {
  'absent' { if ($item) { $r.changed = $true; if (-not $check) { Remove-Item -LiteralPath $path -Recurse -Force } } }
  'directory' {
    if ($item -and -not $item.PSIsContainer) { throw "path $path exists and is a file" }
    if (-not $item) { $r.changed = $true; if (-not $check) { New-Item -Path $path -ItemType Directory -Force | Out-Null } }
  }
  'touch' {
    $r.changed = $true
    if (-not $check) { if ($item) { $item.LastWriteTime = Get-Date } else { New-Item -Path $path -ItemType File -Force | Out-Null } }
  }
  'file' {
    if (-not $item) { $r.failed = $true; $r.msg = "path $path does not exist; use state touch or win_copy" }
    elseif ($item.PSIsContainer) { $r.failed = $true; $r.msg = "path $path is a directory" }
  }
}
$r.state = $state`, psQuote(path), psQuote(state)), nil
}

// NewWinCopyModule creates win_copy
func NewWinCopyModule() types.Module {
	return newWinJSONModule("win_copy", "Copy a file or content to a Windows host", winCopyScript)
}

func winCopyScript(ctx context.Context, c winrm.Runner, args map[string]interface{}) (string, error) {
	dest := getStringArg(args, "dest", "")
	if dest == "" {
		return "", fmt.Errorf("win_copy needs dest")
	}
	force := getBoolArg(args, "force", true)
	remoteSrc := getBoolArg(args, "remote_src", false)
	src := getStringArg(args, "src", "")
	if remoteSrc {
		if src == "" {
			return "", fmt.Errorf("remote_src needs src")
		}
		return fmt.Sprintf(`$src = %s; $dest = %s; $force = %s
if (-not (Test-Path -LiteralPath $src -PathType Leaf)) { throw "src $src is not a file on the host" }
if (Test-Path -LiteralPath $dest -PathType Container) { $dest = Join-Path $dest (Split-Path $src -Leaf) }
$want = (Get-FileHash -LiteralPath $src -Algorithm SHA1).Hash.ToLower()
$have = if (Test-Path -LiteralPath $dest -PathType Leaf) { (Get-FileHash -LiteralPath $dest -Algorithm SHA1).Hash.ToLower() } else { '' }
$r = [ordered]@{ changed = $false; dest = $dest; src = $src; checksum = $want }
if ($have -ne $want -and ($force -or $have -eq '')) {
  $r.changed = $true
  if (-not $check) { New-Item -Path (Split-Path $dest) -ItemType Directory -Force | Out-Null; Copy-Item -LiteralPath $src -Destination $dest -Force }
}`, psQuote(src), psQuote(dest), psBool(force)), nil
	}

	var data []byte
	content, hasContent := args["content"]
	switch {
	case hasContent && src != "":
		return "", fmt.Errorf("win_copy takes src or content, not both")
	case hasContent:
		data = []byte(fmt.Sprint(content))
	case src != "":
		info, err := os.Stat(src)
		if err != nil {
			return "", fmt.Errorf("src: %w", err)
		}
		if info.IsDir() {
			return "", fmt.Errorf("src %s is a directory: copying directories is not supported yet", src)
		}
		data, err = os.ReadFile(src) // #nosec G304 -- the file the task names
		if err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("win_copy needs src or content")
	}
	sum := sha1.Sum(data) // #nosec G401 -- Ansible's checksum format
	checksum := hex.EncodeToString(sum[:])
	name := ""
	if src != "" {
		name = src[strings.LastIndexAny(src, `/\`)+1:]
	}
	// compare before uploading: unchanged files cost one round trip
	probe := fmt.Sprintf(`$dest = %s
if ((Test-Path -LiteralPath $dest -PathType Container) -and %s) { $dest = Join-Path $dest %s }
if (Test-Path -LiteralPath $dest -PathType Leaf) { (Get-FileHash -LiteralPath $dest -Algorithm SHA1).Hash.ToLower() + '|' + $dest } else { '|' + $dest }`,
		psQuote(dest), psBool(name != ""), psQuote(name))
	res, err := c.RunPS(ctx, probe)
	if err != nil {
		return "", err
	}
	have, target, _ := strings.Cut(strings.TrimSpace(res.Stdout), "|")
	base := fmt.Sprintf(`$r = [ordered]@{ changed = $false; dest = %s; checksum = %s; size = %d }`, psQuote(target), psQuote(checksum), len(data))
	if have == checksum || (have != "" && !force) || inCheckMode(args) {
		if have != checksum && (have == "" || force) {
			base += "\n$r.changed = $true"
		}
		return base, nil
	}
	tmp, err := c.Upload(ctx, data)
	if err != nil {
		return "", err
	}
	return base + fmt.Sprintf(`
$r.changed = $true
New-Item -Path (Split-Path %[1]s) -ItemType Directory -Force | Out-Null
Move-Item -LiteralPath %[2]s -Destination %[1]s -Force`, psQuote(target), psQuote(tmp)), nil
}

func psBool(b bool) string {
	if b {
		return "$true"
	}
	return "$false"
}

// NewWinServiceModule creates win_service
func NewWinServiceModule() types.Module {
	return newWinJSONModule("win_service", "Manage Windows services", winServiceScript)
}

func winServiceScript(_ context.Context, _ winrm.Runner, args map[string]interface{}) (string, error) {
	name := getStringArg(args, "name", "")
	if name == "" {
		return "", fmt.Errorf("win_service needs name")
	}
	state := getStringArg(args, "state", "")
	switch state {
	case "", "started", "stopped", "restarted", "paused":
	default:
		return "", fmt.Errorf("state must be started, stopped, restarted or paused")
	}
	mode := getStringArg(args, "start_mode", "")
	modes := map[string]string{"": "", "auto": "Automatic", "delayed": "AutomaticDelayed", "manual": "Manual", "disabled": "Disabled"}
	psMode, ok := modes[mode]
	if !ok {
		return "", fmt.Errorf("start_mode must be auto, delayed, manual or disabled")
	}
	return fmt.Sprintf(`$name = %s; $state = %s; $mode = %s
$svc = Get-Service -Name $name -ErrorAction SilentlyContinue
$r = [ordered]@{ changed = $false; name = $name; exists = [bool]$svc }
if (-not $svc) { if ($state -or $mode) { $r.failed = $true; $r.msg = "service $name does not exist" } }
else {
  $cim = Get-CimInstance Win32_Service -Filter "Name='$($svc.Name)'"
  $cur = switch ($cim.StartMode) { 'Auto' { if ($cim.DelayedAutoStart) { 'AutomaticDelayed' } else { 'Automatic' } } default { "$($cim.StartMode)" } }
  if ($mode -and $cur -ne $mode) {
    $r.changed = $true
    if (-not $check) {
      if ($mode -eq 'AutomaticDelayed') { & sc.exe config $svc.Name start= delayed-auto | Out-Null } else { Set-Service -Name $svc.Name -StartupType $mode }
    }
  }
  $running = $svc.Status -eq 'Running'
  switch ($state) {
    'started' { if (-not $running) { $r.changed = $true; if (-not $check) { Start-Service -Name $svc.Name } } }
    'stopped' { if ($svc.Status -ne 'Stopped') { $r.changed = $true; if (-not $check) { Stop-Service -Name $svc.Name -Force } } }
    'restarted' { $r.changed = $true; if (-not $check) { Restart-Service -Name $svc.Name -Force } }
    'paused' { if ($svc.Status -ne 'Paused') { $r.changed = $true; if (-not $check) { Suspend-Service -Name $svc.Name } } }
  }
  $svc = Get-Service -Name $svc.Name
  $r.name = $svc.Name; $r.display_name = $svc.DisplayName
  $r.state = "$($svc.Status)".ToLower(); $r.start_mode = if ($mode -and -not $check) { $mode } else { $cur }
}`, psQuote(name), psQuote(state), psQuote(psMode)), nil
}

// NewWinTimezoneModule creates win_timezone
func NewWinTimezoneModule() types.Module {
	return newWinJSONModule("win_timezone", "Set the time zone of a Windows host", winTimezoneScript)
}

func winTimezoneScript(_ context.Context, _ winrm.Runner, args map[string]interface{}) (string, error) {
	tz := getStringArg(args, "timezone", "")
	if tz == "" {
		return "", fmt.Errorf("win_timezone needs timezone")
	}
	return fmt.Sprintf(`$want = %s
$cur = (Get-TimeZone).Id
if (-not (Get-TimeZone -ListAvailable | Where-Object Id -eq $want)) { throw "unknown time zone $want (Get-TimeZone -ListAvailable)" }
$r = [ordered]@{ changed = $cur -ne $want; previous_timezone = $cur; timezone = $want }
if ($r.changed -and -not $check) { Set-TimeZone -Id $want }`, psQuote(tz)), nil
}
