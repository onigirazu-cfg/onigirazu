package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
	"github.com/spf13/cobra"
)

// verify runs only the verify: checks of a playbook's plays and reports them
// per host: what goss does after a converge, from the playbook itself.

func newVerifyCmd() *cobra.Command {
	var o driftCheckOptions
	cmd := &cobra.Command{
		Use:   "verify PLAYBOOK",
		Short: "Run the playbook's verify: checks against the hosts and report them",
		Long: `Each play may carry a verify: list of checks of the hosts' state (files,
packages, services, ports, processes, users, groups, commands, http, mounts,
kernel parameters, dns). apply runs them after the play; verify runs only them.
Exit code 1 when a check fails or a host cannot be checked.`,
		Example: `  onigirazu verify site.yml -i hosts.yml
  onigirazu verify site.yml -i hosts.yml --limit web --format json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVerify(cmd, args[0], o)
		},
	}
	cmd.Flags().StringVar(&o.format, "format", "text", "Report format (text, json)")
	cmd.Flags().StringVar(&o.output, "output", "", "Write the report to this file")
	cmd.Flags().StringArrayVarP(&o.extraVars, "extra-vars", "e", nil, "Extra variables, as for apply (repeatable)")
	cmd.Flags().StringVar(&o.limit, "limit", "", "Check only hosts matching this pattern")
	cmd.Flags().StringVar(&o.exclude, "exclude", "", "Leave out hosts matching this pattern")
	cmd.Flags().BoolVarP(&o.become, "become", "b", false, "Use privilege escalation in every play")
	cmd.Flags().StringVar(&o.becomeUser, "become-user", "", "User to become")
	cmd.Flags().StringVarP(&o.user, "user", "u", "", "SSH user for every host")
	cmd.Flags().StringVar(&o.privateKey, "private-key", "", "SSH private key for every host")
	return cmd
}

// VerifyReport is the result of verify, per host
type VerifyReport struct {
	Playbook string                   `json:"playbook"`
	Hosts    map[string]*HostVerify   `json:"hosts"`
	Passed   int                      `json:"passed"`
	Failed   int                      `json:"failed"`
	Errors   map[string]string        `json:"errors,omitempty"` // hosts that could not be checked
	Checks   []map[string]interface{} `json:"-"`
}

type HostVerify struct {
	Passed int                      `json:"passed"`
	Failed int                      `json:"failed"`
	Checks []map[string]interface{} `json:"checks"`
}

func runVerify(cmd *cobra.Command, playbook string, o driftCheckOptions) error {
	args := append(o.applyArgs(playbook, false), "--verify-only")
	result, err := runPlaybook(args)
	if err != nil {
		return err
	}
	report := buildVerifyReport(playbook, result)
	out := io.Writer(os.Stdout)
	if o.output != "" {
		f, err := os.Create(o.output)
		if err != nil {
			return err
		}
		defer f.Close()
		out = f
	}
	switch o.format {
	case "json":
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			return err
		}
	case "text", "":
		writeVerifyText(out, report)
	default:
		return fmt.Errorf("unknown format %q (text, json)", o.format)
	}
	if report.Failed > 0 || len(report.Errors) > 0 {
		return &ExitError{Code: 1}
	}
	return nil
}

func buildVerifyReport(playbook string, result *types.PlaybookResult) *VerifyReport {
	r := &VerifyReport{Playbook: playbook, Hosts: map[string]*HostVerify{}, Errors: map[string]string{}}
	for _, play := range result.Plays {
		for _, host := range play.Hosts {
			for _, t := range host.Tasks {
				if t.Module != "verify" {
					if t.Failed && !t.Ignored {
						r.Errors[host.Host] = t.Error
					}
					continue
				}
				h := r.Hosts[host.Host]
				if h == nil {
					h = &HostVerify{}
					r.Hosts[host.Host] = h
				}
				checks, _ := t.Output["checks"].([]map[string]interface{})
				if checks == nil {
					if list, ok := t.Output["checks"].([]interface{}); ok {
						for _, c := range list {
							if m, ok := c.(map[string]interface{}); ok {
								checks = append(checks, m)
							}
						}
					}
				}
				if len(checks) == 0 && t.Failed {
					r.Errors[host.Host] = t.Error
					continue
				}
				for _, c := range checks {
					h.Checks = append(h.Checks, c)
					if ok, _ := c["ok"].(bool); ok {
						h.Passed++
					} else {
						h.Failed++
					}
				}
			}
		}
	}
	for _, h := range r.Hosts {
		r.Passed += h.Passed
		r.Failed += h.Failed
	}
	return r
}

func writeVerifyText(w io.Writer, r *VerifyReport) {
	names := make([]string, 0, len(r.Hosts))
	for h := range r.Hosts {
		names = append(names, h)
	}
	sort.Strings(names)
	for _, name := range names {
		h := r.Hosts[name]
		fmt.Fprintf(w, "%s: %d passed, %d failed\n", name, h.Passed, h.Failed)
		for _, c := range h.Checks {
			if ok, _ := c["ok"].(bool); !ok {
				fmt.Fprintf(w, "  ✗ %s: %s\n", c["check"], c["detail"])
			}
		}
	}
	errs := make([]string, 0, len(r.Errors))
	for h := range r.Errors {
		errs = append(errs, h)
	}
	sort.Strings(errs)
	for _, h := range errs {
		fmt.Fprintf(w, "%s: could not check: %s\n", h, r.Errors[h])
	}
	switch {
	case len(r.Hosts) == 0 && len(r.Errors) == 0:
		fmt.Fprintf(w, "No verify: checks in %s\n", r.Playbook)
	case r.Failed == 0 && len(r.Errors) == 0:
		fmt.Fprintf(w, "All %d check(s) passed on %d host(s)\n", r.Passed, len(r.Hosts))
	default:
		fmt.Fprintf(w, "%d of %d check(s) failed\n", r.Failed, r.Passed+r.Failed)
	}
}
