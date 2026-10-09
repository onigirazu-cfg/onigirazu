package inventory

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Proxmox VE: QEMU VMs and LXC containers of a cluster.
//
//	plugin: proxmox
//	url: https://pve.example.com:8006
//	user: inventory@pve
//	token_id: onigirazu
//	token_secret: ${PROXMOX_TOKEN_SECRET}   # default $PROXMOX_TOKEN_SECRET
//	insecure: true
//	running: true                     # default true
//	templates: false                  # default false
//	agent: true                       # default true: addresses from the QEMU guest agent
//	group_by: [node, type, status, tags, pool]   # default: node, type, tags
//
// Host variables: proxmox_vmid, proxmox_node, proxmox_type (qemu|lxc),
// proxmox_status, proxmox_tags (list), proxmox_pool, proxmox_ip_addresses
// (list); ansible_host from the guest agent (qemu) or the container's
// interfaces (lxc), else the name.

func loadProxmoxInventory(ctx context.Context, cfg map[string]interface{}) ([]byte, error) {
	base := strings.TrimRight(stringArg(cfg, "url", ""), "/")
	user, tokenID := stringArg(cfg, "user", ""), stringArg(cfg, "token_id", "")
	secret := secretArg(cfg, "token_secret", "PROXMOX_TOKEN_SECRET")
	if base == "" || user == "" || tokenID == "" || secret == "" {
		return nil, fmt.Errorf("proxmox: url, user, token_id and token_secret (or $PROXMOX_TOKEN_SECRET) are required")
	}
	groupBy := listArg(cfg, "group_by")
	if groupBy == nil {
		groupBy = []string{"node", "type", "tags"}
	}
	c := &restClient{base: base, client: &http.Client{Timeout: 60 * time.Second},
		headers: map[string]string{"Authorization": fmt.Sprintf("PVEAPIToken=%s!%s=%s", user, tokenID, secret)}}
	if boolArg(cfg, "insecure", false) {
		c.client.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}} // #nosec G402 -- the user asked for it
	}
	var res struct {
		Data []struct {
			VMID     int    `json:"vmid"`
			Name     string `json:"name"`
			Node     string `json:"node"`
			Type     string `json:"type"`
			Status   string `json:"status"`
			Tags     string `json:"tags"`
			Pool     string `json:"pool"`
			Template int    `json:"template"`
		} `json:"data"`
	}
	if err := c.call(ctx, http.MethodGet, "/api2/json/cluster/resources?type=vm", nil, &res); err != nil {
		return nil, fmt.Errorf("proxmox: %w", err)
	}
	inv := newAnsibleInventory()
	for _, vm := range res.Data {
		if vm.Name == "" || (vm.Template == 1 && !boolArg(cfg, "templates", false)) || (vm.Status != "running" && boolArg(cfg, "running", true)) {
			continue
		}
		var tags []string
		for _, t := range strings.FieldsFunc(vm.Tags, func(r rune) bool { return r == ';' || r == ',' }) {
			if t = strings.TrimSpace(t); t != "" {
				tags = append(tags, t)
			}
		}
		vars := map[string]interface{}{"proxmox_vmid": vm.VMID, "proxmox_node": vm.Node, "proxmox_type": vm.Type,
			"proxmox_status": vm.Status, "proxmox_tags": tags, "proxmox_pool": vm.Pool}
		var ips []string
		if vm.Status == "running" && boolArg(cfg, "agent", true) {
			ips = proxmoxAddresses(ctx, c, vm.Node, vm.Type, vm.VMID)
		}
		vars["proxmox_ip_addresses"] = ips
		if ip := firstUsableIPv4(ips); ip != "" {
			vars["ansible_host"] = ip
		}
		var groups []string
		for _, g := range groupBy {
			switch g {
			case "tags":
				for _, t := range tags {
					groups = append(groups, "tag_"+t)
				}
			default:
				if s, ok := vars["proxmox_"+g].(string); ok && s != "" {
					groups = append(groups, g+"_"+s)
				}
			}
		}
		inv.addHost(vm.Name, vars, groups...)
	}
	return inv.bytes()
}

// proxmoxAddresses asks the QEMU guest agent, or reads a container's
// interfaces; nothing when the agent is not running
func proxmoxAddresses(ctx context.Context, c *restClient, node, kind string, vmid int) []string {
	var ips []string
	if kind == "qemu" {
		var res struct {
			Data struct {
				Result []struct {
					Name  string `json:"name"`
					Addrs []struct {
						Address string `json:"ip-address"`
						Type    string `json:"ip-address-type"`
					} `json:"ip-addresses"`
				} `json:"result"`
			} `json:"data"`
		}
		if c.call(ctx, http.MethodGet, fmt.Sprintf("/api2/json/nodes/%s/qemu/%d/agent/network-get-interfaces", node, vmid), nil, &res) != nil {
			return nil
		}
		for _, n := range res.Data.Result {
			if n.Name == "lo" {
				continue
			}
			for _, a := range n.Addrs {
				ips = append(ips, a.Address)
			}
		}
		return ips
	}
	var res struct {
		Data []struct {
			Name string `json:"name"`
			Inet string `json:"inet"`
		} `json:"data"`
	}
	if c.call(ctx, http.MethodGet, fmt.Sprintf("/api2/json/nodes/%s/lxc/%d/interfaces", node, vmid), nil, &res) != nil {
		return nil
	}
	for _, n := range res.Data {
		if n.Name != "lo" && n.Inet != "" {
			ips = append(ips, strings.SplitN(n.Inet, "/", 2)[0])
		}
	}
	return ips
}
