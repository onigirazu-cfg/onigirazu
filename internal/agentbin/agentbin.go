// Package agentbin holds the onigirazu-agent binaries for the common
// platforms, gzipped, built by go generate (the release runs it); a build
// without them finds none here and looks for files instead.
package agentbin

//go:generate go run ./gen

import (
	"bytes"
	"compress/gzip"
	"embed"
	"io"
)

//go:embed agents
var agents embed.FS

// Get is the agent for goos/goarch, if this build has it
func Get(goos, goarch string) ([]byte, bool) {
	data, err := agents.ReadFile("agents/onigirazu-agent-" + goos + "-" + goarch + ".gz")
	if err != nil {
		return nil, false
	}
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, false
	}
	out, err := io.ReadAll(r)
	if err != nil {
		return nil, false
	}
	return out, true
}
