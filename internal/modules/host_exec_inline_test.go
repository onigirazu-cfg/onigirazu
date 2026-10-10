package modules

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// any bytes survive the trip through the command line
func TestInlineInstallScript(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("install -D is GNU")
	}
	dir := t.TempDir()
	data := append([]byte("it's \"quoted\" $HOME `x`\n"), 0, 1, 2, 255)
	dest := filepath.Join(dir, "sub", "f")
	tmp := filepath.Join(dir, "tmp-f")
	out, err := exec.Command("sh", "-c", inlineInstallScript(tmp, dest, "", data, 0o640)).CombinedOutput()
	if err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	got, err := os.ReadFile(dest)
	if err != nil || !bytes.Equal(got, data) {
		t.Fatalf("content %q %v", got, err)
	}
	if st, _ := os.Stat(dest); st.Mode().Perm() != 0o640 {
		t.Errorf("mode %v", st.Mode())
	}
	if _, err := os.Stat(tmp); err == nil {
		t.Errorf("temporary file left behind")
	}
}
