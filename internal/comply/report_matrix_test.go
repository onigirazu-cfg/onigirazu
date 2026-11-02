package comply

import (
	"bytes"
	"strings"
	"testing"
)

func TestControlMatrixPartialAndUnreachable(t *testing.T) {
	r := &Report{Hosts: map[string]*HostResult{
		"db1": {Controls: []ControlResult{{ID: "a", Title: `SSH <config>`, OK: true}, {ID: "b", Title: "Firewall", OK: false}}},
		"db2": {Controls: []ControlResult{{ID: "b", Title: "Firewall", OK: true}}},
	}, Errors: map[string]string{"db3": "offline"}}
	var b bytes.Buffer
	r.writeControlMatrix(&b)
	out := b.String()
	for _, want := range []string{
		`<th scope="row">a SSH &lt;config&gt;</th>`,
		`aria-label="db1: a SSH &lt;config&gt;: pass"`,
		`aria-label="db1: b Firewall: fail"`,
		`aria-label="db2: a SSH &lt;config&gt;: not checked"`,
		`aria-label="db3: b Firewall: unreachable"`,
		`class="onz-square fail" role="img" title="b Firewall: fail"`,
		`class="onz-square unchecked" role="img" title="a SSH &lt;config&gt;: not checked"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s", want)
		}
	}
	if strings.Contains(out, "<config>") {
		t.Fatal("matrix metadata is not escaped")
	}
	if strings.Index(out, `>db1</th>`) > strings.Index(out, `>db2</th>`) {
		t.Fatal("host columns are not sorted")
	}
}

func TestControlMatrixKeepsProfileOrder(t *testing.T) {
	r := &Report{Hosts: map[string]*HostResult{
		"db1": {Controls: []ControlResult{{ID: "ssh-2", OK: true}, {ID: "ssh-10", OK: true}}},
	}}
	var b bytes.Buffer
	r.writeControlMatrix(&b)
	out := b.String()
	if strings.Index(out, `>ssh-2 `) > strings.Index(out, `>ssh-10 `) {
		t.Fatal("rows must follow profile order, not string order")
	}
}

func TestControlMatrixMergesPartialHosts(t *testing.T) {
	r := &Report{Hosts: map[string]*HostResult{
		"a1": {Controls: []ControlResult{{ID: "c3", OK: true}}},
		"b1": {Controls: []ControlResult{{ID: "c1", OK: true}, {ID: "c2", OK: true}, {ID: "c3", OK: true}}},
		"c1": nil,
	}}
	var b bytes.Buffer
	r.writeControlMatrix(&b)
	out := b.String()
	if !(strings.Index(out, `>c1 `) < strings.Index(out, `>c2 `) && strings.Index(out, `>c2 `) < strings.Index(out, `>c3 `)) {
		t.Fatalf("rows must keep profile order across partial hosts:\n%s", out)
	}
}
