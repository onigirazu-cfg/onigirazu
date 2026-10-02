package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/onigirazu-cfg/onigirazu/internal/bridge"
	"github.com/onigirazu-cfg/onigirazu/internal/config"
	"github.com/onigirazu-cfg/onigirazu/internal/modules"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func newDocCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doc MODULE",
		Short: "Show a module's arguments: built-in, or through the Ansible bridge (from ansible-doc)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if cfg, err := config.LoadConfigWithDiscovery(configPath, "."); err == nil {
				bridge.Configure(cfg.AnsibleBridge)
			}
			return showDoc(cmd.OutOrStdout(), args[0])
		},
	}
}

func showDoc(w io.Writer, name string) error {
	short := types.ShortModuleName(name)
	if m, err := modules.NewRegistry().GetModule(short); err == nil {
		fmt.Fprintf(w, "%s (built in): %s\n", short, m.GetDescription())
		if args := moduleArgs[short]; len(args) > 0 {
			sorted := append([]string{}, args...)
			sort.Strings(sorted)
			fmt.Fprintf(w, "arguments: %s\n", strings.Join(sorted, ", "))
		}
		fmt.Fprintln(w, "details: docs/modules/README.md")
		return nil
	}
	spec, err := bridge.SpecFor(name)
	if err != nil {
		return fmt.Errorf("%s is not a built-in module, and ansible-doc cannot describe it: %w", name, err)
	}
	via := "through the Ansible bridge"
	if !bridge.Allowed(name) {
		via = "Ansible only; add it to ansible_bridge.modules in onigirazu.yml to run it"
	}
	fmt.Fprintf(w, "%s (%s): %s\n\n", name, via, bridge.Text(spec.Description))
	names := make([]string, 0, len(spec.Options))
	for n := range spec.Options {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		o := spec.Options[n]
		var attrs []string
		if o.Type != "" {
			t := o.Type
			if o.Elements != "" {
				t += " of " + o.Elements
			}
			attrs = append(attrs, t)
		}
		if o.Required {
			attrs = append(attrs, "required")
		}
		if o.Default != nil {
			attrs = append(attrs, fmt.Sprintf("default %v", o.Default))
		}
		if len(o.Choices) > 0 {
			attrs = append(attrs, fmt.Sprintf("one of %v", o.Choices))
		}
		if len(o.Aliases) > 0 {
			attrs = append(attrs, "aliases "+strings.Join(o.Aliases, ", "))
		}
		desc := bridge.Text(o.Description)
		if len(desc) > 160 {
			desc = desc[:157] + "..."
		}
		fmt.Fprintf(w, "  %-22s %s\n  %-22s %s\n", n, strings.Join(attrs, "; "), "", desc)
	}
	return nil
}
