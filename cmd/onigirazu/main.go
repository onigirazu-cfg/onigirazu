package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/onigirazu-cfg/onigirazu/internal/cli"
)

func main() {
	// a reader that goes away (apply | head, | grep -q) must not kill a run
	// halfway: it would leave hosts half-changed, the state unsaved and the
	// state lock held; writes to the closed pipe just fail instead
	signal.Ignore(syscall.SIGPIPE)
	code := 0
	if err := cli.Execute(); err != nil {
		var exit *cli.ExitError
		if errors.As(err, &exit) {
			if exit.Message != "" {
				fmt.Fprintln(os.Stderr, exit.Message)
			}
			code = exit.Code
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			code = 1
		}
	}
	// the dashboard asked to run the failed hosts again: the same command,
	// once this run has saved its state and cleaned up
	if args := cli.RerunArgs(); args != nil {
		fmt.Fprintf(os.Stderr, "\nRunning again on the failed hosts: %s\n", args[len(args)-1])
		self, err := os.Executable()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		cmd := exec.Command(self, args...) // #nosec G204 G702 -- this program again, with the user's own arguments
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				os.Exit(exitErr.ExitCode())
			}
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(code)
}
