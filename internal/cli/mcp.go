package cli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/spf13/cobra"

	"github.com/onigirazu-cfg/onigirazu/internal/audit"
	"github.com/onigirazu-cfg/onigirazu/internal/logger"
	"github.com/onigirazu-cfg/onigirazu/internal/version"
)

// mcp: onigirazu as a Model Context Protocol server over stdio, so an AI
// agent (Claude Code, Claude Desktop, any MCP client) can look at the
// inventory, plan and drift-check playbooks, run verify and compliance
// profiles and read the run history — read-only unless --allow-apply.
// The protocol is JSON-RPC 2.0, one message per line.

const mcpProtocolVersion = "2025-06-18"

type mcpOptions struct {
	allowApply bool
	inventory  string
}

func newMCPCmd() *cobra.Command {
	o := &mcpOptions{}
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Serve the Model Context Protocol over stdio for AI agents (read-only by default)",
		Long: `Expose onigirazu as MCP tools on stdin/stdout: inventory_list, inventory_host,
plan, drift, verify, comply, audit_runs, audit_run, module_doc — and apply
only with --allow-apply. Register it in an MCP client as the command
"onigirazu mcp -i hosts.yml". Logs go to stderr.`,
		Example: `  onigirazu mcp -i hosts.yml
  onigirazu mcp -i hosts.yml --allow-apply`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMCP(cmd.Context(), o, os.Stdin, os.Stdout)
		},
	}
	cmd.Flags().BoolVar(&o.allowApply, "allow-apply", false, "Offer the apply tool (changes hosts)")
	return cmd
}

type mcpRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type mcpResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *mcpError       `json:"error,omitempty"`
}

type mcpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type mcpTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
	run         func(args map[string]interface{}) (string, error)
}

func prop(t, desc string) map[string]interface{} {
	return map[string]interface{}{"type": t, "description": desc}
}

func schema(required []string, props map[string]interface{}) map[string]interface{} {
	s := map[string]interface{}{"type": "object", "properties": props}
	if len(required) > 0 {
		s["required"] = required
	}
	return s
}

func argString(args map[string]interface{}, key string) string {
	if v, ok := args[key]; ok && v != nil {
		return fmt.Sprint(v)
	}
	return ""
}

// playbookArgs are the arguments plan, drift, verify and apply share
func playbookArgs() map[string]interface{} {
	return map[string]interface{}{
		"playbook":   prop("string", "Path of the playbook"),
		"inventory":  prop("string", "Inventory path (default: the server's -i)"),
		"limit":      prop("string", "Host pattern, as apply --limit"),
		"tags":       prop("string", "Only tasks with these tags, comma separated"),
		"extra_vars": map[string]interface{}{"type": "object", "description": "Extra variables (name: value)"},
		"become":     prop("boolean", "Privilege escalation in every play"),
	}
}

func (o *mcpOptions) optionsFrom(args map[string]interface{}) driftCheckOptions {
	d := driftCheckOptions{limit: argString(args, "limit"), tags: argString(args, "tags")}
	if b, ok := args["become"].(bool); ok {
		d.become = b
	}
	if ev, ok := args["extra_vars"].(map[string]interface{}); ok {
		keys := make([]string, 0, len(ev))
		for k := range ev {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			v := ev[k]
			if _, isString := v.(string); !isString {
				if j, err := json.Marshal(v); err == nil {
					v = string(j)
				}
			}
			d.extraVars = append(d.extraVars, fmt.Sprintf("%s=%v", k, v))
		}
	}
	if inv := argString(args, "inventory"); inv != "" {
		inventoryPaths = []string{inv}
	} else if o.inventory != "" {
		inventoryPaths = []string{o.inventory}
	}
	return d
}

