package modules

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestScriptModuleCreation(t *testing.T) {
	m := &ScriptModule{BaseModule: NewBaseModule("script")}
	assert.NotNil(t, m)
	assert.Equal(t, "script", m.name)
}

func TestScriptModuleDescription(t *testing.T) {
	m := &ScriptModule{BaseModule: NewBaseModule("script")}
	desc := m.GetDescription()
	assert.NotEmpty(t, desc)
}

// cmd: the script and its arguments, as Ansible takes them
func TestScriptCmd(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/hello.sh"
	if err := os.WriteFile(path, []byte("#!/bin/sh\necho \"args: $*\"\n"), 0o755); err != nil { // #nosec G306 -- a test script
		t.Fatal(err)
	}
	m := NewScriptModule()
	if err := m.Validate(map[string]interface{}{"cmd": path}); err != nil {
		t.Errorf("cmd is accepted: %v", err)
	}
	res, err := m.Execute(context.Background(), types.Host{Name: "localhost", Address: "localhost",
		Vars: map[string]interface{}{"ansible_connection": "local"}}, map[string]interface{}{"cmd": path + " one two"})
	if err != nil || !res.Success || !strings.Contains(fmt.Sprint(res.Output["stdout"]), "args: one two") {
		t.Errorf("script cmd = %+v, %v", res, err)
	}
}
