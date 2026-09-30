// Package winrm runs commands and PowerShell on Windows hosts over WinRM,
// with the connection settings Ansible reads (ansible_winrm_*).
package winrm

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/masterzen/winrm"
	"golang.org/x/text/encoding/unicode"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// IsWinRM reports whether the host is reached over WinRM
func IsWinRM(host types.Host) bool {
	for _, key := range []string{"onigirazu_connection", "ansible_connection"} {
		switch fmt.Sprint(host.Vars[key]) {
		case "winrm", "ansible.builtin.winrm", "ansible.legacy.winrm":
			return true
		}
	}
	return false
}

// Settings are a host's WinRM connection settings
type Settings struct {
	Address    string
	Port       int
	HTTPS      bool
	Insecure   bool // no server certificate check
	Transport  string
	Encryption string // auto, always, never
	User       string
	Password   string
	Timeout    time.Duration
	CACert     []byte
}

func hostVar(host types.Host, name string) string {
	if v, ok := host.Vars[name]; ok && v != nil {
		return strings.TrimSpace(fmt.Sprint(v))
	}
	return ""
}

// SettingsFor reads the settings from the host as Ansible does: port 5986
// and https by default, ntlm transport, message encryption when the
// transport is http
func SettingsFor(host types.Host) (Settings, error) {
	s := Settings{Address: host.Address, Port: host.Port, User: host.User, Password: host.Password,
		Transport: "ntlm", Encryption: "auto", Timeout: 60 * time.Second}
	if s.Address == "" {
		s.Address = host.Name
	}
	if s.Password == "" {
		s.Password = hostVar(host, "ansible_password")
	}
	scheme := hostVar(host, "ansible_winrm_scheme")
	if s.Port == 0 {
		s.Port = 5986
		if scheme == "http" {
			s.Port = 5985
		}
	}
	switch scheme {
	case "":
		s.HTTPS = s.Port != 5985
	case "https":
		s.HTTPS = true
	case "http":
	default:
		return s, fmt.Errorf("ansible_winrm_scheme %q: http or https", scheme)
	}
	s.Insecure = hostVar(host, "ansible_winrm_server_cert_validation") == "ignore"
	if ca := hostVar(host, "ansible_winrm_ca_trust_path"); ca != "" {
		data, err := os.ReadFile(ca) // #nosec G304 -- the CA file the inventory names
		if err != nil {
			return s, fmt.Errorf("ansible_winrm_ca_trust_path: %w", err)
		}
		s.CACert = data
	}
	if t := hostVar(host, "ansible_winrm_transport"); t != "" {
		s.Transport = strings.ToLower(strings.TrimSpace(strings.Split(t, ",")[0]))
	}
	switch s.Transport {
	case "ntlm", "basic":
	default:
		return s, fmt.Errorf("ansible_winrm_transport %q is not supported (ntlm, basic)", s.Transport)
	}
	if e := hostVar(host, "ansible_winrm_message_encryption"); e != "" {
		s.Encryption = e
	}
	switch s.Encryption {
	case "auto", "always", "never":
	default:
		return s, fmt.Errorf("ansible_winrm_message_encryption %q: auto, always or never", s.Encryption)
	}
	if s.Encryption == "always" && s.Transport != "ntlm" {
		return s, fmt.Errorf("message encryption needs the ntlm transport")
	}
	for _, name := range []string{"ansible_winrm_operation_timeout_sec", "ansible_winrm_read_timeout_sec"} {
		if v := hostVar(host, name); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n <= 0 {
				return s, fmt.Errorf("%s %q is not a number of seconds", name, v)
			}
			if d := time.Duration(n) * time.Second; d > s.Timeout {
				s.Timeout = d
			}
		}
	}
	if s.User == "" {
		return s, fmt.Errorf("host %s: WinRM needs ansible_user", host.Name)
	}
	return s, nil
}

// Client runs commands on one Windows host
type Client struct {
	c *winrm.Client
}

// Connect creates a client for the host's settings
func Connect(s Settings) (*Client, error) {
	endpoint := winrm.NewEndpoint(s.Address, s.Port, s.HTTPS, s.Insecure, s.CACert, nil, nil, s.Timeout)
	params := *winrm.DefaultParameters
	params.Timeout = fmt.Sprintf("PT%dS", int(s.Timeout.Seconds()))
	encrypt := s.Transport == "ntlm" && (s.Encryption == "always" || (s.Encryption == "auto" && !s.HTTPS))
	switch {
	case encrypt:
		enc, err := winrm.NewEncryption("ntlm")
		if err != nil {
			return nil, err
		}
		params.TransportDecorator = func() winrm.Transporter { return enc }
	case s.Transport == "ntlm":
		params.TransportDecorator = func() winrm.Transporter { return &winrm.ClientNTLM{} }
	}
	c, err := winrm.NewClientWithParameters(endpoint, s.User, s.Password, &params)
	if err != nil {
		return nil, err
	}
	return &Client{c: c}, nil
}

