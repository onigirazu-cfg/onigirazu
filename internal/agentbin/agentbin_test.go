package agentbin

import "testing"

func TestGet(t *testing.T) {
	if _, ok := Get("plan9", "mips"); ok {
		t.Error("no agent is built for plan9/mips")
	}
	// built only by go generate: a plain build has none
	if data, ok := Get("linux", "amd64"); ok && len(data) < 1<<20 {
		t.Errorf("linux/amd64 agent of %d bytes", len(data))
	}
}
