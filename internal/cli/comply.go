package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/onigirazu-cfg/onigirazu/internal/comply"
)

// comply runs a compliance profile — named controls, each a verify check
// with a severity and a fix — against the hosts and scores them: what
// InSpec or OpenSCAP report, from the verify module in one round trip.

type complyOptions struct {
	driftCheckOptions
	profile, failOn          string
	tags                     []string
	minSeverity              string
	metricsFile, metricsPush string
	metricsLabels            []string
}

func newComplyCmd() *cobra.Command {
	o := &complyOptions{}
	cmd := &cobra.Command{
		Use:   "comply",
		Short: "Check hosts against a compliance profile (CIS-style controls) and score them",
		Long: `Run the controls of a profile against the hosts and report, per host, which
pass and which fail, with severity and remediation hints. Profiles are YAML
files of controls built on verify checks; two ship in the binary
(linux-baseline, ssh); "comply list" shows them, "comply show" prints one to
start your own from. Exit code 1 when a control fails (or one at --fail-on
severity or above) or a host cannot be checked.`,
		Example: `  onigirazu comply --profile linux-baseline -i hosts.yml
  onigirazu comply --profile ssh -i hosts.yml --limit web --format markdown
  onigirazu comply --profile ./company.yml -i hosts.yml --fail-on high --metrics-push http://vm:8428/api/v1/import/prometheus
  onigirazu comply show ssh > my-ssh.yml`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if o.profile == "" {
				return fmt.Errorf("--profile is required (comply list shows the bundled ones)")
			}
			return runComply(o, cmd.OutOrStdout())
		},
	}
	f := cmd.Flags()
	f.StringVar(&o.profile, "profile", "", "Profile: a bundled name or a YAML file")
	f.StringVar(&o.format, "format", "text", "Report format (text, json, markdown, html)")
	f.StringVar(&o.output, "output", "", "Write the report to this file")
	f.StringArrayVar(&o.tags, "control-tags", nil, "Only controls with one of these tags (repeatable)")
	f.StringVar(&o.minSeverity, "min-severity", "", "Only controls of this severity or above (low, medium, high, critical)")
	f.StringVar(&o.failOn, "fail-on", "", "Exit 1 only when a failed control is of this severity or above (default: any failure)")
	f.StringArrayVarP(&o.extraVars, "extra-vars", "e", nil, "Extra variables, as for apply (repeatable)")
	f.StringVar(&o.limit, "limit", "", "Check only hosts matching this pattern")
	f.BoolVarP(&o.become, "become", "b", true, "Use privilege escalation (profiles read root-only files)")
	f.StringVar(&o.becomeUser, "become-user", "", "User to become")
	f.StringVarP(&o.user, "user", "u", "", "SSH user for every host")
	f.StringVar(&o.privateKey, "private-key", "", "SSH private key for every host")
	f.StringVar(&o.metricsFile, "metrics-file", "", "Write the result as Prometheus metrics to this file (node_exporter textfile collector)")
	f.StringVar(&o.metricsPush, "metrics-push", "", "POST the metrics to this URL (VictoriaMetrics import, Pushgateway)")
	f.StringArrayVar(&o.metricsLabels, "metrics-label", nil, "Extra label on every metric, name=value (repeatable)")

	list := &cobra.Command{
		Use:   "list",
		Short: "List the bundled profiles",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			profiles, err := comply.Bundled()
			if err != nil {
				return err
			}
			for _, p := range profiles {
				fmt.Fprintf(cmd.OutOrStdout(), "%-16s %3d controls  %s\n", p.Name, len(p.Controls), p.Title)
			}
			return nil
		},
	}
	show := &cobra.Command{
		Use:   "show PROFILE",
		Short: "Print a profile as YAML (a start for your own)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := comply.Load(args[0])
			if err != nil {
				return err
			}
			enc := yaml.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent(2)
			return enc.Encode(p)
		},
	}
	cmd.AddCommand(list, show)
	return cmd
}

func runComply(o *complyOptions, stdout io.Writer) error {
	profile, err := comply.Load(o.profile)
	if err != nil {
		return err
	}
	profile = profile.Filter(o.tags, o.minSeverity)
	if len(profile.Controls) == 0 {
		return fmt.Errorf("no control of %s matches the filters", profile.Name)
	}
	checks, _ := profile.Checks()
	become := o.become
	if profile.Become != nil && !*profile.Become {
		become = false
	}
	// the profile runs as a playbook with one verify-only play
	play := map[string]interface{}{"name": "comply " + profile.Name, "hosts": "all", "gather_facts": false, "become": become, "verify": checks}
	data, err := yaml.Marshal(map[string]interface{}{"plays": []interface{}{play}})
	if err != nil {
		return err
	}
	dir, err := os.MkdirTemp("", "onigirazu-comply-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	playbook := filepath.Join(dir, profile.Name+".yml")
	if err := os.WriteFile(playbook, data, 0o600); err != nil {
		return err
	}
	o.driftCheckOptions.become = false // the play carries become
	if statePath == "" || statePath == ".onigirazu-state" {
		statePath = filepath.Join(dir, ".onigirazu-state")
	}
	result, err := runPlaybook(append(o.applyArgs(playbook, false), "--verify-only"))
	if err != nil {
		return err
	}
	vr := buildVerifyReport(playbook, result)
	hostChecks := map[string][]map[string]interface{}{}
	for host, h := range vr.Hosts {
		hostChecks[host] = h.Checks
	}
	report := comply.Build(profile, hostChecks, vr.Errors)
	report.Threshold = o.failOn

	out := stdout
	if o.output != "" {
		f, err := os.Create(o.output)
		if err != nil {
			return err
		}
		defer f.Close()
		out = f
	}
	switch strings.ToLower(o.format) {
	case "text", "":
		report.WriteText(out)
	case "json":
		if err := report.WriteJSON(out); err != nil {
			return err
		}
	case "markdown", "md":
		report.WriteMarkdown(out)
	case "html":
		report.WriteHTML(out)
	default:
		return fmt.Errorf("unknown format %q (text, json, markdown, html)", o.format)
	}
	if o.metricsFile != "" || o.metricsPush != "" {
		labels, err := parseLabels(o.metricsLabels)
		if err != nil {
			return err
		}
		emitMetrics(report.Metrics(labels), o.metricsFile, o.metricsPush)
	}
	if len(report.Errors) > 0 || report.FailsAt(o.failOn) {
		return &ExitError{Code: 1}
	}
	return nil
}
