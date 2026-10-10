package modules

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// MikroTik RouterOS, from the control machine (the host is an entry with
// ansible_connection: local, as Ansible's network devices): routeros_api
// manages the items of a menu path through the REST API (RouterOS 7),
// routeros_command runs CLI commands over SSH, routeros_facts reads
// identity, resources and interfaces. Credentials: ansible_user /
// ansible_password; routeros_api_url (default https://<ansible_host>/rest),
// routeros_api_insecure (self-signed certificate), routeros_api_timeout.

// routerosREST is the REST client of one device
type routerosREST struct {
	base   string
	user   string
	pass   string
	client *http.Client
}

func routerosClient(host types.Host) (*routerosREST, error) {
	address := host.Address
	if address == "" {
		address = host.Name
	}
	base := stringVar(host, "routeros_api_url")
	if base == "" {
		base = "https://" + address + "/rest"
	}
	user := stringVar(host, "routeros_api_user")
	if user == "" {
		user = host.User
	}
	pass := stringVar(host, "routeros_api_password")
	if pass == "" {
		pass = host.Password
	}
	if user == "" {
		return nil, fmt.Errorf("routeros: ansible_user (or routeros_api_user) is required")
	}
	timeout := 30 * time.Second
	if t := stringVar(host, "routeros_api_timeout"); t != "" {
		if d, err := time.ParseDuration(t); err == nil {
			timeout = d
		} else if n, err := strconv.Atoi(t); err == nil {
			timeout = time.Duration(n) * time.Second
		}
	}
	insecure := false
	switch v := host.Vars["routeros_api_insecure"].(type) {
	case bool:
		insecure = v
	case string:
		insecure = v == "true" || v == "yes"
	}
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure, MinVersion: tls.VersionTLS12}} // #nosec G402 -- the device's own certificate, by request
	return &routerosREST{base: strings.TrimRight(base, "/"), user: user, pass: pass, client: &http.Client{Timeout: timeout, Transport: tr}}, nil
}

func stringVar(host types.Host, key string) string {
	if v, ok := host.Vars[key]; ok && v != nil {
		return fmt.Sprint(v)
	}
	return ""
}

// call does one REST request; path is the menu path (ip/firewall/address-list)
func (r *routerosREST) call(ctx context.Context, method, path string, query url.Values, body interface{}) ([]byte, error) {
	u := r.base + "/" + strings.Trim(path, "/")
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	var rd io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		rd = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, rd)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(r.user, r.pass)
	req.Header.Set("Content-Type", "application/json")
	res, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("routeros %s %s: %w", method, path, err)
	}
	defer res.Body.Close()
	out, _ := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if res.StatusCode >= 300 {
		var e struct {
			Message string `json:"message"`
			Detail  string `json:"detail"`
		}
		_ = json.Unmarshal(out, &e)
		msg := strings.TrimSpace(e.Message + " " + e.Detail)
		if msg == "" {
			msg = strings.TrimSpace(string(out))
		}
		return nil, fmt.Errorf("routeros %s %s: %s: %s", method, path, res.Status, msg)
	}
	return out, nil
}

// items are the records of a path, every value a string as RouterOS sends them
func (r *routerosREST) items(ctx context.Context, path string, query url.Values) ([]map[string]string, error) {
	out, err := r.call(ctx, http.MethodGet, path, query, nil)
	if err != nil {
		return nil, err
	}
	var raw []map[string]interface{}
	if err := json.Unmarshal(out, &raw); err != nil {
		// a single object (system/identity) is one item
		var one map[string]interface{}
		if err2 := json.Unmarshal(out, &one); err2 != nil {
			return nil, fmt.Errorf("routeros %s: %w", path, err)
		}
		raw = []map[string]interface{}{one}
	}
	items := make([]map[string]string, 0, len(raw))
	for _, m := range raw {
		item := make(map[string]string, len(m))
		for k, v := range m {
			item[k] = fmt.Sprint(v)
		}
		items = append(items, item)
	}
	return items, nil
}

// stringMap turns a module argument map into strings as RouterOS wants them
func stringMap(v interface{}) map[string]string {
	out := map[string]string{}
	if m, ok := v.(map[string]interface{}); ok {
		for k, x := range m {
			switch b := x.(type) {
			case bool:
				out[k] = map[bool]string{true: "yes", false: "no"}[b]
			default:
				out[k] = fmt.Sprint(x)
			}
		}
	}
	return out
}
