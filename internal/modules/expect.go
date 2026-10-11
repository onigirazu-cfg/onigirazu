package modules

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// ExpectModule runs a command and answers its prompts (Ansible's expect):
// command, responses (prompt regexp -> answer, or a list of answers),
// chdir, creates, removes, timeout, echo. The host's expect(1) drives it.
type ExpectModule struct{ *BaseModule }

func NewExpectModule() *ExpectModule { return &ExpectModule{BaseModule: NewBaseModule("expect")} }

func (m *ExpectModule) GetDescription() string { return "Run a command and answer its prompts" }

// expectScript renders the expect(1) program
func expectScript(command string, responses map[string]interface{}, timeout int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "set timeout %d\n", timeout)
	b.WriteString("spawn sh -c " + tclQuote(command) + "\n")
	keys := make([]string, 0, len(responses))
	for k := range responses {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	b.WriteString("expect {\n")
	for _, k := range keys {
		switch v := responses[k].(type) {
		case []interface{}:
			// one answer per occurrence, the last one repeated
			b.WriteString("  -re " + tclQuote(k) + " {\n")
			fmt.Fprintf(&b, "    if {![info exists n_%d]} {set n_%d 0}\n", len(b.String()), len(b.String()))
			b.WriteString("    set answers [list")
			for _, a := range v {
				b.WriteString(" " + tclQuote(fmt.Sprint(a)))
			}
			b.WriteString("]\n    set i [expr {min($idx, [llength $answers]-1)}]\n    send -- \"[lindex $answers $i]\\r\"\n    incr idx\n    exp_continue\n  }\n")
		default:
			b.WriteString("  -re " + tclQuote(k) + " { send -- " + tclQuote(fmt.Sprint(v)+"\r") + "; exp_continue }\n")
		}
	}
	b.WriteString("  timeout { exit 124 }\n  eof\n}\n")
	b.WriteString("catch wait result\nexit [lindex $result 3]\n")
	return "set idx 0\n" + b.String()
}

func tclQuote(s string) string {
	r := strings.NewReplacer("\\", "\\\\", "\"", "\\\"", "[", "\\[", "]", "\\]", "$", "\\$", "\r", "\\r")
	return "\"" + r.Replace(s) + "\""
}

func (m *ExpectModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	start := time.Now()
	result := types.TaskResult{TaskName: taskName(args), Host: host.Name, Module: m.name, Timestamp: start, Success: true, Output: map[string]interface{}{}}
	fail := func(msg string) (types.TaskResult, error) {
		result.Success, result.Error, result.Duration = false, msg, time.Since(start)
		return result, nil
	}
	command := getStringArg(args, "command", getStringArg(args, "cmd", ""))
	responses, _ := args["responses"].(map[string]interface{})
	if command == "" || responses == nil {
		return fail("expect requires 'command' and 'responses'")
	}
	if c := getStringArg(args, "creates", ""); c != "" {
		if _, exists, _ := readHostFile(ctx, host, args, c); exists {
			result.Output["msg"] = c + " exists"
			result.Duration = time.Since(start)
			return result, nil
		}
	}
	if r := getStringArg(args, "removes", ""); r != "" {
		if _, exists, _ := readHostFile(ctx, host, args, r); !exists {
			result.Output["msg"] = r + " does not exist"
			result.Duration = time.Since(start)
			return result, nil
		}
	}
	if inCheckMode(args) {
		result.Output["msg"] = "skipped in check mode"
		result.Duration = time.Since(start)
		return result, nil
	}
	script := expectScript(command, responses, getIntArg(args, "timeout", 30))
	tmp := remoteTempName("expect", "exp")
	if err := writeHostFile(ctx, host, args, tmp, []byte(script), 0o600); err != nil {
		return fail(err.Error())
	}
	cmd := "command -v expect >/dev/null 2>&1 || { echo 'expect is not installed on the host' >&2; exit 127; }; "
	if d := getStringArg(args, "chdir", ""); d != "" {
		cmd += "cd " + shellQuote(d) + " && "
	}
	cmd += "expect -f " + shellQuote(tmp) + "; rc=$?; rm -f " + shellQuote(tmp) + "; exit $rc"
	out, err := runShellOnHost(ctx, host, args, cmd)
	result.Changed = true
	result.Output["stdout"] = strings.TrimSpace(out)
	if err != nil {
		return fail(fmt.Sprintf("expect: %v: %s", err, strings.TrimSpace(out)))
	}
	result.Duration = time.Since(start)
	return result, nil
}
