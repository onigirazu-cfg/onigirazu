package inventory

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// NetBird: the peers of a management server, reached over the mesh.
//
//	plugin: netbird
//	url: https://api.netbird.io          # or the self-hosted management API
//	token: ${NETBIRD_TOKEN}              # a personal access token; default $NETBIRD_TOKEN
//	connected: true                      # default true: only peers online now
//	groups: [servers, admins]            # only peers in one of these NetBird groups; default all
//	group_by: [groups, os]               # default: groups
//
// Host variables: netbird_id, netbird_ip, netbird_hostname, netbird_dns_label,
// netbird_connected, netbird_os, netbird_version, netbird_groups (list),
// netbird_ssh_enabled, netbird_last_seen; ansible_host is the NetBird address.

func loadNetbirdInventory(ctx context.Context, cfg map[string]interface{}) ([]byte, error) {
	base := strings.TrimRight(stringArg(cfg, "url", "https://api.netbird.io"), "/")
	token := secretArg(cfg, "token", "NETBIRD_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("netbird: token is required (or $NETBIRD_TOKEN)")
	}
	groupBy := listArg(cfg, "group_by")
	if groupBy == nil {
		groupBy = []string{"groups"}
	}
	only := map[string]bool{}
	for _, g := range listArg(cfg, "groups") {
		only[g] = true
	}
	c := &restClient{base: base, client: &http.Client{Timeout: 60 * time.Second}, headers: map[string]string{"Authorization": "Token " + token}}
	var peers []struct {
		Id        string `json:"id"`
		N         string `json:"name"`
		Ip        string `json:"ip"`
		H         string `json:"hostname"`
		D         string `json:"dns_label"`
		O         string `json:"os"`
		V         string `json:"version"`
		L         string `json:"last_seen"`
		Connected bool   `json:"connected"`
		SSH       bool   `json:"ssh_enabled"`
		Groups    []struct {
			Name string `json:"name"`
		} `json:"groups"`
	}
	if err := c.call(ctx, http.MethodGet, "/api/peers", nil, &peers); err != nil {
		return nil, fmt.Errorf("netbird: %w", err)
	}
	inv := newAnsibleInventory()
	for _, p := range peers {
		if !p.Connected && boolArg(cfg, "connected", true) {
			continue
		}
		var names []string
		in := len(only) == 0
		for _, g := range p.Groups {
			names = append(names, g.Name)
			if only[g.Name] {
				in = true
			}
		}
		if !in {
			continue
		}
		name := p.D
		if name == "" {
			name = p.N
		}
		vars := map[string]interface{}{"ansible_host": p.Ip, "netbird_id": p.Id, "netbird_ip": p.Ip, "netbird_hostname": p.H,
			"netbird_dns_label": p.D, "netbird_connected": p.Connected, "netbird_os": p.O, "netbird_version": p.V,
			"netbird_groups": names, "netbird_ssh_enabled": p.SSH, "netbird_last_seen": p.L}
		var groups []string
		for _, g := range groupBy {
			switch g {
			case "groups":
				for _, n := range names {
					groups = append(groups, "netbird_"+n)
				}
			case "os":
				groups = append(groups, "os_"+strings.Fields(p.O + " x")[0])
			}
		}
		inv.addHost(name, vars, groups...)
	}
	return inv.bytes()
}
