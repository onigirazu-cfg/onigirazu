package modules

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/winrm"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// winResultMarker brackets the JSON a Windows module prints, so profile
// output or stray writes around it do not matter
const winResultMarker = "@@ONIGIRAZU-RESULT@@"

// runWinJSON runs a PowerShell script that prints its result as JSON
// between winResultMarker lines and decodes it
func runWinJSON(ctx context.Context, c *winrm.Client, script string) (map[string]interface{}, winrm.Result, error) {
	res, err := c.RunPS(ctx, script)
	if err != nil {
		return nil, res, err
	}
	parts := strings.Split(res.Stdout, winResultMarker)
	if len(parts) < 3 {
		msg := strings.TrimSpace(res.Stderr)
		if msg == "" {
			msg = strings.TrimSpace(res.Stdout)
		}
		return nil, res, fmt.Errorf("no result from the host (exit code %d): %s", res.ExitCode, msg)
	}
	out := map[string]interface{}{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(parts[1])), &out); err != nil {
		return nil, res, fmt.Errorf("reading the result: %w", err)
	}
	return out, res, nil
}

// winPrintResult is the PowerShell that prints $r as the module result
func winPrintResult(depth int) string {
	return fmt.Sprintf("[Console]::Out.WriteLine('%[1]s')\n[Console]::Out.WriteLine(($r | ConvertTo-Json -Depth %[2]d -Compress))\n[Console]::Out.WriteLine('%[1]s')\n",
		winResultMarker, depth)
}

// psJSON is v as a PowerShell expression: JSON parsed on the host
func psJSON(v interface{}) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return "(ConvertFrom-Json " + psQuote(string(data)) + ")", nil
}

// WinPowerShellModule runs a PowerShell script with the $Ansible variable
// of ansible.windows.win_powershell and returns its streams
type WinPowerShellModule struct{ *BaseModule }

// NewWinPowerShellModule creates win_powershell
func NewWinPowerShellModule() *WinPowerShellModule {
	return &WinPowerShellModule{NewBaseModule("win_powershell")}
}

func (m *WinPowerShellModule) GetDescription() string {
	return "Run a PowerShell script on a Windows host and return its output, errors and $Ansible.Result"
}

func (m *WinPowerShellModule) Validate(args map[string]interface{}) error {
	if getStringArg(args, "script", "") == "" {
		return fmt.Errorf("win_powershell needs script")
	}
	switch getStringArg(args, "error_action", "continue") {
	case "continue", "stop", "silently_continue":
	default:
		return fmt.Errorf("error_action must be continue, stop or silently_continue")
	}
	return nil
}

// winPowerShellWrapper runs the script in its own PowerShell instance, so
// every stream is collected, with $Ansible as in ansible.windows
const winPowerShellWrapper = `$ErrorActionPreference = 'Stop'
$Ansible = [pscustomobject]@{ Changed = $true; Failed = $false; Result = $null; CheckMode = %[5]s; Diff = @{}; Tmpdir = $env:TEMP; Verbosity = 0 }
$params = @{}
$raw = %[2]s
if ($raw) { foreach ($p in $raw.PSObject.Properties) { $params[$p.Name] = $p.Value } }
$ps = [PowerShell]::Create()
[void]$ps.Runspace.SessionStateProxy.SetVariable('Ansible', $Ansible)
[void]$ps.Runspace.SessionStateProxy.SetVariable('ErrorActionPreference', %[3]s)
[void]$ps.AddScript(%[1]s)
if ($params.Count) { [void]$ps.AddParameters($params) }
$failed = $false
$output = @()
try { $output = @($ps.Invoke()) } catch { $failed = $true; $terminating = $_.Exception.InnerException; if (-not $terminating) { $terminating = $_.Exception } }
function ErrorInfo($e) {
  [ordered]@{
    output = ($e | Out-String).Trim()
    exception = [ordered]@{ message = $e.Exception.Message; type = $e.Exception.GetType().FullName }
    fully_qualified_error_id = $e.FullyQualifiedErrorId
    category_info = [ordered]@{ category = "$($e.CategoryInfo.Category)"; reason = $e.CategoryInfo.Reason }
    script_stack_trace = $e.ScriptStackTrace
  }
}
$errors = @($ps.Streams.Error | ForEach-Object { ErrorInfo $_ })
if ($terminating) {
  $record = $terminating.ErrorRecord
  if ($record) { $errors += ErrorInfo $record } else { $errors += [ordered]@{ output = $terminating.Message; exception = [ordered]@{ message = $terminating.Message; type = $terminating.GetType().FullName } } }
}
$hostOut = @($ps.Streams.Information | Where-Object { $_.Tags -contains 'PSHOST' } | ForEach-Object { "$($_.MessageData)" }) -join [Environment]::NewLine
$info = @($ps.Streams.Information | Where-Object { $_.Tags -notcontains 'PSHOST' } | ForEach-Object { [ordered]@{ message_data = "$($_.MessageData)"; source = $_.Source; tags = @($_.Tags) } })
$r = [ordered]@{
  changed = [bool]$Ansible.Changed
  failed = [bool]($Ansible.Failed -or $failed)
  result = $Ansible.Result
  output = @($output | ForEach-Object { if ($_ -is [string] -or $_ -is [ValueType]) { $_ } else { $_.PSObject.BaseObject } })
  error = $errors
  warning = @($ps.Streams.Warning | ForEach-Object { $_.Message })
  verbose = @($ps.Streams.Verbose | ForEach-Object { $_.Message })
  debug = @($ps.Streams.Debug | ForEach-Object { $_.Message })
  information = $info
  host_out = $hostOut
  host_err = ''
}
%[4]s`

