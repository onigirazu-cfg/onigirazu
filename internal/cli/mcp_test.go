package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMCPProtocol(t *testing.T) {
	dir := t.TempDir()
	inv := filepath.Join(dir, "hosts.yml")
	if err := os.WriteFile(inv, []byte("all:\n  hosts:\n    web1: {ansible_host: 10.0.0.1, role: web}\n  children:\n    web: {hosts: {web1: {}}}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := inventoryPaths
	defer func() { inventoryPaths = old }()
	inventoryPaths = []string{inv}

	in := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"0"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"inventory_host","arguments":{"host":"web1"}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"module_doc","arguments":{"module":"copy"}}}`,
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"apply","arguments":{"playbook":"x.yml"}}}`,
		`{"jsonrpc":"2.0","id":6,"method":"nope"}`,
	}, "\n") + "\n"
	var out bytes.Buffer
	if err := runMCP(context.Background(), &mcpOptions{}, strings.NewReader(in), &out); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 6 {
		t.Fatalf("6 answers for 6 requests (the notification gets none), got %d:\n%s", len(lines), out.String())
	}
	var resp []mcpResponse
	for _, l := range lines {
		var r mcpResponse
		if err := json.Unmarshal([]byte(l), &r); err != nil {
			t.Fatal(err)
		}
		resp = append(resp, r)
	}
	init, _ := json.Marshal(resp[0].Result)
	if !strings.Contains(string(init), `"protocolVersion":"2025-06-18"`) || !strings.Contains(string(init), `"name":"onigirazu"`) {
		t.Errorf("initialize: %s", init)
	}
	list, _ := json.Marshal(resp[1].Result)
	for _, name := range []string{"inventory_list", "plan", "drift", "verify", "comply", "audit_runs", "module_doc"} {
		if !strings.Contains(string(list), `"name":"`+name+`"`) {
			t.Errorf("tools/list lacks %s", name)
		}
	}
	if strings.Contains(string(list), `"name":"apply"`) {
		t.Error("apply is not offered without --allow-apply")
	}
	host, _ := json.Marshal(resp[2].Result)
	if !strings.Contains(string(host), `10.0.0.1`) || !strings.Contains(string(host), `\"role\": \"web\"`) {
		t.Errorf("inventory_host: %s", host)
	}
	doc, _ := json.Marshal(resp[3].Result)
	if !strings.Contains(string(doc), "dest") {
		t.Errorf("module_doc copy: %s", doc)
	}
	if resp[4].Error == nil || !strings.Contains(resp[4].Error.Message, "unknown tool apply") {
		t.Errorf("apply without --allow-apply: %+v", resp[4].Error)
	}
	if resp[5].Error == nil || resp[5].Error.Code != -32601 {
		t.Errorf("unknown method: %+v", resp[5].Error)
	}
	// with --allow-apply the tool exists
	s := newMCPServer(&mcpOptions{allowApply: true}, &bytes.Buffer{})
	if _, ok := s.tools["apply"]; !ok {
		t.Error("--allow-apply offers apply")
	}
}
