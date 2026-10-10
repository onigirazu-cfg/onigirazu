//go:build !windows

package main

import (
	"io/fs"
	"os"
	"os/exec"
	"syscall"
)

// the script the command is written to, and how it runs
const scriptName = "c"

func commandFor(script string) *exec.Cmd { return exec.Command("sh", script) }

// exitCodeOf is the exit code, 128+signal for a command a signal ended
func exitCodeOf(err *exec.ExitError) int {
	if st, ok := err.Sys().(syscall.WaitStatus); ok && st.Signaled() {
		return 128 + int(st.Signal())
	}
	return err.ExitCode()
}

func ownerIDsOK(st fs.FileInfo) (uid, gid uint32, ok bool) {
	sys, ok := st.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0, false
	}
	return sys.Uid, sys.Gid, true
}

func ownerIDs(st fs.FileInfo) (uid, gid uint32) {
	uid, gid, _ = ownerIDsOK(st)
	return uid, gid
}

// ownsFile tells whether the file belongs to this process's user
func ownsFile(st fs.FileInfo) bool {
	uid, _, ok := ownerIDsOK(st)
	return ok && int(uid) == os.Geteuid()
}

func chownFile(path string, uid, gid int) error { return os.Chown(path, uid, gid) }
