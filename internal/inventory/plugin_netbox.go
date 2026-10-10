package inventory

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// NetBox: devices and virtual machines with a primary IP. Configuration:
//
//	plugin: netbox
//	url: https://netbox.example.com
//	token: ${NETBOX_TOKEN}            # or env:NETBOX_TOKEN; default $NETBOX_TOKEN
//	filters: {site: fra1, role: server, tag: managed, status: active, tenant: ops}
//	devices: true                     # default true
//	virtual_machines: true            # default true
//	group_by: [site, role, platform, tenant, tags, type]   # default: site, role, tags
//	ansible_host: primary_ip          # or name
//
// Host variables: netbox_site, netbox_role, netbox_platform, netbox_tenant,
// netbox_tags (list), netbox_status, netbox_primary_ip4, netbox_primary_ip6,
// netbox_device_type (devices), netbox_cluster (VMs), netbox_type
// (device|virtual_machine), netbox_id, netbox_serial, netbox_custom_fields.

func loadNetboxInventory(ctx context.Context, cfg map[string]interface{}) ([]byte, error) {
	base := strings.TrimRight(stringArg(cfg, "url", ""), "/")
	if base == "" {
		return nil, fmt.Errorf("netbox: url is required")
	}
	token := secretArg(cfg, "token", "NETBOX_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("netbox: token is required (or $NETBOX_TOKEN)")
	}
	groupBy := listArg(cfg, "group_by")
	if groupBy == nil {
		groupBy = []string{"site", "role", "tags"}
	}
	filters, _ := cfg["filters"].(map[string]interface{})
	query := url.Values{"limit": {"500"}}
	for k, v := range filters {
		query.Set(map[string]string{"site": "site", "role": "role", "tag": "tag", "status": "status", "tenant": "tenant", "platform": "platform"}[k], fmt.Sprint(v))
	}
	if _, ok := filters["status"]; !ok {
		query.Set("status", "active")
	}
	inv := newAnsibleInventory()
	client := &http.Client{Timeout: 60 * time.Second}
	kinds := []struct{ path, kind string }{}
	if boolArg(cfg, "devices", true) {
		kinds = append(kinds, struct{ path, kind string }{"/api/dcim/devices/", "device"})
	}
	if boolArg(cfg, "virtual_machines", true) {
		kinds = append(kinds, struct{ path, kind string }{"/api/virtualization/virtual-machines/", "virtual_machine"})
	}
	for _, k := range kinds {
		next := base + k.path + "?" + query.Encode()
		for next != "" {
			page, err := netboxGet(ctx, client, next, token)
			if err != nil {
				return nil, fmt.Errorf("netbox: %s: %w", k.kind, err)
			}
			for _, r := range page.Results {
				netboxAddHost(inv, r, k.kind, groupBy, stringArg(cfg, "ansible_host", "primary_ip"))
			}
			next = page.Next
		}
	}
	return inv.bytes()
}

type netboxPage struct {
	Next    string            `json:"next"`
	Results []json.RawMessage `json:"results"`
}

func netboxGet(ctx context.Context, client *http.Client, u, token string) (*netboxPage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Token "+token)
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var page netboxPage
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return nil, err
	}
	return &page, nil
}

// netboxName reads the name of a nested object ({"name": ..}) or a tag list
func netboxName(v interface{}) string {
	if m, ok := v.(map[string]interface{}); ok {
		for _, k := range []string{"slug", "name", "model", "display"} {
			if s, ok := m[k].(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}

func netboxAddHost(inv *ansibleInventory, raw json.RawMessage, kind string, groupBy []string, hostFrom string) {
	var r map[string]interface{}
	if json.Unmarshal(raw, &r) != nil {
		return
	}
	name, _ := r["name"].(string)
	if name == "" {
		return
	}
	ip4, ip6 := "", ""
	if m, ok := r["primary_ip4"].(map[string]interface{}); ok {
		ip4, _ = m["address"].(string)
	}
	if m, ok := r["primary_ip6"].(map[string]interface{}); ok {
		ip6, _ = m["address"].(string)
	}
	strip := func(s string) string { return strings.SplitN(s, "/", 2)[0] }
	vars := map[string]interface{}{
		"netbox_type": kind, "netbox_id": r["id"], "netbox_site": netboxName(r["site"]), "netbox_role": netboxName(r["role"]),
		"netbox_platform": netboxName(r["platform"]), "netbox_tenant": netboxName(r["tenant"]), "netbox_status": netboxName(r["status"]),
		"netbox_primary_ip4": strip(ip4), "netbox_primary_ip6": strip(ip6), "netbox_custom_fields": r["custom_fields"],
	}
	if _, ok := r["status"].(map[string]interface{}); ok {
		vars["netbox_status"], _ = r["status"].(map[string]interface{})["value"].(string)
	}
	if kind == "device" {
		vars["netbox_device_type"] = netboxName(r["device_type"])
		vars["netbox_serial"], _ = r["serial"].(string)
	} else {
		vars["netbox_cluster"] = netboxName(r["cluster"])
	}
	var tags []string
	if list, ok := r["tags"].([]interface{}); ok {
		for _, t := range list {
			if s := netboxName(t); s != "" {
				tags = append(tags, s)
			}
		}
	}
	vars["netbox_tags"] = tags
	switch {
	case hostFrom == "name":
	case ip4 != "":
		vars["ansible_host"] = strip(ip4)
	case ip6 != "":
		vars["ansible_host"] = strip(ip6)
	}
	var groups []string
	for _, g := range groupBy {
		switch g {
		case "tags":
			for _, t := range tags {
				groups = append(groups, "tag_"+t)
			}
		case "type":
			groups = append(groups, kind)
		default:
			if s, ok := vars["netbox_"+g].(string); ok && s != "" {
				groups = append(groups, g+"_"+s)
			}
		}
	}
	inv.addHost(name, vars, groups...)
}
