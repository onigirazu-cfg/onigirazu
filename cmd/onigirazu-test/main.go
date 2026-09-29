// onigirazu-test runs Molecule scenarios with onigirazu. As a command
// plugin it is "onigirazu test".
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/onigirazu-cfg/onigirazu/internal/moltest"
)

const usage = `Usage: onigirazu test [flags] [STEP]

Runs a Molecule scenario (molecule/<scenario>/molecule.yml of the role in the
current directory) with onigirazu, on docker or podman containers.

STEP is test (default: the whole sequence), or one of
  dependency create prepare converge idempotence side_effect verify cleanup destroy syntax
  login  (a shell in the instance)
  list   (the scenarios and their instances)

Flags:
`

func main() {
	os.Exit(run())
}

func run() int {
	fs := flag.NewFlagSet("onigirazu test", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, usage)
		fs.PrintDefaults()
	}
	scenario := fs.String("s", "default", "scenario name (all: every scenario)")
	fs.StringVar(scenario, "scenario-name", "default", "scenario name (all: every scenario)")
	destroy := fs.String("destroy", "always", "test: always or never destroy the instances at the end")
	noDeps := fs.Bool("no-deps", false, "skip the dependency step")
	host := fs.String("host", "", "login: the platform (default the first)")
	if err := fs.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	step := "test"
	if fs.NArg() > 0 {
		step = fs.Arg(0)
	}
	if *destroy != "always" && *destroy != "never" {
		fmt.Fprintln(os.Stderr, "--destroy is always or never")
		return 2
	}
	bin := os.Getenv("ONIGIRAZU_BIN")
	if bin == "" {
		p, err := exec.LookPath("onigirazu")
		if err != nil {
			fmt.Fprintln(os.Stderr, "onigirazu not found: run as \"onigirazu test\" or put onigirazu in PATH")
			return 1
		}
		bin = p
	}

	names := []string{*scenario}
	if *scenario == "all" || step == "list" {
		names = moltest.Scenarios(".")
		if len(names) == 0 {
			fmt.Fprintln(os.Stderr, "no molecule/*/molecule.yml here: run it in a role directory")
			return 1
		}
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	code := 0
	for _, name := range names {
		s, err := moltest.Load(".", name)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		r := &moltest.Runner{S: s, Bin: bin, Out: os.Stdout, NoDeps: *noDeps, Destroy: *destroy}
		switch step {
		case "list":
			for _, p := range s.Platforms {
				fmt.Printf("%-12s %-16s %-10s %s\n", name, p.Name, s.Driver.Name, r.Container(p))
			}
			continue
		case "login":
			return login(r, *host)
		case "test":
			err = r.Test(ctx)
		default:
			err = r.Run(ctx, step)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", name, err)
			code = 1
			continue
		}
		fmt.Printf("--> %s: %s passed\n", name, step)
	}
	return code
}

func login(r *moltest.Runner, host string) int {
	p := r.S.Platforms[0]
	for _, q := range r.S.Platforms {
		if q.Name == host {
			p = q
		}
	}
	if host != "" && p.Name != host {
		fmt.Fprintf(os.Stderr, "no platform %s (%s)\n", host, platformNames(r.S))
		return 1
	}
	shell := "if command -v bash >/dev/null; then exec bash; else exec sh; fi"
	cmd := exec.Command(r.S.Driver.Name, "exec", "-it", r.Container(p), "sh", "-c", shell) // #nosec G204 -- docker/podman exec into the scenario's container
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return exit.ExitCode()
		}
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func platformNames(s *moltest.Scenario) string {
	names := make([]string, 0, len(s.Platforms))
	for _, p := range s.Platforms {
		names = append(names, p.Name)
	}
	return strings.Join(names, ", ")
}
