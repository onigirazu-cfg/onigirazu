package modules

import (
	"regexp"
	"testing"
)

func TestReplaceWithin(t *testing.T) {
	content := "key = 1\n[section]\nkey = 1\n[other]\nkey = 1\n"
	re := regexp.MustCompile("(?m)^key = .*$")
	for _, c := range []struct {
		after, before, want string
		n                   int
	}{
		{"", "", "key = 2\n[section]\nkey = 2\n[other]\nkey = 2\n", 3},
		{`\[section\]`, "", "key = 1\n[section]\nkey = 2\n[other]\nkey = 2\n", 2},
		{"", `\[other\]`, "key = 2\n[section]\nkey = 2\n[other]\nkey = 1\n", 2},
		{`\[section\]`, `\[other\]`, "key = 1\n[section]\nkey = 2\n[other]\nkey = 1\n", 1},
		{`\[missing\]`, "", content, 0},
	} {
		got, n, err := replaceWithin(content, re, "key = 2", c.after, c.before)
		if err != nil {
			t.Fatal(err)
		}
		if got != c.want || n != c.n {
			t.Errorf("after=%q before=%q: got %q (%d), want %q (%d)", c.after, c.before, got, n, c.want, c.n)
		}
	}
	if _, _, err := replaceWithin(content, re, "x", "(", ""); err == nil {
		t.Error("expected an error for an invalid after")
	}
}
