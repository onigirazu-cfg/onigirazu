package modules

import (
	"context"
	"strings"
	"testing"
)

func TestZypperParsers(t *testing.T) {
	updates := `Loading repository data...
S | Repository | Name | Current Version | Available Version | Arch
--+------------+------+-----------------+-------------------+-------
v | Update     | curl | 8.0.1-1.1       | 8.0.1-2.1         | aarch64
`
	rows := zypperTable(updates)
	if len(rows) != 1 || rows[0][2] != "curl" || rows[0][4] != "8.0.1-2.1" {
		t.Errorf("rows = %q", rows)
	}
	info := "Name           : bash\nVersion        : 4.4-150400.27.6.1\nStatus         : up-to-date\n"
	if v := zypperField(info, "Version"); v != "4.4-150400.27.6.1" {
		t.Errorf("Version = %q", v)
	}
	if v := zypperField("Depends On      : glibc  ncurses\n", "Depends On"); v != "glibc  ncurses" {
		t.Errorf("Depends On = %q", v)
	}
}

func TestUnsupportedPackageManager(t *testing.T) {
	op, err := (&unsupportedPackageManager{tool: "choco"}).Install(context.Background(), "git", "")
	if err == nil || !strings.Contains(op.Error, "does not support choco") {
		t.Errorf("choco: %v, %+v", err, op)
	}
	if _, err := (&unsupportedPackageManager{}).IsInstalled(context.Background(), "git"); err == nil || !strings.Contains(err.Error(), "no supported package manager") {
		t.Errorf("none: %v", err)
	}
}

func TestPacmanRejectsVersion(t *testing.T) {
	if _, err := newCmdPackageManager("pacman", nil).Install(context.Background(), "tree", "2.0"); err == nil {
		t.Error("pacman must refuse a version")
	}
}
