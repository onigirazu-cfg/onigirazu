package modules

import (
	"context"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// WinRegeditModule manages registry keys and values
type WinRegeditModule struct{ *BaseModule }

// NewWinRegeditModule creates win_regedit
func NewWinRegeditModule() *WinRegeditModule { return &WinRegeditModule{NewBaseModule("win_regedit")} }

func (m *WinRegeditModule) GetDescription() string {
	return "Add, change or remove registry keys and values"
}

// registry value kinds of win_regedit's type
var regKinds = map[string]string{
	"none": "None", "binary": "Binary", "dword": "DWord", "expandstring": "ExpandString",
	"multistring": "MultiString", "string": "String", "qword": "QWord",
}

// regProviderPath turns HKLM:\x, HKEY_LOCAL_MACHINE\x and friends into a
// Registry:: provider path
func regProviderPath(p string) (string, error) {
	p = strings.ReplaceAll(strings.TrimSpace(p), "/", `\`)
	roots := map[string]string{
		"HKLM": "HKEY_LOCAL_MACHINE", "HKCU": "HKEY_CURRENT_USER", "HKCR": "HKEY_CLASSES_ROOT",
		"HKU": "HKEY_USERS", "HKCC": "HKEY_CURRENT_CONFIG",
	}
	root, rest, _ := strings.Cut(p, `\`)
	root = strings.TrimSuffix(strings.ToUpper(root), ":")
	if long, ok := roots[root]; ok {
		root = long
	}
	switch root {
	case "HKEY_LOCAL_MACHINE", "HKEY_CURRENT_USER", "HKEY_CLASSES_ROOT", "HKEY_USERS", "HKEY_CURRENT_CONFIG":
	default:
		return "", fmt.Errorf("path %q does not start with a registry hive (HKLM:, HKCU:, HKCR:, HKU:, HKCC:)", p)
	}
	return `Registry::` + root + `\` + strings.Trim(rest, `\`), nil
}

// regData turns data into what PowerShell gets for kind: integers as
// unsigned text, lists of strings, bytes as hex
func regData(kind string, data interface{}) (interface{}, error) {
	switch kind {
	case "DWord", "QWord":
		s := strings.TrimSpace(fmt.Sprint(data))
		if data == nil || s == "" {
			s = "0"
		}
		bits := 32
		if kind == "QWord" {
			bits = 64
		}
		var n uint64
		var err error
		if strings.HasPrefix(strings.ToLower(s), "0x") {
			n, err = strconv.ParseUint(s[2:], 16, bits)
		} else {
			n, err = strconv.ParseUint(s, 10, bits)
		}
		if err != nil {
			return nil, fmt.Errorf("data %q is not a %s", s, strings.ToLower(kind))
		}
		return strconv.FormatUint(n, 10), nil
	case "MultiString":
		switch v := data.(type) {
		case nil:
			return []string{}, nil
		case []interface{}:
			out := make([]string, len(v))
			for i, e := range v {
				out[i] = fmt.Sprint(e)
			}
			return out, nil
		case []string:
			return v, nil
		}
		return []string{fmt.Sprint(data)}, nil
	case "Binary", "None":
		switch v := data.(type) {
		case nil:
			return "", nil
		case []interface{}:
			b := make([]byte, len(v))
			for i, e := range v {
				n, ok := toInt(e)
				if !ok || n < 0 || n > 255 {
					return nil, fmt.Errorf("binary data: %v is not a byte", e)
				}
				b[i] = byte(n)
			}
			return hex.EncodeToString(b), nil
		}
		s := strings.ToLower(strings.TrimSpace(fmt.Sprint(data)))
		s = strings.TrimPrefix(s, "hex:")
		s = strings.NewReplacer(",", "", " ", "", "0x", "").Replace(s)
		if _, err := hex.DecodeString(s); err != nil {
			return nil, fmt.Errorf("binary data %q is not hex bytes", fmt.Sprint(data))
		}
		return s, nil
	}
	if data == nil {
		return "", nil
	}
	return fmt.Sprint(data), nil
}

// winRegeditScript compares and changes one key or value; $name $null is
// the key itself, ” the default value
const winRegeditScript = `$ErrorActionPreference = 'Stop'
$path = %[1]s; $name = %[2]s; $kind = %[3]s; $state = %[4]s; $check = %[5]s; $deleteKey = %[6]s
$want = %[7]s
function ToValue($kind, $v) {
  switch ($kind) {
    'DWord' { return [BitConverter]::ToInt32([BitConverter]::GetBytes([uint32]$v), 0) }
    'QWord' { return [BitConverter]::ToInt64([BitConverter]::GetBytes([uint64]$v), 0) }
    'MultiString' { if ($null -eq $v) { return ,([string[]]@()) }; return ,([string[]]@($v)) }
    { $_ -in 'Binary', 'None' } { $b = New-Object byte[] ($v.Length / 2); for ($i = 0; $i -lt $b.Length; $i++) { $b[$i] = [Convert]::ToByte($v.Substring($i * 2, 2), 16) }; return ,$b }
    default { return [string]$v }
  }
}
function Same($a, $b) {
  if ($a -is [array] -or $b -is [array]) { return (@($a).Count -eq @($b).Count) -and -not (Compare-Object @($a) @($b) -SyncWindow 0 -CaseSensitive) }
  return [string]$a -ceq [string]$b
}
$r = [ordered]@{ changed = $false; data_changed = $false; data_type_changed = $false }
$exists = Test-Path -LiteralPath $path
if ($state -eq 'absent') {
  if ($null -eq $name) {
    if ($exists -and $deleteKey) { $r.changed = $true; if (-not $check) { Remove-Item -LiteralPath $path -Recurse -Force } }
  } elseif ($exists) {
    $key = Get-Item -LiteralPath $path
    if ($key.GetValueNames() -contains $name) { $r.changed = $true; if (-not $check) { Remove-ItemProperty -LiteralPath $path -Name $name -Force } }
  }
} else {
  if (-not $exists) { $r.changed = $true; if (-not $check) { New-Item -Path $path -Force | Out-Null } }
  if ($null -ne $name) {
    $value = ToValue $kind $want
    $cur = $null; $curKind = $null
    if ($exists) {
      $key = Get-Item -LiteralPath $path
      if ($key.GetValueNames() -contains $name) {
        $cur = $key.GetValue($name, $null, 'DoNotExpandEnvironmentNames')
        $curKind = "$($key.GetValueKind($name))"
        $r.old_data = $cur
        $r.old_type = $curKind
      }
    }
    if ($curKind -ne $kind) { $r.data_type_changed = $null -ne $curKind }
    if ($curKind -ne $kind -or -not (Same $cur $value)) {
      $r.changed = $true; $r.data_changed = $true
      if (-not $check) { New-ItemProperty -LiteralPath $path -Name $name -PropertyType $kind -Value $value -Force | Out-Null }
    }
  }
}
%[8]s`

func (m *WinRegeditModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	start := time.Now()
	res := types.TaskResult{TaskName: taskName(args), Host: host.Name, Module: m.name, Timestamp: start,
		Output: map[string]interface{}{}}
	fail := func(err error) (types.TaskResult, error) {
		res.Failed, res.Error = true, err.Error()
		res.Duration = time.Since(start)
		return res, nil
	}
	script, err := regeditScript(args)
	if err != nil {
		return fail(err)
	}
	c, err := winClient(host, args)
	if err != nil {
		return fail(err)
	}
	out, _, err := runWinJSON(ctx, c, script)
	if err != nil {
		return fail(err)
	}
	for k, v := range out {
		if k != "changed" {
			res.Output[k] = v
		}
	}
	res.Changed, _ = out["changed"].(bool)
	res.Success = true
	res.Duration = time.Since(start)
	return res, nil
}

// regeditScript is the PowerShell that brings the key or value to the
// task's state
func regeditScript(args map[string]interface{}) (string, error) {
	path, err := regProviderPath(getStringArg(args, "path", getStringArg(args, "key", "")))
	if err != nil {
		return "", err
	}
	state := getStringArg(args, "state", "present")
	if state != "present" && state != "absent" {
		return "", fmt.Errorf("state must be present or absent")
	}
	typ := strings.ToLower(getStringArg(args, "type", "string"))
	kind, ok := regKinds[typ]
	if !ok {
		return "", fmt.Errorf("type %q: none, binary, dword, expandstring, multistring, string or qword", typ)
	}
	nameLit := "$null"
	if v, has := args["name"]; has && v != nil {
		nameLit = psQuote(fmt.Sprint(v))
	}
	data, err := regData(kind, args["data"])
	if err != nil {
		return "", err
	}
	want, err := psJSON(data)
	if err != nil {
		return "", err
	}
	b := func(v bool) string {
		if v {
			return "$true"
		}
		return "$false"
	}
	return fmt.Sprintf(winRegeditScript, psQuote(path), nameLit, psQuote(kind), psQuote(state),
		b(inCheckMode(args)), b(getBoolArg(args, "delete_key", true)), want, winPrintResult(4)), nil
}
