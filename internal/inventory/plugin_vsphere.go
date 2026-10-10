package inventory

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// vSphere (vCenter REST API): virtual machines with their guest address.
//
//	plugin: vsphere
//	url: https://vcenter.example.com
//	user: ro@vsphere.local
//	password: ${VSPHERE_PASSWORD}     # default $VSPHERE_PASSWORD
//	insecure: true                    # self-signed certificate
//	folders: [Production, Staging]    # VM folder names; default: all
//	powered_on: true                  # default true: only running VMs
//	group_by: [folder, power_state, guest_family]   # default: folder, power_state
//
// Host variables: vsphere_id (vm-123), vsphere_folder, vsphere_power_state,
// vsphere_cpu_count, vsphere_memory_mib, vsphere_guest_host_name,
// vsphere_guest_family, vsphere_guest_os, vsphere_ip_addresses (list);
// ansible_host is the guest's first non-link-local IPv4 (VMware Tools).

func loadVsphereInventory(ctx context.Context, cfg map[string]interface{}) ([]byte, error) {
	base := strings.TrimRight(stringArg(cfg, "url", ""), "/")
	user := stringArg(cfg, "user", "")
	password := secretArg(cfg, "password", "VSPHERE_PASSWORD")
	if base == "" || user == "" || password == "" {
		return nil, fmt.Errorf("vsphere: url, user and password (or $VSPHERE_PASSWORD) are required")
	}
	groupBy := listArg(cfg, "group_by")
	if groupBy == nil {
		groupBy = []string{"folder", "power_state"}
	}
	c := &restClient{base: base, client: &http.Client{Timeout: 60 * time.Second}}
	if boolArg(cfg, "insecure", false) {
		c.client.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}} // #nosec G402 -- the user asked for it
	}
	// a session: POST /api/session with basic auth answers the token
	var token string
	if err := c.call(ctx, http.MethodPost, "/api/session", map[string]string{"basic": user + ":" + password}, &token); err != nil {
		return nil, fmt.Errorf("vsphere: session: %w", err)
	}
	c.headers = map[string]string{"vmware-api-session-id": token}
	defer func() { _ = c.call(ctx, http.MethodDelete, "/api/session", nil, nil) }()

	// folders by id, so a VM can be listed under its folder's name
	var folders []struct {
		Folder, Name string
	}
	if err := c.call(ctx, http.MethodGet, "/api/vcenter/folder?type=VIRTUAL_MACHINE", nil, &folders); err != nil {
		return nil, fmt.Errorf("vsphere: folders: %w", err)
	}
	wanted := map[string]bool{}
	for _, f := range listArg(cfg, "folders") {
		wanted[f] = true
	}
	inv := newAnsibleInventory()
	for _, f := range folders {
		if len(wanted) > 0 && !wanted[f.Name] {
			continue
		}
		q := url.Values{"folders": {f.Folder}}
		if boolArg(cfg, "powered_on", true) {
			q.Set("power_states", "POWERED_ON")
		}
		var vms []struct {
			ID  string `json:"vm"`
			N   string `json:"name"`
			P   string `json:"power_state"`
			CPU int    `json:"cpu_count"`
			Mem int    `json:"memory_size_MiB"`
		}
		if err := c.call(ctx, http.MethodGet, "/api/vcenter/vm?"+q.Encode(), nil, &vms); err != nil {
			return nil, fmt.Errorf("vsphere: vms of %s: %w", f.Name, err)
		}
		for _, vm := range vms {
			vars := map[string]interface{}{"vsphere_id": vm.ID, "vsphere_folder": f.Name, "vsphere_power_state": vm.P,
				"vsphere_cpu_count": vm.CPU, "vsphere_memory_mib": vm.Mem}
			var ident struct {
				HostName string `json:"host_name"`
				IP       string `json:"ip_address"`
				Family   string `json:"family"`
				FullName struct {
					DefaultMessage string `json:"default_message"`
				} `json:"full_name"`
			}
			if c.call(ctx, http.MethodGet, "/api/vcenter/vm/"+vm.ID+"/guest/identity", nil, &ident) == nil {
				vars["vsphere_guest_host_name"], vars["vsphere_guest_family"], vars["vsphere_guest_os"] = ident.HostName, ident.Family, ident.FullName.DefaultMessage
			}
			var nics []struct {
				IP struct {
					Addresses []struct {
						Address string `json:"ip_address"`
						State   string `json:"state"`
					} `json:"ip_addresses"`
				} `json:"ip"`
			}
			var ips []string
			if c.call(ctx, http.MethodGet, "/api/vcenter/vm/"+vm.ID+"/guest/networking/interfaces", nil, &nics) == nil {
				for _, n := range nics {
					for _, a := range n.IP.Addresses {
						if a.State == "" || a.State == "PREFERRED" {
							ips = append(ips, a.Address)
						}
					}
				}
			}
			vars["vsphere_ip_addresses"] = ips
			if ip := firstUsableIPv4(ips); ip != "" {
				vars["ansible_host"] = ip
			} else if ident.IP != "" {
				vars["ansible_host"] = ident.IP
			}
			var groups []string
			for _, g := range groupBy {
				if s, ok := vars["vsphere_"+g].(string); ok && s != "" {
					groups = append(groups, g+"_"+s)
				}
			}
			inv.addHost(vm.N, vars, groups...)
		}
	}
	return inv.bytes()
}

// firstUsableIPv4 is the first IPv4 that is not loopback, link-local or a
// container bridge of a guest
func firstUsableIPv4(ips []string) string {
	for _, ip := range ips {
		if !strings.Contains(ip, ".") || strings.HasPrefix(ip, "127.") || strings.HasPrefix(ip, "169.254.") || strings.HasPrefix(ip, "172.17.") {
			continue
		}
		return ip
	}
	return ""
}

// restClient is a small JSON client with fixed headers
type restClient struct {
	base    string
	client  *http.Client
	headers map[string]string
}

// call does one request; auth["basic"] = "user:password" sets basic auth;
// the JSON answer (or its "value" wrapper, as older vCenters send) lands in out
func (c *restClient) call(ctx context.Context, method, path string, auth map[string]string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if b := auth["basic"]; b != "" {
		u, p, _ := strings.Cut(b, ":")
		req.SetBasicAuth(u, p)
	}
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
		return fmt.Errorf("%s %s: HTTP %d: %s", method, path, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if out == nil {
		return nil
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var wrapped struct {
		Value json.RawMessage `json:"value"`
	}
	if json.Unmarshal(body, &wrapped) == nil && len(wrapped.Value) > 0 && strings.HasPrefix(strings.TrimSpace(string(body)), `{"value"`) {
		body = wrapped.Value
	}
	return json.Unmarshal(body, out)
}
