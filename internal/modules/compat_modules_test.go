package modules

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestLineinfileBackrefs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(path, []byte("listen 80\nlisten 81\nname x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	host := types.Host{Name: "h", Address: "localhost"}
	run := func(args map[string]interface{}) types.TaskResult {
		args["path"] = path
		res, err := NewLineinfileModule().Execute(context.Background(), host, args)
		if err != nil || !res.Success {
			t.Fatalf("%+v %v", res, err)
		}
		return res
	}
	// the last matching line takes the groups
	if !run(map[string]interface{}{"regexp": `^listen (\d+)$`, "line": `listen \1 ssl`, "backrefs": true}).Changed {
		t.Error("backrefs on a matching line must change it")
	}
	// no match with backrefs: nothing added
	if run(map[string]interface{}{"regexp": `^nomatch`, "line": "x", "backrefs": true}).Changed {
		t.Error("backrefs without a match must not change the file")
	}
	got, _ := os.ReadFile(path)
	if string(got) != "listen 80\nlisten 81 ssl\nname x\n" {
		t.Errorf("file = %q", got)
	}
	if pythonTemplate(`a\1b\g<2>$c\g<name>`) != `a${1}b${2}$$c${name}` {
		t.Errorf("template = %q", pythonTemplate(`a\1b\g<2>$c\g<name>`))
	}
}

func TestListArg(t *testing.T) {
	cases := []struct {
		args map[string]interface{}
		want []string
	}{
		{map[string]interface{}{"patterns": "*.conf, *.copy"}, []string{"*.conf", "*.copy"}},
		{map[string]interface{}{"patterns": []interface{}{"a", "b"}}, []string{"a", "b"}},
		{map[string]interface{}{"pattern": "x"}, []string{"x"}},
		{map[string]interface{}{}, nil},
	}
	for _, c := range cases {
		if got := listArg(c.args, "patterns", "pattern"); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%v -> %v, want %v", c.args, got, c.want)
		}
	}
}
