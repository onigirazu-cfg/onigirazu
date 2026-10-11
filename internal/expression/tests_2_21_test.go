package expression

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTests221(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "f")
	_ = os.WriteFile(f, []byte("$ANSIBLE_VAULT;1.1;AES256\nabc\n"), 0o600)
	vars := map[string]interface{}{"f": f, "dir": dir, "done": map[string]interface{}{"finished": 1}, "job": map[string]interface{}{"started": 1, "finished": 0}}
	cases := map[string]bool{
		`{{ [1, 2] is all }}`: true, `{{ [1, 0] is all }}`: false, `{{ [0, 2] is any }}`: true, `{{ [0, 0] is any }}`: false,
		`{{ f is is_file }}`: true, `{{ dir is is_dir }}`: true, `{{ f is is_abs }}`: true, `{{ dir is is_mount }}`: false, `{{ '/' is mount }}`: true,
		`{{ f is same_file(f) }}`: true, `{{ f is vaulted_file }}`: true, `{{ '$ANSIBLE_VAULT;1.1;AES256' is vault_encrypted }}`: true,
		`{{ 'x' is vault_encrypted }}`: false, `{{ [1] is issubset([1, 2]) }}`: true, `{{ [1, 2] is issuperset([1]) }}`: true,
		`{{ 'http://a/b' is url }}`: true, `{{ 'mailto:a@b' is url }}`: false, `{{ 'mailto:a@b' is uri }}`: true, `{{ 'urn:isbn:1' is urn }}`: true,
		`{{ 'http://a' is uri(['https']) }}`: false, `{{ done is finished }}`: true, `{{ job is finished }}`: false, `{{ job is started }}`: true,
		`{{ done is successful }}`: true, `{{ done is reachable }}`: true, `{{ 'nan' | float is nan }}`: true, `{{ 1 is nan }}`: false,
	}
	for code, want := range cases {
		got, err := Eval(code, vars)
		if err != nil || got != want {
			t.Errorf("%s = %v, %v; want %v", code, got, err, want)
		}
	}
}
