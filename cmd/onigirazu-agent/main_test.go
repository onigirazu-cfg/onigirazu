package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

type client struct {
	w io.Writer
	r *bufio.Reader
}

func (c *client) do(t *testing.T, mode, payload string) (string, string, int) {
	t.Helper()
	fmt.Fprintf(c.w, "%s %s\n", mode, base64.StdEncoding.EncodeToString([]byte(payload)))
	head, err := c.r.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	f := strings.Fields(head)
	if len(f) != 4 || f[0] != "ONIGIRAZU" {
		t.Fatalf("answer %q", head)
	}
	rc, _ := strconv.Atoi(f[1])
	no, _ := strconv.Atoi(f[2])
	ne, _ := strconv.Atoi(f[3])
	buf := make([]byte, no+ne)
	if _, err := io.ReadFull(c.r, buf); err != nil {
		t.Fatal(err)
	}
	return string(buf[:no]), string(buf[no:]), rc
}

func start(t *testing.T) *client {
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	work := t.TempDir()
	go func() { _ = serve(inR, outW, work); outW.Close() }()
	t.Cleanup(func() { inW.Close() })
	c := &client{w: inW, r: bufio.NewReader(outR)}
	if ready, _ := c.r.ReadString('\n'); ready != "ONIGIRAZU-READY P\n" {
		t.Fatalf("ready %q", ready)
	}
	return c
}

func TestCommands(t *testing.T) {
	c := start(t)
	o, e, rc := c.do(t, "S", "echo out; echo err >&2; exit 3")
	if o != "out\n" || e != "err\n" || rc != 3 {
		t.Errorf("S: %q %q %d", o, e, rc)
	}
	o, e, rc = c.do(t, "C", "echo out; echo err >&2")
	if o != "out\nerr\n" || e != "" || rc != 0 {
		t.Errorf("C: %q %q %d", o, e, rc)
	}
	// a background child keeps its output open: the answer does not wait
	start := time.Now()
	if _, _, rc := c.do(t, "S", "sleep 5 >/dev/null 2>&1 & echo started"); rc != 0 || time.Since(start) > 3*time.Second {
		t.Errorf("background: rc %d after %v", rc, time.Since(start))
	}
	if _, _, rc := c.do(t, "S", "kill -TERM $$"); rc != 128+15 {
		t.Errorf("signal: rc %d", rc)
	}
}

func TestProbe(t *testing.T) {
	c := start(t)
	dir := t.TempDir()
	p := filepath.Join(dir, "f")
	if err := os.WriteFile(p, []byte("v1\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	_ = os.Chmod(p, 0o640)
	o, _, rc := c.do(t, "P", "100 "+p)
	f := strings.Fields(strings.SplitN(o, "\n", 2)[0])
	if rc != 0 || len(f) != 6 || f[0] != "file" || f[1] != "640" || f[4] != "3" || f[5] != "+" || !strings.Contains(o, "\nC:djEK\n") {
		t.Errorf("P: %q", o)
	}
	o, _, _ = c.do(t, "P", "2 "+p)
	if f := strings.Fields(o); len(f) != 6 || len(f[5]) != 64 {
		t.Errorf("P over the limit: %q", o)
	}
	o, _, _ = c.do(t, "Q", "100\n"+p+"\n"+filepath.Join(dir, "none")+"\n"+dir)
	r := strings.Split(o, "\x1e\n")
	if len(r) != 4 || !strings.HasPrefix(r[0], "file 640 ") || r[1] != "absent\n" || !strings.HasPrefix(r[2], "directory ") || r[3] != "" {
		t.Errorf("Q: %q", o)
	}
}

func TestBadPayload(t *testing.T) {
	c := start(t)
	fmt.Fprintf(c.w, "S !!!\n")
	if head, _ := c.r.ReadString('\n'); head != "ONIGIRAZU 255 0 0\n" {
		t.Errorf("answer %q", head)
	}
}

func TestProbeSpecialBits(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o777|os.ModeSticky); err != nil {
		t.Fatal(err)
	}
	ids := &names{users: map[uint32]string{}, groups: map[uint32]string{}}
	rec, err := probe(ids, 10, dir)
	if err != nil || !strings.HasPrefix(string(rec), "directory 1777 ") {
		t.Errorf("sticky directory: %q %v", rec, err)
	}
}
