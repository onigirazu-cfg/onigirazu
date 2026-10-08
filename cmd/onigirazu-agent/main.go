// onigirazu-agent is onigirazu's command server on a managed host: the
// protocol of the shell and Python servers (internal/ssh), with probes done
// in-process. onigirazu uploads it once per version (remote_server auto) and
// starts it as "onigirazu-agent serve" over SSH, with sudo for become.
package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

func main() {
	if len(os.Args) != 2 || os.Args[1] != "serve" {
		fmt.Fprintln(os.Stderr, "usage: onigirazu-agent serve")
		os.Exit(2)
	}
	work, err := workDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(97)
	}
	err = serve(os.Stdin, os.Stdout, work)
	_ = os.RemoveAll(work)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// workDir holds the script and output files: under ~/.onigirazu/tmp, never
// in the home of another user (sudo may keep HOME)
func workDir() (string, error) {
	if home, err := os.UserHomeDir(); err == nil {
		if st, err := os.Stat(home); err == nil {
			if sys, ok := st.Sys().(*syscall.Stat_t); ok && int(sys.Uid) == os.Geteuid() {
				root := filepath.Join(home, ".onigirazu", "tmp")
				if os.MkdirAll(root, 0o700) == nil {
					if d, err := os.MkdirTemp(root, "go."); err == nil {
						return d, nil
					}
				}
			}
		}
	}
	return os.MkdirTemp("", "onigirazu.")
}

func serve(in io.Reader, out io.Writer, work string) error {
	r := bufio.NewReaderSize(in, 64*1024)
	w := bufio.NewWriterSize(out, 64*1024)
	if _, err := w.WriteString("ONIGIRAZU-READY P W\n"); err != nil {
		return err
	}
	if err := w.Flush(); err != nil {
		return err
	}
	ids := &names{users: map[uint32]string{}, groups: map[uint32]string{}}
	for n := 1; ; n++ {
		line, err := r.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		mode, data, _ := strings.Cut(strings.TrimSuffix(line, "\n"), " ")
		payload, err := base64.StdEncoding.DecodeString(data)
		var so, se []byte
		rc := 0
		switch {
		case err != nil:
			rc = 255
		case mode == "P":
			limit, path, _ := strings.Cut(string(payload), " ")
			so, se, rc = probeAnswer(ids, limit, path)
		case mode == "W":
			if werr := writeFile(payload); werr != nil {
				se, rc = []byte(werr.Error()+"\n"), 1
			}
		case mode == "Q":
			limit, paths, _ := strings.Cut(string(payload), "\n")
			var b bytes.Buffer
			for _, p := range strings.Split(paths, "\n") {
				rec, perr := probe(ids, atoi(limit), p)
				if perr != nil {
					rec = []byte("error " + perr.Error() + "\n")
				}
				b.Write(rec)
				b.WriteString("\x1e\n")
			}
			so = b.Bytes()
		default:
			so, se, rc = run(work, n, payload, mode == "C")
		}
		if _, err := fmt.Fprintf(w, "ONIGIRAZU %d %d %d\n", rc, len(so), len(se)); err != nil {
			return err
		}
		_, _ = w.Write(so)
		_, _ = w.Write(se)
		if err := w.Flush(); err != nil {
			return err
		}
	}
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func probeAnswer(ids *names, limit, path string) ([]byte, []byte, int) {
	rec, err := probe(ids, atoi(limit), path)
	if err != nil {
		return nil, []byte(err.Error() + "\n"), 1
	}
	return rec, nil, 0
}

// run runs a command with sh; output goes through files, so a background
// process the command leaves behind does not hold up the answer
func run(work string, n int, command []byte, combined bool) ([]byte, []byte, int) {
	if err := os.MkdirAll(work, 0o700); err != nil {
		return nil, []byte(err.Error()), 255
	}
	script := filepath.Join(work, "c")
	if err := os.WriteFile(script, command, 0o600); err != nil {
		return nil, []byte(err.Error()), 255
	}
	o, e := filepath.Join(work, "o"+strconv.Itoa(n)), filepath.Join(work, "e"+strconv.Itoa(n))
	fo, err := os.Create(o)
	if err != nil {
		return nil, []byte(err.Error()), 255
	}
	fe := fo
	if !combined {
		if fe, err = os.Create(e); err != nil {
			_ = fo.Close()
			return nil, []byte(err.Error()), 255
		}
	}
	cmd := exec.Command("sh", script)
	cmd.Stdout, cmd.Stderr = fo, fe
	devnull, _ := os.Open(os.DevNull)
	cmd.Stdin = devnull
	err = cmd.Run()
	if devnull != nil {
		_ = devnull.Close()
	}
	_ = fo.Close()
	if !combined {
		_ = fe.Close()
	}
	rc := 0
	var exitErr *exec.ExitError
	switch {
	case errors.As(err, &exitErr):
		if st, ok := exitErr.Sys().(syscall.WaitStatus); ok && st.Signaled() {
			rc = 128 + int(st.Signal())
		} else {
			rc = exitErr.ExitCode()
		}
	case err != nil:
		return nil, []byte(err.Error()), 127
	}
	so, _ := os.ReadFile(o) // the command may have removed its own output
	var se []byte
	if !combined {
		se, _ = os.ReadFile(e)
	}
	_ = os.Remove(o)
	_ = os.Remove(e)
	return so, se, rc
}

type names struct {
	users, groups map[uint32]string
}

func (n *names) user(id uint32) string {
	if s, ok := n.users[id]; ok {
		return s
	}
	s := "UNKNOWN"
	if u, err := user.LookupId(strconv.Itoa(int(id))); err == nil {
		s = u.Username
	}
	n.users[id] = s
	return s
}

func (n *names) group(id uint32) string {
	if s, ok := n.groups[id]; ok {
		return s
	}
	s := "UNKNOWN"
	if g, err := user.LookupGroupId(strconv.Itoa(int(id))); err == nil {
		s = g.Name
	}
	n.groups[id] = s
	return s
}

// probe describes path as the shell probe of the file modules' capture
// prints it: "absent", or "kind mode owner group size" and "+" with a
// "C:" line of the content up to limit bytes, else its sha256, "-" for
// what is not a file
func probe(ids *names, limit int, path string) ([]byte, error) {
	st, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return []byte("absent\n"), nil
	}
	if err != nil {
		return nil, err
	}
	kind := "other"
	switch m := st.Mode(); {
	case m&fs.ModeSymlink != 0:
		kind = "link"
	case m.IsDir():
		kind = "directory"
	case m.IsRegular():
		kind = "file"
	}
	// the permission bits with setuid, setgid and sticky, as stat %a prints
	m := st.Mode()
	perm := uint32(m.Perm())
	for bit, v := range map[fs.FileMode]uint32{fs.ModeSetuid: 0o4000, fs.ModeSetgid: 0o2000, fs.ModeSticky: 0o1000} {
		if m&bit != 0 {
			perm |= v
		}
	}
	var uid, gid uint32
	if sys, ok := st.Sys().(*syscall.Stat_t); ok {
		uid, gid = sys.Uid, sys.Gid
	}
	head := fmt.Sprintf("%s %o %s %s %d", kind, perm, ids.user(uid), ids.group(gid), st.Size())
	if kind != "file" {
		return []byte(head + " -\n"), nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if st.Size() <= int64(limit) {
		data, err := io.ReadAll(f)
		if err != nil {
			return nil, err
		}
		return []byte(head + " +\nC:" + base64.StdEncoding.EncodeToString(data) + "\n"), nil
	}
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, err
	}
	return []byte(head + " " + hex.EncodeToString(h.Sum(nil)) + "\n"), nil
}