// Result is what a command printed and its exit code
type Result struct {
	Stdout, Stderr string
	ExitCode       int
}

// RunCmd runs a command line with cmd.exe semantics
func (c *Client) RunCmd(ctx context.Context, command string) (Result, error) {
	out, errOut, code, err := c.c.RunCmdWithContext(ctx, command)
	return Result{Stdout: out, Stderr: errOut, ExitCode: code}, err
}

// RunCmdInput runs a command line with stdin
func (c *Client) RunCmdInput(ctx context.Context, command, stdin string) (Result, error) {
	out, errOut, code, err := c.c.RunWithContextWithString(ctx, command, stdin)
	return Result{Stdout: out, Stderr: errOut, ExitCode: code}, err
}

// bootstrap reads a base64 UTF-8 script from stdin and runs it, so scripts
// of any length pass the 8 KiB command line limit; $LASTEXITCODE or a
// terminating error become the exit code
const bootstrap = `$ProgressPreference='SilentlyContinue'
$s=[Text.Encoding]::UTF8.GetString([Convert]::FromBase64String([Console]::In.ReadToEnd()))
try { & ([ScriptBlock]::Create($s)); if ($LASTEXITCODE) { exit $LASTEXITCODE } }
catch { [Console]::Error.WriteLine($_.ToString()); exit 1 }`

// EncodedCommand is a powershell.exe command line running script
func EncodedCommand(script string) string {
	enc, _ := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewEncoder().String(script)
	return "powershell.exe -NoProfile -NonInteractive -ExecutionPolicy Bypass -EncodedCommand " +
		base64.StdEncoding.EncodeToString([]byte(enc))
}

// RunPS runs a PowerShell script
func (c *Client) RunPS(ctx context.Context, script string) (Result, error) {
	stdin := base64.StdEncoding.EncodeToString([]byte(script))
	out, errOut, code, err := c.c.RunWithContextWithString(ctx, EncodedCommand(bootstrap), stdin)
	return Result{Stdout: out, Stderr: cleanCLIXML(errOut), ExitCode: code}, err
}

// cleanCLIXML turns PowerShell's #< CLIXML error stream into plain text
func cleanCLIXML(s string) string {
	if !strings.HasPrefix(strings.TrimSpace(s), "#< CLIXML") {
		return s
	}
	var b strings.Builder
	for _, part := range strings.Split(s, `<S S="Error">`)[1:] {
		text, _, _ := strings.Cut(part, "</S>")
		text = strings.ReplaceAll(text, "_x000D__x000A_", "\n")
		text = strings.NewReplacer("&lt;", "<", "&gt;", ">", "&amp;", "&", "&quot;", `"`, "&apos;", "'").Replace(text)
		b.WriteString(text)
	}
	return strings.TrimRight(b.String(), "\n")
}

var (
	poolMu sync.Mutex
	pool   = map[string]*Client{}
)

// For returns the client of a host, one per host and settings
func For(host types.Host) (*Client, error) {
	s, err := SettingsFor(host)
	if err != nil {
		return nil, err
	}
	key := fmt.Sprintf("%s|%s:%d|%s|%s|%s", host.Name, s.Address, s.Port, s.User, s.Transport, s.Encryption)
	poolMu.Lock()
	defer poolMu.Unlock()
	if c, ok := pool[key]; ok {
		return c, nil
	}
	c, err := Connect(s)
	if err != nil {
		return nil, err
	}
	pool[key] = c
	return c, nil
}

// uploadChunk is the size of one piece of an upload: WinRM envelopes are
// limited (MaxEnvelopeSizekb, 500 KiB by default) and base64 grows data by
// a third
const uploadChunk = 128 * 1024

// Upload writes data to a temporary file on the host piece by piece and
// returns its path; the caller moves or removes it
func (c *Client) Upload(ctx context.Context, data []byte) (string, error) {
	res, err := c.RunPS(ctx, "$p = [IO.Path]::GetTempFileName(); Write-Output $p")
	if err != nil {
		return "", err
	}
	if res.ExitCode != 0 {
		return "", fmt.Errorf("temporary file: %s", strings.TrimSpace(res.Stderr))
	}
	path := strings.TrimSpace(res.Stdout)
	for off := 0; off < len(data) || off == 0; off += uploadChunk {
		end := off + uploadChunk
		if end > len(data) {
			end = len(data)
		}
		chunk := base64.StdEncoding.EncodeToString(data[off:end])
		script := fmt.Sprintf("$b = [Convert]::FromBase64String('%s'); $f = [IO.File]::Open('%s', 'Append'); $f.Write($b, 0, $b.Length); $f.Close()",
			chunk, strings.ReplaceAll(path, "'", "''"))
		res, err := c.RunPS(ctx, script)
		if err != nil {
			return "", err
		}
		if res.ExitCode != 0 {
			return "", fmt.Errorf("upload: %s", strings.TrimSpace(res.Stderr))
		}
		if len(data) == 0 {
			break
		}
	}
	return path, nil
}
