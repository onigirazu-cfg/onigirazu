package modules

import (
	"context"
	"strings"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestUnsupportedParameters(t *testing.T) {
	if msg := unsupportedParameters("find", map[string]interface{}{"paths": "/tmp", "age_stamp": "atime", "_before": 1}); !strings.HasPrefix(msg, "Unsupported parameters for (find) module: age_stamp. Supported parameters include: ") {
		t.Fatalf("got %q", msg)
	}
	if msg := unsupportedParameters("find", map[string]interface{}{"paths": "/tmp", "recurse": true}); msg != "" {
		t.Fatalf("known args: %q", msg)
	}
	// free-form and undescribed modules are not checked
	for _, m := range []string{"add_host", "set_fact", "no_such_module"} {
		if msg := unsupportedParameters(m, map[string]interface{}{"anything": 1}); msg != "" {
			t.Fatalf("%s: %q", m, msg)
		}
	}
	AcceptedArgs["ping"] = []string{"extra"}
	defer delete(AcceptedArgs, "ping")
	if msg := unsupportedParameters("ping", map[string]interface{}{"extra": 1}); msg != "" {
		t.Fatalf("accepted arg: %q", msg)
	}
}

func TestExecuteTaskFailsOnUnknownArgument(t *testing.T) {
	r := NewRegistry()
	task := &types.Task{Name: "t", Module: "ping", Args: map[string]interface{}{"bogus": true}}
	res, err := r.ExecuteTask(context.Background(), task, types.Host{Name: "h", Address: "127.0.0.1"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Failed || !strings.Contains(res.Error, "Unsupported parameters for (ping) module: bogus") {
		t.Fatalf("got failed=%v error=%q", res.Failed, res.Error)
	}
}
