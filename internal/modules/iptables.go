package modules

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// IptablesModule keeps an iptables rule (Ansible's iptables): table, chain,
// the match options (protocol, source, destination, in/out_interface,
// destination_port, source_port, match, ctstate, comment, ...), jump/goto,
// state present/absent, action insert/append, rule_num, ip_version,
// policy (sets the chain's policy), flush (empties the chain)
type IptablesModule struct{ *BaseModule }

func NewIptablesModule() *IptablesModule {
	return &IptablesModule{BaseModule: NewBaseModule("iptables")}
}

func (m *IptablesModule) GetDescription() string { return "Manage iptables rules" }

// iptablesFlags maps arguments to the rule's flags, in Ansible's order
var iptablesFlags = []struct{ arg, flag string }{
	{"protocol", "-p"}, {"source", "-s"}, {"destination", "-d"}, {"in_interface", "-i"}, {"out_interface", "-o"},
	{"fragment", "-f"}, {"match", "-m"}, {"source_port", "--sport"}, {"destination_port", "--dport"}, {"destination_ports", "--dports"},
	{"to_ports", "--to-ports"}, {"to_destination", "--to-destination"}, {"to_source", "--to-source"}, {"set_dscp_mark", "--set-dscp"},
	{"set_dscp_mark_class", "--set-dscp-class"}, {"ctstate", "--ctstate"}, {"src_range", "--src-range"}, {"dst_range", "--dst-range"},
	{"icmp_type", "--icmp-type"}, {"uid_owner", "--uid-owner"}, {"gid_owner", "--gid-owner"}, {"limit", "--limit"},
	{"limit_burst", "--limit-burst"}, {"reject_with", "--reject-with"}, {"log_prefix", "--log-prefix"}, {"log_level", "--log-level"},
	{"tcp_flags", "--tcp-flags"}, {"syn", "--syn"}, {"set_counters", "--set-counters"}, {"comment", "--comment"},
}

// iptablesRule renders the rule's arguments after the chain
func iptablesRule(args map[string]interface{}) []string {
	var out []string
	add := func(flag string, v interface{}) {
		switch x := v.(type) {
		case bool:
			if x {
				out = append(out, flag)
			}
		case []interface{}:
			parts := make([]string, 0, len(x))
			for _, p := range x {
				parts = append(parts, fmt.Sprint(p))
			}
			out = append(out, flag, strings.Join(parts, ","))
		case nil:
		default:
			s := fmt.Sprint(x)
			if strings.HasPrefix(s, "!") {
				out = append(out, "!", flag, strings.TrimSpace(s[1:]))
			} else {
				out = append(out, flag, s)
			}
		}
	}
	for _, f := range iptablesFlags {
		if v, set := args[f.arg]; set {
			if f.arg == "comment" {
				out = append(out, "-m", "comment")
			}
			if f.arg == "ctstate" || f.arg == "limit" || f.arg == "uid_owner" || f.arg == "gid_owner" || f.arg == "destination_ports" {
				m := map[string]string{"ctstate": "conntrack", "limit": "limit", "uid_owner": "owner", "gid_owner": "owner", "destination_ports": "multiport"}[f.arg]
				if _, has := args["match"]; !has {
					out = append(out, "-m", m)
				}
			}
			add(f.flag, v)
		}
	}
	if j := getStringArg(args, "jump", ""); j != "" {
		out = append(out, "-j", j)
	}
	if g := getStringArg(args, "goto", ""); g != "" {
		out = append(out, "-g", g)
	}
	return out
}

func (m *IptablesModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	start := time.Now()
	result := types.TaskResult{TaskName: taskName(args), Host: host.Name, Module: m.name, Timestamp: start, Success: true, Output: map[string]interface{}{}}
	fail := func(msg string) (types.TaskResult, error) {
		result.Success, result.Error, result.Duration = false, msg, time.Since(start)
		return result, nil
	}
	bin := "iptables"
	if getStringArg(args, "ip_version", "ipv4") == "ipv6" {
		bin = "ip6tables"
	}
	table := getStringArg(args, "table", "filter")
	chain := getStringArg(args, "chain", "")
	base := bin + " -w 10 -t " + shellQuote(table)
	quoteAll := func(parts []string) string {
		q := make([]string, len(parts))
		for i, p := range parts {
			q[i] = shellQuote(p)
		}
		return strings.Join(q, " ")
	}
	switch {
	case getBoolArg(args, "flush", false):
		if chain == "" {
			return fail("iptables: flush needs a chain")
		}
		result.Changed = true
		if !inCheckMode(args) {
			if _, err := runShellOnHost(ctx, host, args, base+" -F "+shellQuote(chain)); err != nil {
				return fail("iptables: " + err.Error())
			}
		}
	case getStringArg(args, "policy", "") != "":
		policy := getStringArg(args, "policy", "")
		out, _ := runShellOnHost(ctx, host, args, base+" -S "+shellQuote(chain)+" 2>/dev/null | awk '/^-P/{print $3}'")
		if strings.TrimSpace(out) == policy {
			break
		}
		result.Changed = true
		if !inCheckMode(args) {
			if _, err := runShellOnHost(ctx, host, args, base+" -P "+shellQuote(chain)+" "+shellQuote(policy)); err != nil {
				return fail("iptables: " + err.Error())
			}
		}
	default:
		if chain == "" {
			return fail("iptables requires 'chain'")
		}
		rule := quoteAll(iptablesRule(args))
		_, err := runShellOnHost(ctx, host, args, base+" -C "+shellQuote(chain)+" "+rule+" 2>/dev/null")
		present := err == nil
		want := getStringArg(args, "state", "present") == "present"
		result.Output["rule"] = rule
		if present == want {
			break
		}
		result.Changed = true
		if inCheckMode(args) {
			break
		}
		var cmd string
		switch {
		case !want:
			cmd = base + " -D " + shellQuote(chain) + " " + rule
		case getStringArg(args, "action", "append") == "insert" && getIntArg(args, "rule_num", 0) > 0:
			cmd = base + " -I " + shellQuote(chain) + " " + itoa(getIntArg(args, "rule_num", 0)) + " " + rule
		case getStringArg(args, "action", "append") == "insert":
			cmd = base + " -I " + shellQuote(chain) + " " + rule
		default:
			cmd = base + " -A " + shellQuote(chain) + " " + rule
		}
		if out, err := runShellOnHost(ctx, host, args, cmd+" 2>&1"); err != nil {
			return fail(fmt.Sprintf("iptables: %v: %s", err, strings.TrimSpace(out)))
		}
	}
	result.Duration = time.Since(start)
	return result, nil
}
