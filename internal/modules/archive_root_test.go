package modules

import "testing"

func TestArchiveRoot(t *testing.T) {
	for _, c := range []struct {
		paths []string
		want  string
	}{
		{[]string{"/opt/bench/conf"}, "/opt/bench/"},
		{[]string{"/opt/a/x", "/opt/b/y"}, "/opt/"},
		{[]string{"/opt/ab/x", "/opt/ac/y"}, "/opt/"},
		{[]string{"/var/log/*.log"}, "/var/log/"},
		{[]string{"/etc/hosts", "/opt/x"}, "/"},
		{[]string{"/srv/app/a", "/srv/app/b/"}, "/srv/app/"},
	} {
		if got := archiveRoot(c.paths); got != c.want {
			t.Errorf("%v: %q, want %q", c.paths, got, c.want)
		}
	}
}