func (o *mcpOptions) tools() []mcpTool {
	tools := []mcpTool{
		{Name: "inventory_list", Description: "The inventory as ansible-inventory --list prints it: groups, hosts and host variables (passwords left out)",
			InputSchema: schema(nil, map[string]interface{}{"inventory": prop("string", "Inventory path (default: the server's -i)")}),
			run: func(args map[string]interface{}) (string, error) {
				o.optionsFrom(args)
				m, err := loadInventoryQuiet()
				if err != nil {
					return "", err
				}
				return jsonString(m.AnsibleList())
			}},
		{Name: "inventory_host", Description: "One host's variables as ansible-inventory --host prints them",
			InputSchema: schema([]string{"host"}, map[string]interface{}{"host": prop("string", "Host name"), "inventory": prop("string", "Inventory path")}),
			run: func(args map[string]interface{}) (string, error) {
				o.optionsFrom(args)
				m, err := loadInventoryQuiet()
				if err != nil {
					return "", err
				}
				vars, ok := m.AnsibleHostVars(argString(args, "host"))
				if !ok {
					return "", fmt.Errorf("host %s is not in the inventory", argString(args, "host"))
				}
				return jsonString(vars)
			}},
		{Name: "plan", Description: "What apply would change on the hosts (check mode, nothing changes): per host, every task that would change, with file diffs. Markdown.",
			InputSchema: schema([]string{"playbook"}, playbookArgs()),
			run: func(args map[string]interface{}) (string, error) {
				d := o.optionsFrom(args)
				result, err := runPlaybook(d.applyArgs(argString(args, "playbook"), true))
				if err != nil {
					return "", err
				}
				report := buildDriftReport(argString(args, "playbook"), result)
				report.Plan = true
				var b strings.Builder
				writeDriftMarkdown(&b, report)
				return b.String(), nil
			}},
		{Name: "drift", Description: "Check that hosts still match a playbook (check mode): the drifting tasks per host, with diffs. Markdown.",
			InputSchema: schema([]string{"playbook"}, playbookArgs()),
			run: func(args map[string]interface{}) (string, error) {
				d := o.optionsFrom(args)
				result, err := runPlaybook(d.applyArgs(argString(args, "playbook"), true))
				if err != nil {
					return "", err
				}
				report := buildDriftReport(argString(args, "playbook"), result)
				recordDriftHistory(argString(args, "playbook"), report)
				var b strings.Builder
				writeDriftMarkdown(&b, report)
				return b.String(), nil
			}},
		{Name: "verify", Description: "Run the playbook's verify: checks on the hosts and report them (JSON: per host the checks, passed and failed)",
			InputSchema: schema([]string{"playbook"}, playbookArgs()),
			run: func(args map[string]interface{}) (string, error) {
				d := o.optionsFrom(args)
				result, err := runPlaybook(append(d.applyArgs(argString(args, "playbook"), false), "--verify-only"))
				if err != nil {
					return "", err
				}
				return jsonString(buildVerifyReport(argString(args, "playbook"), result))
			}},
		{Name: "comply", Description: "Score the hosts against a compliance profile (bundled: linux-baseline, ssh; or a YAML file): controls with severity and fixes, per host. Markdown.",
			InputSchema: schema([]string{"profile"}, map[string]interface{}{
				"profile": prop("string", "Profile name or file"), "inventory": prop("string", "Inventory path"),
				"limit": prop("string", "Host pattern"), "control_tags": prop("string", "Only controls with one of these tags, comma separated"),
				"min_severity": prop("string", "Only controls of this severity or above")}),
			run: func(args map[string]interface{}) (string, error) {
				d := o.optionsFrom(args)
				d.become = true
				co := &complyOptions{driftCheckOptions: d, profile: argString(args, "profile"), minSeverity: argString(args, "min_severity")}
				if t := argString(args, "control_tags"); t != "" {
					co.tags = strings.Split(t, ",")
				}
				report, err := complyReport(co)
				if err != nil {
					return "", err
				}
				var b strings.Builder
				report.WriteMarkdown(&b)
				return b.String(), nil
			}},
		{Name: "audit_runs", Description: "The last runs from the audit store: id, playbook, user, start, status, task counts",
			InputSchema: schema(nil, map[string]interface{}{"host": prop("string", "Only runs that touched this host"), "playbook": prop("string", "Only this playbook"), "limit": prop("integer", "How many (default 20)")}),
			run: func(args map[string]interface{}) (string, error) {
				st, err := auditStorageQuiet()
				if err != nil {
					return "", err
				}
				defer st.Close()
				limit := 20
				if n, ok := args["limit"].(float64); ok && n > 0 {
					limit = int(n)
				}
				records, err := st.ListRecords(audit.FilterOptions{Limit: limit, HostFilter: argString(args, "host"), PlaybookPath: argString(args, "playbook"), SortBy: "time", SortOrder: "desc"})
				if err != nil {
					return "", err
				}
				rows := make([]map[string]interface{}, 0, len(records))
				for _, r := range records {
					rows = append(rows, map[string]interface{}{"id": r.ID, "playbook": r.PlaybookPath, "user": r.User, "start": r.StartTime, "duration": r.Duration,
						"status": r.Status, "tasks": r.TotalTasks, "failed": r.FailedTasks})
				}
				return jsonString(rows)
			}},
		{Name: "audit_run", Description: "One audit record in full: plays, hosts, tasks and their results",
			InputSchema: schema([]string{"id"}, map[string]interface{}{"id": prop("string", "Record id from audit_runs")}),
			run: func(args map[string]interface{}) (string, error) {
				st, err := auditStorageQuiet()
				if err != nil {
					return "", err
				}
				defer st.Close()
				rec, err := st.LoadRecord(argString(args, "id"))
				if err != nil {
					return "", err
				}
				return jsonString(rec)
			}},
		{Name: "module_doc", Description: "A module's arguments and description, as onigirazu doc prints them",
			InputSchema: schema([]string{"module"}, map[string]interface{}{"module": prop("string", "Module name, e.g. apt or ansible.builtin.copy")}),
			run: func(args map[string]interface{}) (string, error) {
				var b strings.Builder
				if err := showDoc(&b, argString(args, "module")); err != nil {
					return "", err
				}
				return b.String(), nil
			}},
	}
	if o.allowApply {
		tools = append(tools, mcpTool{Name: "apply", Description: "Run the playbook on the hosts — THIS CHANGES HOSTS. Returns per host the tasks that changed or failed.",
			InputSchema: schema([]string{"playbook"}, playbookArgs()),
			run: func(args map[string]interface{}) (string, error) {
				d := o.optionsFrom(args)
				result, err := runPlaybook(d.applyArgs(argString(args, "playbook"), false))
				if err != nil {
					return "", err
				}
				report := buildDriftReport(argString(args, "playbook"), result)
				var b strings.Builder
				writeDriftMarkdown(&b, report)
				return b.String(), nil
			}})
	}
	return tools
}

