package cli

import (
	"strings"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestLintTemplatedBecome(t *testing.T) {
	result := &LintResult{}
	checkSecurity(&types.Playbook{Plays: []types.Play{{Tasks: []types.Task{{
		Name: "conditional", Module: "ping", BecomeSet: true, BecomeExpr: "{{ use_sudo }}",
	}}}}}, "play.yml", result)
	if result.Errors != 0 || result.Warns != 0 || result.Infos != 1 {
		t.Fatalf("unexpected lint result: %+v", result)
	}
	if !strings.Contains(result.Issues[0].Message, "may use") {
		t.Fatalf("expected conditional escalation message, got %q", result.Issues[0].Message)
	}
}
