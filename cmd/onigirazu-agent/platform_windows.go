//go:build windows

package main

import (
	"io/fs"
	"os/exec"
)

// On Windows a command is a PowerShell script; there are no uids, modes
// are what Go reports, and ownership requests are accepted without effect.
const scriptName = "c.ps1"

func commandFor(script string) *exec.Cmd {
	return exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", script)
}

func exitCodeOf(err *exec.ExitError) int { return err.ExitCode() }

func ownerIDsOK(fs.FileInfo) (uint32, uint32, bool) { return 0, 0, false }
func ownerIDs(fs.FileInfo) (uint32, uint32)         { return 0, 0 }
func ownsFile(fs.FileInfo) bool                     { return true }
func chownFile(string, int, int) error              { return nil }