func (m *WinPowerShellModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	start := time.Now()
	res := types.TaskResult{TaskName: taskName(args), Host: host.Name, Module: m.name, Timestamp: start,
		Output: map[string]interface{}{}}
	fail := func(err error) (types.TaskResult, error) {
		res.Failed, res.Error = true, err.Error()
		res.Duration = time.Since(start)
		return res, nil
	}
	if err := m.Validate(args); err != nil {
		return fail(err)
	}
	c, err := winClient(host, args)
	if err != nil {
		return fail(err)
	}
	if skip, msg, err := winSkip(ctx, c, args); err != nil {
		return fail(err)
	} else if skip {
		res.Success = true
		res.Output["msg"] = msg
		res.Duration = time.Since(start)
		return res, nil
	}
	script := getStringArg(args, "script", "")
	check := inCheckMode(args)
	// as in ansible.windows: a script runs in check mode only when it says
	// it supports it
	if check && !strings.Contains(script, "SupportsShouldProcess") {
		res.Success, res.Skipped = true, true
		res.Output["msg"] = "skipped in check mode: the script does not declare SupportsShouldProcess"
		return res, nil
	}
	params, err := psJSON(args["parameters"])
	if err != nil {
		return fail(fmt.Errorf("parameters: %w", err))
	}
	depth := 2
	if d, ok := toInt(args["depth"]); ok && d > 0 {
		depth = d
	}
	action := map[string]string{"continue": "Continue", "stop": "Stop", "silently_continue": "SilentlyContinue"}[getStringArg(args, "error_action", "continue")]
	body := script
	if chdir := getStringArg(args, "chdir", ""); chdir != "" {
		body = "Set-Location -LiteralPath " + psQuote(chdir) + "\n" + body
	}
	checkLit := "$false"
	if check {
		checkLit = "$true"
	}
	wrapper := winEnvPrefix(host) + fmt.Sprintf(winPowerShellWrapper, psQuote(body), params, psQuote(action),
		winPrintResult(depth+3), checkLit)
	out, _, err := runWinJSON(ctx, c, wrapper)
	if err != nil {
		return fail(err)
	}
	for k, v := range out {
		if k != "changed" && k != "failed" {
			res.Output[k] = v
		}
	}
	res.Changed, _ = out["changed"].(bool)
	if failed, _ := out["failed"].(bool); failed {
		res.Failed = true
		res.Error = "the script failed"
		if errs, ok := out["error"].([]interface{}); ok && len(errs) > 0 {
			if e, ok := errs[len(errs)-1].(map[string]interface{}); ok {
				res.Error = fmt.Sprint(e["output"])
			}
		}
	} else {
		res.Success = true
	}
	res.Duration = time.Since(start)
	return res, nil
}
