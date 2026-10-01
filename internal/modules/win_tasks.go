package modules

import (
	"context"
	"fmt"
	"strings"

	"github.com/onigirazu-cfg/onigirazu/internal/winrm"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// NewWinScheduledTaskModule creates win_scheduled_task
func NewWinScheduledTaskModule() types.Module {
	return newWinJSONModule("win_scheduled_task", "Manage Windows scheduled tasks", winScheduledTaskScript)
}

var taskTriggerTypes = map[string]bool{"daily": true, "weekly": true, "once": true, "time": true,
	"boot": true, "logon": true, "registration": true}

func winScheduledTaskScript(_ context.Context, _ winrm.Runner, args map[string]interface{}) (string, error) {
	name := getStringArg(args, "name", "")
	if name == "" {
		return "", fmt.Errorf("win_scheduled_task needs name")
	}
	state := getStringArg(args, "state", "present")
	if state != "present" && state != "absent" {
		return "", fmt.Errorf("state must be present or absent")
	}
	path := "\\" + strings.Trim(getStringArg(args, "path", "\\"), "\\") + "\\"
	path = strings.ReplaceAll(path, "\\\\", "\\")

	var actions []map[string]interface{}
	if list, ok := args["actions"].([]interface{}); ok {
		for _, a := range list {
			m, ok := a.(map[string]interface{})
			if !ok || getStringArg(m, "path", "") == "" {
				return "", fmt.Errorf("every action needs path")
			}
			actions = append(actions, map[string]interface{}{"path": getStringArg(m, "path", ""),
				"arguments": getStringArg(m, "arguments", ""), "working_directory": getStringArg(m, "working_directory", "")})
		}
	}
	var triggers []map[string]interface{}
	if list, ok := args["triggers"].([]interface{}); ok {
		for _, tr := range list {
			m, ok := tr.(map[string]interface{})
			typ := ""
			if ok {
				typ = strings.ToLower(getStringArg(m, "type", ""))
			}
			if !taskTriggerTypes[typ] {
				return "", fmt.Errorf("trigger type %q: daily, weekly, once, boot, logon or registration", typ)
			}
			if (typ == "daily" || typ == "weekly" || typ == "once" || typ == "time") && getStringArg(m, "start_boundary", "") == "" {
				return "", fmt.Errorf("a %s trigger needs start_boundary", typ)
			}
			days := ""
			switch d := m["days_of_week"].(type) {
			case []interface{}:
				parts := make([]string, len(d))
				for i, e := range d {
					parts[i] = fmt.Sprint(e)
				}
				days = strings.Join(parts, ",")
			case nil:
			default:
				days = fmt.Sprint(d)
			}
			if typ == "weekly" && days == "" {
				return "", fmt.Errorf("a weekly trigger needs days_of_week")
			}
			enabled := true
			if v, ok := m["enabled"]; ok && v != nil {
				enabled = getBoolArg(m, "enabled", true)
			}
			triggers = append(triggers, map[string]interface{}{"type": typ,
				"start_boundary": getStringArg(m, "start_boundary", ""), "days_of_week": days,
				"days_interval": anyArg(m, "days_interval"), "weeks_interval": anyArg(m, "weeks_interval"),
				"user": getStringArg(m, "user_id", ""), "enabled": enabled})
		}
	}
	if state == "present" && len(actions) == 0 {
		return "", fmt.Errorf("a present task needs actions")
	}
	runLevel := strings.ToLower(getStringArg(args, "run_level", "limited"))
	if runLevel != "limited" && runLevel != "highest" {
		return "", fmt.Errorf("run_level must be limited or highest")
	}
	logonType := getStringArg(args, "logon_type", "")
	switch logonType {
	case "", "password", "s4u", "interactive_token", "group", "service_account":
	default:
		return "", fmt.Errorf("logon_type must be password, s4u, interactive_token, group or service_account")
	}
	actionsJSON, err := psJSON(actions)
	if err != nil {
		return "", err
	}
	triggersJSON, err := psJSON(triggers)
	if err != nil {
		return "", err
	}
	enabled := getBoolArg(args, "enabled", true)
	return fmt.Sprintf(`$name = %[1]s; $path = %[2]s; $state = %[3]s; $description = %[4]s
$actions = @(%[5]s); $triggers = @(%[6]s)
$user = %[7]s; $password = %[8]s; $runLevel = %[9]s; $logonType = %[10]s; $enabled = %[11]s
$r = [ordered]@{ changed = $false; name = $name; path = $path }
$task = Get-ScheduledTask -TaskName $name -TaskPath $path -ErrorAction SilentlyContinue
if ($state -eq 'absent') {
  if ($task) { $r.changed = $true; if (-not $check) { Unregister-ScheduledTask -TaskName $name -TaskPath $path -Confirm:$false } }
} else {
  $days = @{ sunday = 1; monday = 2; tuesday = 4; wednesday = 8; thursday = 16; friday = 32; saturday = 64 }
  $newActions = @($actions | ForEach-Object {
    $o = @{ Execute = $_.path }
    if ($_.arguments) { $o.Argument = $_.arguments }
    if ($_.working_directory) { $o.WorkingDirectory = $_.working_directory }
    New-ScheduledTaskAction @o })
  $newTriggers = @($triggers | ForEach-Object {
    $t = $_
    switch ($t.type) {
      'daily' { $o = @{ Daily = $true; At = [datetime]$t.start_boundary }; if ($t.days_interval) { $o.DaysInterval = [int]$t.days_interval }; $x = New-ScheduledTaskTrigger @o }
      'weekly' {
        $o = @{ Weekly = $true; At = [datetime]$t.start_boundary; DaysOfWeek = @($t.days_of_week -split ',\s*' | ForEach-Object { (Get-Culture).TextInfo.ToTitleCase($_.ToLower()) }) }
        if ($t.weeks_interval) { $o.WeeksInterval = [int]$t.weeks_interval }
        $x = New-ScheduledTaskTrigger @o }
      { $_ -in 'once', 'time' } { $x = New-ScheduledTaskTrigger -Once -At ([datetime]$t.start_boundary) }
      'boot' { $x = New-ScheduledTaskTrigger -AtStartup }
      'logon' { $o = @{ AtLogOn = $true }; if ($t.user) { $o.User = $t.user }; $x = New-ScheduledTaskTrigger @o }
      'registration' { $class = Get-CimClass -ClassName MSFT_TaskRegistrationTrigger -Namespace Root/Microsoft/Windows/TaskScheduler; $x = New-CimInstance -CimClass $class -ClientOnly }
    }
    $x.Enabled = [bool]$t.enabled
    $x })
  $service = @('SYSTEM', 'NT AUTHORITY\SYSTEM', 'LOCAL SERVICE', 'NT AUTHORITY\LOCAL SERVICE', 'NETWORK SERVICE', 'NT AUTHORITY\NETWORK SERVICE') -contains "$user".ToUpper()
  if (-not $logonType) { $logonType = if ($password) { 'password' } elseif ($service) { 'service_account' } elseif ($user) { 'interactive_token' } else { '' } }
  $lt = @{ password = 'Password'; s4u = 'S4U'; interactive_token = 'Interactive'; group = 'Group'; service_account = 'ServiceAccount' }[$logonType]
  $rl = if ($runLevel -eq 'highest') { 'Highest' } else { 'Limited' }
  function Sid($n) { if (-not $n) { return '' }; try { (New-Object Security.Principal.NTAccount($n)).Translate([Security.Principal.SecurityIdentifier]).Value } catch { "$n".ToLower() } }
  function Shape($acts, $trigs, $desc, $uid, $level, $logon, $on) {
    [ordered]@{
      actions = @($acts | ForEach-Object { "$($_.Execute)|$($_.Arguments)|$($_.WorkingDirectory)" })
      triggers = @($trigs | ForEach-Object { "$($_.CimClass.CimClassName)|$($_.StartBoundary -replace '([+-]\d\d:\d\d|Z)$', '')|$($_.DaysOfWeek)|$($_.DaysInterval)|$($_.WeeksInterval)|$($_.UserId)|$($_.Enabled)" })
      description = "$desc"; user = Sid $uid; run_level = "$level"; logon = "$logon"; enabled = [bool]$on }
  }
  $want = Shape $newActions $newTriggers $description $user $rl $lt $enabled
  if ($task) {
    $have = Shape $task.Actions $task.Triggers $task.Description $task.Principal.UserId $task.Principal.RunLevel $task.Principal.LogonType ($task.State -ne 'Disabled')
    if (-not $user) { $want.user = $have.user; $want.logon = $have.logon }
    if (-not $description) { $want.description = $have.description }
    $diff = @($want.Keys | Where-Object { ($want[$_] -join '||') -ne ($have[$_] -join '||') })
    $r.differences = @($diff | ForEach-Object { "$($_): registered [$($have[$_] -join '; ')] wanted [$($want[$_] -join '; ')]" })
  } else { $diff = @('task') }
  if ($diff) {
    $r.changed = $true; $r.changed_properties = $diff
    if (-not $check) {
      $reg = @{ TaskName = $name; TaskPath = $path; Action = $newActions; Force = $true }
      if ($newTriggers) { $reg.Trigger = $newTriggers }
      if ($description) { $reg.Description = $description }
      $settings = New-ScheduledTaskSettingsSet
      $settings.Enabled = $enabled
      $reg.Settings = $settings
      if ($password) { $reg.User = $user; $reg.Password = $password; $reg.RunLevel = $rl }
      elseif ($user) { $reg.Principal = New-ScheduledTaskPrincipal -UserId $user -LogonType $lt -RunLevel $rl }
      else { $reg.RunLevel = $rl }
      Register-ScheduledTask @reg | Out-Null
    }
  }
}`, psQuote(name), psQuote(path), psQuote(state), psQuote(getStringArg(args, "description", "")),
		actionsJSON, triggersJSON, psQuote(getStringArg(args, "username", "")), psQuote(getStringArg(args, "password", "")),
		psQuote(runLevel), psQuote(logonType), psBool(enabled)), nil
}

// NewWinChocolateyModule creates win_chocolatey
func NewWinChocolateyModule() types.Module {
	return newWinJSONModule("win_chocolatey", "Install, upgrade or remove Chocolatey packages", winChocolateyScript)
}

func winChocolateyScript(_ context.Context, _ winrm.Runner, args map[string]interface{}) (string, error) {
	if args["name"] == nil {
		return "", fmt.Errorf("win_chocolatey needs name")
	}
	state := getStringArg(args, "state", "present")
	switch state {
	case "present", "latest", "absent", "downgrade", "upgrade", "reinstalled":
	default:
		return "", fmt.Errorf("state must be present, latest, absent, downgrade, upgrade or reinstalled")
	}
	if state == "upgrade" {
		state = "latest"
	}
	return fmt.Sprintf(`$names = %s; $state = %s; $version = %s; $source = %s; $extra = %s; $params = %s
$r = [ordered]@{ changed = $false; rc = 0; results = [ordered]@{} }
& {
$choco = Get-Command choco.exe -ErrorAction SilentlyContinue
if (-not $choco) {
  if ($state -eq 'absent') { return }
  $r.changed = $true
  if ($check) { $r.msg = 'would install Chocolatey'; return }
  [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor 3072
  $env:chocolateyUseWindowsCompression = 'true'
  Invoke-Expression ((New-Object Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1')) | Out-Null
  $choco = Get-Command "$env:ProgramData\chocolatey\bin\choco.exe"
}
$names = @($names | Where-Object { $_ -ne 'chocolatey' -or $state -ne 'present' })
function Installed {
  $v2 = [version]((& $choco.Source --version) -replace '[^0-9.].*$', '') -ge [version]'2.0'
  $list = if ($v2) { & $choco.Source list --limit-output } else { & $choco.Source list --local-only --limit-output }
  $h = @{}; foreach ($l in $list) { $p = "$l" -split '\|'; if ($p.Count -ge 2) { $h[$p[0].ToLower()] = $p[1] } }
  $h
}
$have = Installed
$common = @('-y', '--no-progress', '--limit-output')
if ($source) { $common += "--source=$source" }
if ($params) { $common += "--params=$params" }
if ($extra) { $common += ($extra -split '\s+') }
foreach ($n in $names) {
  $cur = $have[$n.ToLower()]
  $cmd = $null; $more = @()
  switch ($state) {
    'present' { if (-not $cur) { $cmd = 'install' } elseif ($version -and $cur -ne $version) { $cmd = 'install'; $more = @('--allow-downgrade', '--force') } }
    'downgrade' { if ($version -and $cur -ne $version) { $cmd = 'install'; $more = @('--allow-downgrade') } elseif (-not $cur) { $cmd = 'install' } }
    'latest' { $cmd = if ($cur) { 'upgrade' } else { 'install' } }
    'reinstalled' { $cmd = 'install'; $more = @('--force') }
    'absent' { if ($cur) { $cmd = 'uninstall' } }
  }
  if (-not $cmd) { continue }
  $opts = @($cmd, $n) + $common + $more
  if ($version -and $cmd -ne 'uninstall') { $opts += "--version=$version" }
  if ($check) { if ($cmd -ne 'upgrade') { $r.changed = $true }; continue }
  $out = & $choco.Source @opts 2>&1 | Out-String
  $rc = $LASTEXITCODE
  $r.results[$n] = [ordered]@{ command = $cmd; rc = $rc; stdout = $out.Trim() }
  if ($rc -notin 0, 1605, 1614, 1641, 3010) { $r.failed = $true; $r.rc = $rc; $r.msg = "choco $cmd $n failed ($rc): $($out.Trim())"; break }
  if ($rc -in 1641, 3010) { $r.reboot_required = $true }
  if ($cmd -eq 'upgrade') { if ((Installed)[$n.ToLower()] -ne $cur) { $r.changed = $true } } else { $r.changed = $true }
}
}`, psList(args["name"]), psQuote(state), psQuote(getStringArg(args, "version", "")),
		psQuote(getStringArg(args, "source", "")), psQuote(getStringArg(args, "install_args", "")),
		psQuote(getStringArg(args, "package_params", ""))), nil
}
