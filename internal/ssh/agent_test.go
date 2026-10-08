package ssh

import "testing"

func TestPlatform(t *testing.T) {
	for in, want := range map[string]string{
		"Linux x86_64": "linux/amd64", "Linux aarch64": "linux/arm64", "Darwin arm64": "darwin/arm64",
		"FreeBSD amd64": "freebsd/amd64", "Linux armv7l": "linux/arm", "Linux i686": "linux/386",
	} {
		goos, goarch, ok := platform(in)
		if !ok || goos+"/"+goarch != want {
			t.Errorf("%s: %s/%s %v", in, goos, goarch, ok)
		}
	}
	for _, in := range []string{"", "SunOS sparc", "Linux mips"} {
		if _, _, ok := platform(in); ok {
			t.Errorf("%q: no agent expected", in)
		}
	}
}

func TestServerMode(t *testing.T) {
	t.Setenv("ONIGIRAZU_NO_PYTHON", "")
	defer SetRemoteServer("")
	SetRemoteServer("")
	if serverMode() != "auto" || !agentEnabled() {
		t.Errorf("default: %s", serverMode())
	}
	SetRemoteServer("python")
	if agentEnabled() || serverScript() != shellScript {
		t.Error("python: no agent, the Python script")
	}
	SetRemoteServer("sh")
	if agentEnabled() || serverScript() != posixServer {
		t.Error("sh: the POSIX script")
	}
	SetRemoteServer("auto")
	t.Setenv("ONIGIRAZU_NO_PYTHON", "1")
	if serverMode() != "sh" {
		t.Error("ONIGIRAZU_NO_PYTHON=1 is sh")
	}
}