// writeFile handles "MODE OWNER GROUP\npath\ncontent": MODE octal or "-"
// (keep, 0644 for a new file), OWNER/GROUP a name, an id or "-" (keep). The
// content goes to a file next to the target and is moved over it, as
// install(1) does; missing parent directories are made.
func writeFile(request []byte) error {
	head, rest, _ := bytes.Cut(request, []byte("\n"))
	path, data, ok := bytes.Cut(rest, []byte("\n"))
	f := strings.Fields(string(head))
	if !ok || len(f) != 3 || len(path) == 0 {
		return errors.New("bad write request")
	}
	target := string(path)
	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	st, statErr := os.Stat(target)
	mode := fs.FileMode(0o644)
	uid, gid := -1, -1
	if statErr == nil {
		mode = st.Mode().Perm() | st.Mode()&(fs.ModeSetuid|fs.ModeSetgid|fs.ModeSticky)
		if sys, isStat := st.Sys().(*syscall.Stat_t); isStat {
			uid, gid = int(sys.Uid), int(sys.Gid)
		}
	}
	if f[0] != "-" {
		m, err := strconv.ParseUint(f[0], 8, 32)
		if err != nil {
			return fmt.Errorf("bad mode %q", f[0])
		}
		mode = fs.FileMode(m & 0o777)
		if m&0o4000 != 0 {
			mode |= fs.ModeSetuid
		}
		if m&0o2000 != 0 {
			mode |= fs.ModeSetgid
		}
		if m&0o1000 != 0 {
			mode |= fs.ModeSticky
		}
	}
	if f[1] != "-" {
		id, err := lookupID(f[1], func(n string) (string, error) {
			u, err := user.Lookup(n)
			if err != nil {
				return "", err
			}
			return u.Uid, nil
		})
		if err != nil {
			return err
		}
		uid = id
	}
	if f[2] != "-" {
		id, err := lookupID(f[2], func(n string) (string, error) {
			g, err := user.LookupGroup(n)
			if err != nil {
				return "", err
			}
			return g.Gid, nil
		})
		if err != nil {
			return err
		}
		gid = id
	}
	tmp, err := os.CreateTemp(dir, ".onigirazu-")
	if err != nil {
		return err
	}
	ok = false
	defer func() {
		if !ok {
			_ = os.Remove(tmp.Name())
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), mode); err != nil {
		return err
	}
	if uid != -1 || gid != -1 {
		if err := os.Chown(tmp.Name(), uid, gid); err != nil {
			return err
		}
	}
	if err := os.Rename(tmp.Name(), target); err != nil {
		return err
	}
	ok = true
	return nil
}

// lookupID is a numeric id as it is, else the id of the name
func lookupID(s string, lookup func(string) (string, error)) (int, error) {
	if n, err := strconv.Atoi(s); err == nil {
		return n, nil
	}
	id, err := lookup(s)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(id)
}
