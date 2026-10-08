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