func jsonString(v interface{}) (string, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	return string(data), err
}

func loadInventoryQuiet() (inventoryManager, error) {
	lg, err := logger.NewEnhancedLogger("error", "text", os.Stderr)
	if err != nil {
		return nil, err
	}
	return loadInventory(context.Background(), lg)
}

// inventoryManager is what the tools need of the inventory
type inventoryManager interface {
	AnsibleList() map[string]interface{}
	AnsibleHostVars(host string) (map[string]interface{}, bool)
}

func auditStorageQuiet() (*audit.Storage, error) {
	lg, err := logger.NewEnhancedLogger("error", "text", os.Stderr)
	if err != nil {
		return nil, err
	}
	return audit.NewStorage(getAuditPath(), lg)
}

// mcpServer answers the protocol
type mcpServer struct {
	tools map[string]mcpTool
	list  []mcpTool
	out   io.Writer
	mu    sync.Mutex
}

func newMCPServer(o *mcpOptions, out io.Writer) *mcpServer {
	s := &mcpServer{tools: map[string]mcpTool{}, out: out}
	for _, t := range o.tools() {
		s.tools[t.Name] = t
		s.list = append(s.list, t)
	}
	return s
}

func (s *mcpServer) send(resp mcpResponse) {
	resp.JSONRPC = "2.0"
	data, _ := json.Marshal(resp)
	s.mu.Lock()
	defer s.mu.Unlock()
	_, _ = s.out.Write(append(data, '\n'))
}

// handle answers one request; notifications (no id) get no answer
func (s *mcpServer) handle(req mcpRequest) {
	if len(req.ID) == 0 {
		return
	}
	switch req.Method {
	case "initialize":
		s.send(mcpResponse{ID: req.ID, Result: map[string]interface{}{
			"protocolVersion": mcpProtocolVersion,
			"capabilities":    map[string]interface{}{"tools": map[string]interface{}{"listChanged": false}},
			"serverInfo":      map[string]interface{}{"name": "onigirazu", "version": versionString()},
			"instructions":    "Configuration management of the hosts in the inventory. plan and drift change nothing; apply exists only when the server was started with --allow-apply.",
		}})
	case "ping":
		s.send(mcpResponse{ID: req.ID, Result: map[string]interface{}{}})
	case "tools/list":
		public := make([]map[string]interface{}, 0, len(s.list))
		for _, t := range s.list {
			public = append(public, map[string]interface{}{"name": t.Name, "description": t.Description, "inputSchema": t.InputSchema})
		}
		s.send(mcpResponse{ID: req.ID, Result: map[string]interface{}{"tools": public}})
	case "tools/call":
		var p struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			s.send(mcpResponse{ID: req.ID, Error: &mcpError{Code: -32602, Message: "invalid params: " + err.Error()}})
			return
		}
		t, ok := s.tools[p.Name]
		if !ok {
			s.send(mcpResponse{ID: req.ID, Error: &mcpError{Code: -32602, Message: "unknown tool " + p.Name}})
			return
		}
		if p.Arguments == nil {
			p.Arguments = map[string]interface{}{}
		}
		text, err := t.run(p.Arguments)
		result := map[string]interface{}{"content": []map[string]interface{}{{"type": "text", "text": text}}}
		if err != nil {
			result["content"] = []map[string]interface{}{{"type": "text", "text": err.Error()}}
			result["isError"] = true
		}
		s.send(mcpResponse{ID: req.ID, Result: result})
	default:
		s.send(mcpResponse{ID: req.ID, Error: &mcpError{Code: -32601, Message: "method not found: " + req.Method}})
	}
}

func versionString() string {
	if v := strings.TrimSpace(version.Version); v != "" {
		return v
	}
	return "dev"
}

func runMCP(ctx context.Context, o *mcpOptions, in io.Reader, out io.Writer) error {
	if len(inventoryPaths) > 0 {
		o.inventory = inventoryPaths[0]
	}
	// stdout is the protocol channel: everything the tools print goes to
	// stderr instead
	realStdout := os.Stdout
	if f, ok := out.(*os.File); ok && f == realStdout {
		os.Stdout = os.Stderr
		defer func() { os.Stdout = realStdout }()
	}
	s := newMCPServer(o, out)
	fmt.Fprintf(os.Stderr, "onigirazu mcp: %d tool(s), inventory %q, apply %v\n", len(s.list), o.inventory, o.allowApply)
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 1<<20), 16<<20)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var req mcpRequest
		if err := json.Unmarshal(line, &req); err != nil {
			s.send(mcpResponse{Error: &mcpError{Code: -32700, Message: "parse error"}})
			continue
		}
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		s.handle(req)
	}
	return scanner.Err()
}
