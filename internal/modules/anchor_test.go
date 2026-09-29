package modules

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestAnchorIndex(t *testing.T) {
	lines := []string{"[main]", "a=1", "[extra]", "b=2", "[extra]"}
	cases := []struct {
		name          string
		after, before string
		first         bool
		want          int
	}{
		{"default is EOF", "", "", false, 5},
		{"EOF keyword", "EOF", "", false, 5},
		{"BOF keyword", "", "BOF", false, 0},
		{"after last match", `^\[extra\]`, "", false, 5},
		{"after first match", `^\[extra\]`, "", true, 3},
		{"before last match", "", `^\[extra\]`, false, 4},
		{"before first match", "", `^\[extra\]`, true, 2},
		{"regex, not literal", `^a=\d$`, "", false, 2},
		{"no match goes to EOF", "", "^missing", false, 5},
		{"insertbefore wins over insertafter", `^\[main\]`, `^b=`, false, 3},
	}
	for _, c := range cases {
		got, err := anchorIndex(lines, c.after, c.before, c.first)
		if err != nil || got != c.want {
			t.Errorf("%s: got %d, %v; want %d", c.name, got, err, c.want)
		}
	}
	if _, err := anchorIndex(lines, "([", "", false); err == nil {
		t.Error("an invalid pattern must be an error")
	}
}

func TestBlockinfileAnchors(t *testing.T) {
	host := types.Host{Name: "h", Address: "localhost"}
	cases := []struct {
		name string
		args map[string]interface{}
		want string
	}{
		{"missing anchor appends", map[string]interface{}{"insertafter": "^nothing"},
			"one\ntwo\n# BEGIN ANSIBLE MANAGED BLOCK\nx\n# END ANSIBLE MANAGED BLOCK\n"},
		{"insertbefore BOF", map[string]interface{}{"insertbefore": "BOF"},
			"# BEGIN ANSIBLE MANAGED BLOCK\nx\n# END ANSIBLE MANAGED BLOCK\none\ntwo\n"},
		{"insertafter regex", map[string]interface{}{"insertafter": "^o.e$"},
			"one\n# BEGIN ANSIBLE MANAGED BLOCK\nx\n# END ANSIBLE MANAGED BLOCK\ntwo\n"},
	}
	for _, c := range cases {
		path := filepath.Join(t.TempDir(), "f")
		if err := os.WriteFile(path, []byte("one\ntwo\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		args := map[string]interface{}{"path": path, "block": "x\n"}
		for k, v := range c.args {
			args[k] = v
		}
		res, err := NewBlockinfileModule().Execute(context.Background(), host, args)
		if err != nil || !res.Success || !res.Changed {
			t.Fatalf("%s: %+v, %v", c.name, res, err)
		}
		got, _ := os.ReadFile(path)
		if string(got) != c.want {
			t.Errorf("%s:\n%s\nwant:\n%s", c.name, got, c.want)
		}
	}
}

func TestLineinfileAnchors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(path, []byte("[s]\na=1\n[s]\nb=2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	host := types.Host{Name: "h", Address: "localhost"}
	res, err := NewLineinfileModule().Execute(context.Background(), host,
		map[string]interface{}{"path": path, "line": "c=3", "insertafter": `^\[s\]`})
	if err != nil || !res.Success {
		t.Fatalf("%+v, %v", res, err)
	}
	if got, _ := os.ReadFile(path); string(got) != "[s]\na=1\n[s]\nc=3\nb=2\n" {
		t.Errorf("got %q", got)
	}
}
