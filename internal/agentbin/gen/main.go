// gen builds onigirazu-agent for the platforms most hosts run and writes
// them gzipped to agents/ for agentbin to embed
package main

import (
	"compress/gzip"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

var platforms = [][2]string{{"linux", "amd64"}, {"linux", "arm64"}, {"linux", "arm"}, {"linux", "386"}, {"windows", "amd64"}, {"windows", "arm64"}}

func main() {
	tmp, err := os.MkdirTemp("", "onigirazu-agent")
	if err != nil {
		fail(err)
	}
	defer os.RemoveAll(tmp)
	for _, p := range platforms {
		bin := filepath.Join(tmp, p[0]+"-"+p[1])
		cmd := exec.Command("go", "build", "-trimpath", "-ldflags", "-s -w", "-o", bin, "../../cmd/onigirazu-agent")
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS="+p[0], "GOARCH="+p[1], "GOARM=6")
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			fail(fmt.Errorf("agent for %s/%s: %w", p[0], p[1], err))
		}
		data, err := os.ReadFile(bin) // #nosec G304 -- built just above
		if err != nil {
			fail(err)
		}
		out, err := os.Create(filepath.Join("agents", "onigirazu-agent-"+p[0]+"-"+p[1]+".gz"))
		if err != nil {
			fail(err)
		}
		zw, _ := gzip.NewWriterLevel(out, gzip.BestCompression)
		if _, err := zw.Write(data); err != nil {
			fail(err)
		}
		if err := zw.Close(); err != nil {
			fail(err)
		}
		if err := out.Close(); err != nil {
			fail(err)
		}
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "agentbin:", err)
	os.Exit(1)
}
