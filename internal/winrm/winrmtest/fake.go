// Package winrmtest is a fake WinRM server for tests
package winrmtest

import (
	"encoding/base64"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"

	"golang.org/x/text/encoding/unicode"
)

// Server answers the WinRM shell protocol over plain HTTP with basic
// auth; handle gets the command line and stdin and returns the output
type Server struct {
	mu sync.Mutex
	// Handle gets the command line and stdin and returns the output
	Handle   func(command, stdin string) (stdout, stderr string, code int)
	stdin    strings.Builder
	command  string
	received bool
	stdinEnd bool
	polls    int
	// Commands are the command lines run so far
	Commands []string
}

const envelope = `<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope" xmlns:a="http://schemas.xmlsoap.org/ws/2004/08/addressing" xmlns:x="http://schemas.xmlsoap.org/ws/2004/09/transfer" xmlns:w="http://schemas.dmtf.org/wbem/wsman/1/wsman.xsd" xmlns:rsp="http://schemas.microsoft.com/wbem/wsman/1/windows/shell"><s:Header><a:Action>%s</a:Action></s:Header><s:Body>%s</s:Body></s:Envelope>`

var (
	commandRe = regexp.MustCompile(`(?s)<rsp:Command>(.*?)</rsp:Command>(?:\s*<rsp:Arguments>(.*?)</rsp:Arguments>)*`)
	argsRe    = regexp.MustCompile(`(?s)<rsp:Arguments>(.*?)</rsp:Arguments>`)
	stdinRe   = regexp.MustCompile(`(?s)<rsp:Stream[^>]*Name="stdin"[^>]*>(.*?)</rsp:Stream>`)
)

func (f *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if u, p, ok := r.BasicAuth(); !ok || u != "admin" || p != "secret" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	data, _ := io.ReadAll(r.Body)
	body := string(data)
	w.Header().Set("Content-Type", "application/soap+xml")
	f.mu.Lock()
	defer f.mu.Unlock()
	b64 := func(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }
	switch {
	case strings.Contains(body, "transfer/Create"):
		fmt.Fprintf(w, envelope, "http://schemas.xmlsoap.org/ws/2004/09/transfer/CreateResponse", `<rsp:Shell><rsp:ShellId>SHELL-1</rsp:ShellId></rsp:Shell>`)
	case strings.Contains(body, "shell/Command"):
		m := commandRe.FindStringSubmatch(body)
		cmd := html.UnescapeString(strings.TrimSpace(m[1]))
		cmd = strings.TrimSuffix(strings.TrimPrefix(cmd, "<![CDATA["), "]]>")
		for _, a := range argsRe.FindAllStringSubmatch(body, -1) {
			cmd += " " + html.UnescapeString(a[1])
		}
		f.command, f.received, f.stdinEnd, f.polls = cmd, false, false, 0
		f.stdin.Reset()
		f.Commands = append(f.Commands, cmd)
		fmt.Fprintf(w, envelope, "http://schemas.microsoft.com/wbem/wsman/1/windows/shell/CommandResponse", `<rsp:CommandResponse><rsp:CommandId>CMD-1</rsp:CommandId></rsp:CommandResponse>`)
	case strings.Contains(body, "shell/Send"):
		for _, m := range stdinRe.FindAllStringSubmatch(body, -1) {
			d, _ := base64.StdEncoding.DecodeString(strings.TrimSpace(m[1]))
			f.stdin.Write(d)
		}
		if strings.Contains(body, `End="true"`) {
			f.stdinEnd = true
		}
		fmt.Fprintf(w, envelope, "http://schemas.microsoft.com/wbem/wsman/1/windows/shell/SendResponse", `<rsp:SendResponse/>`)
	case strings.Contains(body, "shell/Receive"):
		if f.received {
			fmt.Fprintf(w, envelope, "http://schemas.microsoft.com/wbem/wsman/1/windows/shell/ReceiveResponse", `<rsp:ReceiveResponse><rsp:CommandState CommandId="CMD-1" State="http://schemas.microsoft.com/wbem/wsman/1/windows/shell/CommandState/Done"><rsp:ExitCode>0</rsp:ExitCode></rsp:CommandState></rsp:ReceiveResponse>`)
			return
		}
		// the command runs once stdin is complete: a script on stdin
		// always has it; other commands may send some (polls give the
		// client's sender goroutine time to start)
		f.polls++
		if !f.stdinEnd && (strings.Contains(DecodePS(f.command), "ReadToEnd") || f.polls < 20) {
			f.mu.Unlock()
			time.Sleep(5 * time.Millisecond)
			f.mu.Lock()
			fmt.Fprintf(w, envelope, "http://schemas.microsoft.com/wbem/wsman/1/windows/shell/ReceiveResponse", `<rsp:ReceiveResponse><rsp:CommandState CommandId="CMD-1" State="http://schemas.microsoft.com/wbem/wsman/1/windows/shell/CommandState/Running"></rsp:CommandState></rsp:ReceiveResponse>`)
			return
		}
		f.received = true
		out, errOut, code := f.Handle(f.command, f.stdin.String())
		fmt.Fprintf(w, envelope, "http://schemas.microsoft.com/wbem/wsman/1/windows/shell/ReceiveResponse", fmt.Sprintf(
			`<rsp:ReceiveResponse><rsp:Stream Name="stdout" CommandId="CMD-1">%s</rsp:Stream><rsp:Stream Name="stderr" CommandId="CMD-1">%s</rsp:Stream><rsp:CommandState CommandId="CMD-1" State="http://schemas.microsoft.com/wbem/wsman/1/windows/shell/CommandState/Done"><rsp:ExitCode>%d</rsp:ExitCode></rsp:CommandState></rsp:ReceiveResponse>`,
			b64(out), b64(errOut), code))
	default: // Signal, Delete
		fmt.Fprintf(w, envelope, "http://schemas.xmlsoap.org/ws/2004/09/transfer/DeleteResponse", "")
	}
}

// DecodePS returns the script of "powershell.exe ... -EncodedCommand X"
func DecodePS(command string) string {
	i := strings.LastIndex(command, "-EncodedCommand ")
	if i < 0 {
		return ""
	}
	raw, _ := base64.StdEncoding.DecodeString(strings.TrimSpace(command[i+len("-EncodedCommand "):]))
	s, _ := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder().Bytes(raw)
	return string(s)
}

// Script is the PowerShell script a RunPS sent on stdin, or ""
func Script(command, stdin string) string {
	if !strings.Contains(DecodePS(command), "ReadToEnd") {
		return ""
	}
	raw, _ := base64.StdEncoding.DecodeString(stdin)
	return string(raw)
}

// Start serves f (user admin, password secret) and returns its address
// and port; it stops with the test
func Start(t testing.TB, f *Server) (string, int) {
	t.Helper()
	ts := httptest.NewServer(f)
	t.Cleanup(ts.Close)
	host, port, _ := net.SplitHostPort(strings.TrimPrefix(ts.URL, "http://"))
	p, _ := strconv.Atoi(port)
	return host, p
}

// Host is an inventory host for the fake server
func Host(name, address string, port int) types.Host {
	return types.Host{Name: name, Address: address, Port: port, User: "admin", Password: "secret",
		Vars: map[string]interface{}{"ansible_connection": "winrm", "ansible_winrm_transport": "basic",
			"ansible_winrm_scheme": "http", "ansible_winrm_message_encryption": "never"}}
}
