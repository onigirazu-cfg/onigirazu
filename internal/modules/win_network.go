package modules

import (
	"context"
	"fmt"
	"strings"

	"github.com/onigirazu-cfg/onigirazu/internal/winrm"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// psList is a list argument (a list, or a comma separated string) as a
// PowerShell array expression
func psList(v interface{}) string {
	var items []string
	switch x := v.(type) {
	case nil:
	case []interface{}:
		for _, e := range x {
			items = append(items, fmt.Sprint(e))
		}
	case []string:
		items = x
	default:
		for _, e := range strings.Split(fmt.Sprint(x), ",") {
			if e = strings.TrimSpace(e); e != "" {
				items = append(items, e)
			}
		}
	}
	quoted := make([]string, len(items))
	for i, e := range items {
		quoted[i] = psQuote(e)
	}
	return "@(" + strings.Join(quoted, ", ") + ")"
}

// NewWinFirewallRuleModule creates win_firewall_rule
func NewWinFirewallRuleModule() types.Module {
	return newWinJSONModule("win_firewall_rule", "Manage Windows Firewall rules", winFirewallRuleScript)
}

func winFirewallRuleScript(_ context.Context, _ winrm.Runner, args map[string]interface{}) (string, error) {
	name := getStringArg(args, "name", "")
	group := getStringArg(args, "group", "")
	if name == "" && group == "" {
		return "", fmt.Errorf("win_firewall_rule needs name or group")
	}
	state := getStringArg(args, "state", "present")
	if state != "present" && state != "absent" {
		return "", fmt.Errorf("state must be present or absent")
	}
	enabled := "$null"
	if v, ok := args["enabled"]; ok && v != nil {
		enabled = psBool(getBoolArg(args, "enabled", true))
	}
	// the other properties, $null when the task leaves them alone
	opt := func(key string, values ...string) (string, error) {
		v := strings.TrimSpace(getStringArg(args, key, ""))
		if v == "" {
			return "$null", nil
		}
		if len(values) > 0 {
			for _, ok := range values {
				if strings.EqualFold(v, ok) {
					return psQuote(ok), nil
				}
			}
			return "", fmt.Errorf("%s must be one of %s", key, strings.Join(values, ", "))
		}
		return psQuote(v), nil
	}
	list := func(key string) string {
		if v, ok := args[key]; ok && v != nil && fmt.Sprint(v) != "" {
			return psList(v)
		}
		return "$null"
	}
	action, err := opt("action", "Allow", "Block")
	if err != nil {
		return "", err
	}
	direction, err := opt("direction", "In", "Out")
	if err != nil {
		return "", err
	}
	protocol, _ := opt("protocol")
	program, _ := opt("program")
	service, _ := opt("service")
	description, _ := opt("description")
	return fmt.Sprintf(`$name = %s; $group = %s; $state = %s; $enabled = %s
$want = [ordered]@{ Action = %s; Direction = %s; Protocol = %s; LocalPort = %s; RemotePort = %s; LocalAddress = %s; RemoteAddress = %s; Program = %s; Service = %s; Profile = %s; Description = %s }
$r = [ordered]@{ changed = $false }
# Windows keeps 10.0.0.0/8 as 10.0.0.0/255.0.0.0 and a /32 as the bare address
function Cidr($s) {
  if ($s -notmatch '^(\d+\.\d+\.\d+\.\d+)/(\d{1,2})$') { return $s }
  $bits = [int]$Matches[2]
  if ($bits -eq 32) { return $Matches[1] }
  $mask = 0..3 | ForEach-Object { 256 - [Math]::Pow(2, 8 - [Math]::Min(8, [Math]::Max(0, $bits - 8 * $_))) }
  return "$($Matches[1])/$($mask -join '.')"
}
function Norm($v) { @($v | ForEach-Object { "$_" } | Where-Object { $_ -ne '' } | ForEach-Object { if ($_ -eq 'any') { 'Any' } else { Cidr $_ } } | Sort-Object) -join ',' }
if (-not $name) {
  # a group: turn every rule of it on or off
  $rules = @(Get-NetFirewallRule -DisplayGroup $group -ErrorAction SilentlyContinue)
  if (-not $rules) { $rules = @(Get-NetFirewallRule -Group $group -ErrorAction SilentlyContinue) }
  if (-not $rules) { throw "no firewall rules in group $group" }
  if ($null -ne $enabled) {
    $flag = if ($enabled) { 'True' } else { 'False' }
    $todo = @($rules | Where-Object { "$($_.Enabled)" -ne $flag })
    if ($todo) { $r.changed = $true; if (-not $check) { $todo | Set-NetFirewallRule -Enabled $flag } }
  }
  $r.rules = $rules.Count
} else {
  $rule = Get-NetFirewallRule -DisplayName $name -ErrorAction SilentlyContinue | Select-Object -First 1
  if ($state -eq 'absent') {
    if ($rule) { $r.changed = $true; if (-not $check) { Get-NetFirewallRule -DisplayName $name | Remove-NetFirewallRule } }
  } else {
    $props = @{}
    foreach ($k in $want.Keys) { if ($null -ne $want[$k]) { $props[$k] = $want[$k] } }
    if ($props.ContainsKey('Profile')) { $props['Profile'] = (@($props['Profile']) | ForEach-Object { (Get-Culture).TextInfo.ToTitleCase("$_".ToLower()) }) -join ',' }
    if ($null -ne $enabled) { $props['Enabled'] = if ($enabled) { 'True' } else { 'False' } }
    if (-not $rule) {
      $r.changed = $true
      if (-not $check) {
        if ($group) { $props['Group'] = $group }
        if (-not $props.ContainsKey('Direction')) { $props['Direction'] = 'Inbound' } elseif ($props['Direction'] -eq 'In') { $props['Direction'] = 'Inbound' } else { $props['Direction'] = 'Outbound' }
        New-NetFirewallRule -DisplayName $name @props | Out-Null
      }
    } else {
      $port = $rule | Get-NetFirewallPortFilter; $addr = $rule | Get-NetFirewallAddressFilter
      $app = $rule | Get-NetFirewallApplicationFilter; $svc = $rule | Get-NetFirewallServiceFilter
      $have = @{ Action = "$($rule.Action)"; Direction = "$($rule.Direction)".Replace('bound', ''); Protocol = "$($port.Protocol)"
        LocalPort = $port.LocalPort; RemotePort = $port.RemotePort; LocalAddress = $addr.LocalAddress; RemoteAddress = $addr.RemoteAddress
        Program = "$($app.Program)"; Service = "$($svc.Service)"; Profile = "$($rule.Profile)"; Description = "$($rule.Description)"; Enabled = "$($rule.Enabled)" }
      $diff = @{}
      foreach ($k in $props.Keys) {
        $a = Norm $props[$k]; $b = Norm $have[$k]
        if ($k -eq 'Profile') { $a = Norm ("$($props[$k])" -split ',\s*'); $b = Norm ("$($have[$k])" -split ',\s*') }
        if ($a -ne $b) { $diff[$k] = $props[$k] }
      }
      if ($diff.Count) {
        $r.changed = $true
        $r.changed_properties = @($diff.Keys | Sort-Object)
        if ($diff.ContainsKey('Direction')) { $diff['Direction'] = if ($diff['Direction'] -eq 'In') { 'Inbound' } else { 'Outbound' } }
        if (-not $check) { Get-NetFirewallRule -DisplayName $name | Set-NetFirewallRule @diff }
      }
    }
  }
}`, psQuote(name), psQuote(group), psQuote(state), enabled, action, direction, protocol,
		list("localport"), list("remoteport"), list("localip"), list("remoteip"), program, service,
		list("profiles"), description), nil
}

// NewWinFirewallModule creates win_firewall
func NewWinFirewallModule() types.Module {
	return newWinJSONModule("win_firewall", "Turn Windows Firewall profiles on or off", winFirewallScript)
}

func winFirewallScript(_ context.Context, _ winrm.Runner, args map[string]interface{}) (string, error) {
	state := getStringArg(args, "state", "")
	if state != "" && state != "enabled" && state != "disabled" {
		return "", fmt.Errorf("state must be enabled or disabled")
	}
	profiles := args["profiles"]
	if profiles == nil {
		profiles = []interface{}{"Domain", "Private", "Public"}
	}
	in, _ := args["inbound_action"].(string)
	out, _ := args["outbound_action"].(string)
	for _, a := range []string{in, out} {
		if a != "" && a != "allow" && a != "block" && a != "not_configured" {
			return "", fmt.Errorf("inbound_action/outbound_action must be allow, block or not_configured")
		}
	}
	return fmt.Sprintf(`$state = %s; $in = %s; $out = %s
$map = @{ allow = 'Allow'; block = 'Block'; not_configured = 'NotConfigured' }
$r = [ordered]@{ changed = $false; profiles = @() }
foreach ($p in %s) {
  $prof = Get-NetFirewallProfile -Name $p
  $set = @{}
  if ($state) { $flag = if ($state -eq 'enabled') { 'True' } else { 'False' }; if ("$($prof.Enabled)" -ne $flag) { $set['Enabled'] = $flag } }
  if ($in -and "$($prof.DefaultInboundAction)" -ne $map[$in]) { $set['DefaultInboundAction'] = $map[$in] }
  if ($out -and "$($prof.DefaultOutboundAction)" -ne $map[$out]) { $set['DefaultOutboundAction'] = $map[$out] }
  if ($set.Count) { $r.changed = $true; if (-not $check) { Set-NetFirewallProfile -Name $p @set } }
  $r.profiles += $prof.Name
}`, psQuote(state), psQuote(in), psQuote(out), psList(profiles)), nil
}

// NewWinGroupMembershipModule creates win_group_membership
func NewWinGroupMembershipModule() types.Module {
	return newWinJSONModule("win_group_membership", "Manage the members of a local Windows group", winGroupMembershipScript)
}

func winGroupMembershipScript(_ context.Context, _ winrm.Runner, args map[string]interface{}) (string, error) {
	name := getStringArg(args, "name", "")
	if name == "" {
		return "", fmt.Errorf("win_group_membership needs name")
	}
	state := getStringArg(args, "state", "present")
	if state != "present" && state != "absent" && state != "pure" {
		return "", fmt.Errorf("state must be present, absent or pure")
	}
	if args["members"] == nil {
		return "", fmt.Errorf("win_group_membership needs members")
	}
	// members compare by SID, so DOMAIN\user, user and user@domain match
	return fmt.Sprintf(`$name = %s; $state = %s; $members = %s
function Sid($n) {
  try { (New-Object Security.Principal.NTAccount($n)).Translate([Security.Principal.SecurityIdentifier]).Value }
  catch { if ($n -match '^S-1-') { $n } else { throw "cannot find $n" } }
}
$group = [ADSI]"WinNT://$env:COMPUTERNAME/$name,group"
if (-not $group.Path) { throw "group $name does not exist" }
$current = @{}
foreach ($m in @($group.Invoke('Members'))) {
  $sid = (New-Object Security.Principal.SecurityIdentifier($m.GetType().InvokeMember('objectSid', 'GetProperty', $null, $m, $null), 0)).Value
  $current[$sid] = $m.GetType().InvokeMember('AdsPath', 'GetProperty', $null, $m, $null)
}
$wanted = @{}
foreach ($n in $members) { $wanted[(Sid $n)] = $n }
$r = [ordered]@{ changed = $false; name = $name; added = @(); removed = @() }
if ($state -in 'present', 'pure') {
  foreach ($sid in $wanted.Keys) { if (-not $current.ContainsKey($sid)) { $r.added += $wanted[$sid]; if (-not $check) { $group.Add("WinNT://$sid") } } }
}
if ($state -eq 'absent') {
  foreach ($sid in $wanted.Keys) { if ($current.ContainsKey($sid)) { $r.removed += $wanted[$sid]; if (-not $check) { $group.Remove("WinNT://$sid") } } }
}
if ($state -eq 'pure') {
  foreach ($sid in $current.Keys) { if (-not $wanted.ContainsKey($sid)) { $r.removed += $current[$sid]; if (-not $check) { $group.Remove("WinNT://$sid") } } }
}
$r.changed = ($r.added.Count + $r.removed.Count) -gt 0`, psQuote(name), psQuote(state), psList(args["members"])), nil
}

// NewWinFeatureModule creates win_feature
func NewWinFeatureModule() types.Module {
	return newWinJSONModule("win_feature", "Install or remove Windows Server roles and features", winFeatureScript)
}

func winFeatureScript(_ context.Context, _ winrm.Runner, args map[string]interface{}) (string, error) {
	if args["name"] == nil {
		return "", fmt.Errorf("win_feature needs name")
	}
	state := getStringArg(args, "state", "present")
	if state != "present" && state != "absent" {
		return "", fmt.Errorf("state must be present or absent")
	}
	source, _ := args["source"].(string)
	return fmt.Sprintf(`$names = %s; $state = %s; $sub = %s; $tools = %s; $source = %s
$features = @(Get-WindowsFeature -Name $names)
$missing = @($names | Where-Object { $n = $_; -not ($features | Where-Object Name -eq $n) })
if ($missing) { throw "unknown features: $($missing -join ', ')" }
$todo = if ($state -eq 'present') { @($features | Where-Object { -not $_.Installed }) } else { @($features | Where-Object { $_.Installed }) }
$r = [ordered]@{ changed = [bool]$todo; reboot_required = $false; success = $true; exitcode = 'NoChangeNeeded'; feature_result = @() }
if ($todo -and -not $check) {
  $opts = @{ Name = @($todo.Name) }
  if ($state -eq 'present') {
    if ($sub) { $opts['IncludeAllSubFeature'] = $true }
    if ($tools) { $opts['IncludeManagementTools'] = $true }
    if ($source) { $opts['Source'] = $source }
    $res = Install-WindowsFeature @opts
  } else {
    if ($tools) { $opts['IncludeManagementTools'] = $true }
    $res = Uninstall-WindowsFeature @opts
  }
  $r.success = [bool]$res.Success
  $r.exitcode = "$($res.ExitCode)"
  $r.reboot_required = "$($res.RestartNeeded)" -eq 'Yes'
  $r.feature_result = @($res.FeatureResult | ForEach-Object { [ordered]@{ id = $_.Id; display_name = $_.DisplayName; message = "$($_.Message)"; reboot_required = [bool]$_.RestartNeeded; skip_reason = "$($_.SkipReason)"; success = [bool]$_.Success } })
  if (-not $res.Success) { $r.failed = $true; $r.msg = "win_feature failed: $($res.ExitCode)" }
}`, psList(args["name"]), psQuote(state), psBool(getBoolArg(args, "include_sub_features", false)),
		psBool(getBoolArg(args, "include_management_tools", false)), psQuote(source)), nil
}
