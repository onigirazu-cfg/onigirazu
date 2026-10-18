package cli

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/onigirazu-cfg/onigirazu/internal/cache"
	"github.com/onigirazu-cfg/onigirazu/internal/inventory"
	"github.com/onigirazu-cfg/onigirazu/internal/logger"
	"github.com/onigirazu-cfg/onigirazu/internal/parser"
	"github.com/onigirazu-cfg/onigirazu/internal/template"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

const serveInventoryFixture = `all:
  children:
    fleet:
      vars:
        db_password: s3cret2
      children:
        web:
          hosts:
            web1:
              ansible_host: 192.0.2.10
              ansible_password: s3cret
              ansible_ssh_private_key_file: /private/key
`

func TestServeInventory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hosts.yml")
	other := filepath.Join(dir, "other.yml")
	for _, p := range []string{path, other} {
		if err := os.WriteFile(p, []byte(serveInventoryFixture), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	cfg := &ServeConfig{DataDir: filepath.Join(dir, "data"), Retain: 10,
		Auth: ServeAuth{UserHeader: "X-User", GroupsHeader: "X-Groups", Operators: []string{"admins"}},
		Jobs: []ServeJob{{Name: "z", Inventory: path}, {Name: "a", Inventory: path}, {Name: "other", Inventory: other}, {Name: "broken", Inventory: filepath.Join(dir, "missing.yml")}},
	}
	s, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	var loads atomic.Int32
	s.inventoryLoad = func(paths []string) (*types.Inventory, error) {
		loads.Add(1)
		log, _ := logger.NewEnhancedLogger("error", "text", io.Discard)
		p := parser.NewEnhancedParser(template.NewEngine(), log)
		p.PreserveInventoryStructure()
		return inventory.NewMultiSourceLoader(p, log, cache.NewManager(time.Minute), time.Minute).LoadFromMultipleSources(context.Background(), paths)
	}
	mux := s.mux()
	request := func(method, path string, operator bool) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, nil)
		r.Header.Set("X-User", "alice")
		if operator {
			r.Header.Set("X-Groups", "admins")
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}
	get := func(operator bool) []serveInventorySource {
		w := request("GET", "/api/inventory", operator)
		if w.Code != 200 {
			t.Fatalf("GET: %d %s", w.Code, w.Body.String())
		}
		for _, secret := range []string{"s3cret", "s3cret2", "/private/key", `"password":`, `"key_file":`, `"user":`, `"ssh_args":`} {
			if strings.Contains(w.Body.String(), secret) {
				t.Fatalf("leaked %q: %s", secret, w.Body.String())
			}
		}
		if !operator && strings.Contains(w.Body.String(), `"address"`) {
			t.Fatal("viewer address")
		}
		var sources []serveInventorySource
		if err := json.Unmarshal(w.Body.Bytes(), &sources); err != nil {
			t.Fatal(err)
		}
		return sources
	}
	sources := get(false)
	if len(sources) != 3 || loads.Load() != 3 {
		t.Fatalf("sources/loads: %d/%d", len(sources), loads.Load())
	}
	var shared serveInventorySource
	for _, source := range sources {
		if source.Source == path {
			shared = source
		}
		if strings.HasSuffix(source.Source, "missing.yml") && (source.Error == "" || len(source.Groups) != 0 || len(source.Hosts) != 0) {
			t.Fatalf("broken source: %+v", source)
		}
	}
	if !slices.Equal(shared.Jobs, []string{"a", "z"}) {
		t.Fatalf("jobs: %v", shared.Jobs)
	}
	if len(shared.Hosts) != 1 || !slices.Contains(shared.Hosts[0].Vars, "ansible_password") || slices.Contains(shared.Hosts[0].Vars, "_original_port") || !slices.Equal(shared.Hosts[0].Groups, []string{"all", "fleet", "web"}) {
		t.Fatalf("hosts: %+v", shared.Hosts)
	}
	var all, fleet bool
	for _, g := range shared.Groups {
		all = all || g.Name == "all"
		if g.Name == "fleet" {
			fleet = true
			if !slices.Equal(g.Vars, []string{"db_password"}) || !slices.Equal(g.Children, []string{"web"}) || len(g.Hosts) != 0 {
				t.Fatalf("fleet: %+v", g)
			}
		}
	}
	if !all || !fleet {
		t.Fatal("missing groups")
	}
	operators := get(true)
	for _, source := range operators {
		if source.Error == "" && (source.Hosts[0].Address == nil || *source.Hosts[0].Address != "192.0.2.10") {
			t.Fatalf("operator address: %+v", source.Hosts)
		}
	}
	if err := os.WriteFile(path, []byte("all:\n  hosts:\n    replacement: {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	get(false)
	if loads.Load() != 3 {
		t.Fatal("snapshot reloaded")
	}
	if w := request("POST", "/api/inventory/refresh", false); w.Code != 403 {
		t.Fatalf("viewer refresh: %d", w.Code)
	}
	get(false)
	if loads.Load() != 3 {
		t.Fatal("viewer cleared snapshots")
	}
	if w := request("POST", "/api/inventory/refresh", true); w.Code != 200 || strings.TrimSpace(w.Body.String()) != `{"ok":true}` {
		t.Fatalf("refresh: %d %s", w.Code, w.Body.String())
	}
	for _, source := range get(false) {
		if source.Source == path && (len(source.Hosts) != 1 || source.Hosts[0].Name != "replacement") {
			t.Fatalf("stale source: %+v", source)
		}
	}
	if loads.Load() != 6 {
		t.Fatalf("refresh loads: %d", loads.Load())
	}
	actions, _ := s.actions(serveActionFilter{Action: "inventory_refresh", Limit: 10})
	if len(actions) != 2 || !actions[0].OK || actions[1].OK || actions[0].User != "alice" {
		t.Fatalf("audit: %+v", actions)
	}
	s.inventoryMu.Lock()
	for _, snapshot := range s.inventorySnapshots {
		snapshot.mu.Lock()
		snapshot.loadedAt = time.Now().Add(-6 * time.Minute)
		snapshot.mu.Unlock()
	}
	s.inventoryMu.Unlock()
	get(false)
	if loads.Load() != 9 {
		t.Fatalf("expired loads: %d", loads.Load())
	}
}

func TestServeInventoryConcurrent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hosts.yml")
	if err := os.WriteFile(path, []byte(serveInventoryFixture), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := newServer(&ServeConfig{DataDir: filepath.Join(dir, "data"), Retain: 10, Jobs: []ServeJob{{Name: "one", Inventory: path}, {Name: "two", Inventory: path}}}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	var loads atomic.Int32
	s.inventoryLoad = func(paths []string) (*types.Inventory, error) {
		loads.Add(1)
		time.Sleep(10 * time.Millisecond)
		log, _ := logger.NewEnhancedLogger("error", "text", io.Discard)
		p := parser.NewEnhancedParser(template.NewEngine(), log)
		p.PreserveInventoryStructure()
		return inventory.NewMultiSourceLoader(p, log, cache.NewManager(time.Minute), time.Minute).LoadFromMultipleSources(context.Background(), paths)
	}
	mux := s.mux()
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/inventory", nil))
			if w.Code != 200 || !strings.Contains(w.Body.String(), "web1") {
				t.Errorf("GET: %d %s", w.Code, w.Body.String())
			}
		}()
	}
	close(start)
	wg.Wait()
	if loads.Load() != 1 {
		t.Fatalf("concurrent loads: %d", loads.Load())
	}
}

func TestServeInventoryDiscovery(t *testing.T) {
	dir := t.TempDir()
	jobs := make([]ServeJob, 0)
	for _, name := range []string{"one", "two"} {
		folder := filepath.Join(dir, name)
		if err := os.MkdirAll(folder, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(folder, "inventory.yml"), []byte(serveInventoryFixture), 0o600); err != nil {
			t.Fatal(err)
		}
		jobs = append(jobs, ServeJob{Name: name, Playbook: filepath.Join(folder, "site.yml")})
	}
	s, err := newServer(&ServeConfig{DataDir: filepath.Join(dir, "data"), Retain: 10, Jobs: jobs}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	s.inventory = nil // the process's -i, which other tests set, must not decide discovery here
	sources := s.inventoryView(false)
	if len(sources) != 2 || sources[0].Error != "" || sources[1].Error != "" || sources[0].Source == sources[1].Source {
		t.Fatalf("discovery: %+v", sources)
	}
	s.inventory = []string{filepath.Join(dir, "one", "inventory.yml")}
	sources = s.inventoryView(false)
	if len(sources) != 1 || !slices.Equal(sources[0].Jobs, []string{"one", "two"}) {
		t.Fatalf("server sources: %+v", sources)
	}
	if inventorySourceKey([]string{"one", "two"}) != inventorySourceKey([]string{"two", "one"}) || inventorySourceKey([]string{"a,b", "c"}) == inventorySourceKey([]string{"a", "b,c"}) {
		t.Fatal("source keys")
	}
}

func TestServeInventoryDirectAll(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "inventory.yml")
	body := `all:
  hosts:
    shared:
      ansible_ssh_pass: secret-alias
  children:
    web:
      hosts:
        shared: {}
        child: {}
`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := newServer(&ServeConfig{DataDir: filepath.Join(dir, "data"), Retain: 10, Jobs: []ServeJob{{Name: "job", Inventory: path}}}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	sources := s.inventoryView(false)
	if len(sources) != 1 || sources[0].Error != "" {
		t.Fatalf("source: %+v", sources)
	}
	var found bool
	for _, group := range sources[0].Groups {
		if group.Name == "all" {
			found = true
			if !slices.Equal(group.Hosts, []string{"shared"}) || !slices.Equal(group.Children, []string{"web"}) {
				t.Fatalf("direct all: %+v", group)
			}
		}
	}
	if !found {
		t.Fatal("missing all")
	}
	for _, host := range sources[0].Hosts {
		if host.Name == "shared" && !slices.Contains(host.Vars, "ansible_ssh_pass") {
			t.Fatalf("alias lost: %+v", host)
		}
	}
	encoded, err := json.Marshal(sources)
	if err != nil || strings.Contains(string(encoded), "secret-alias") {
		t.Fatalf("secret alias: %s %v", encoded, err)
	}
}

func TestServeInventoryNative(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "inventory.yml")
	body := `hosts:
  - name: standalone
    address: 192.0.2.1
groups:
  web:
    vars:
      group_setting: hidden
    hosts:
      grouped:
        address: 192.0.2.2
        vars:
          host_setting: hidden
`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := newServer(&ServeConfig{DataDir: filepath.Join(dir, "data"), Retain: 10, Jobs: []ServeJob{{Name: "job", Inventory: path}}}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	sources := s.inventoryView(true)
	if len(sources) != 1 || sources[0].Error != "" || len(sources[0].Hosts) != 2 {
		t.Fatalf("native inventory: %+v", sources)
	}
	all := sources[0].Groups[0]
	if all.Name != "all" || !slices.Equal(all.Children, []string{"web"}) || !slices.Equal(all.Hosts, []string{"standalone"}) {
		t.Fatalf("implicit all: %+v", all)
	}
	if !slices.Equal(sources[0].Hosts[0].Groups, []string{"all", "web"}) || !slices.Contains(sources[0].Hosts[0].Vars, "host_setting") {
		t.Fatalf("grouped host: %+v", sources[0].Hosts[0])
	}
}

func TestServeInventorySourceLabel(t *testing.T) {
	paths := []string{"/etc/fleet/hosts.yml", "deploy@10.0.0.5:2222,"}
	if got := inventorySourceLabel(paths, false); got != "/etc/fleet/hosts.yml, inline" {
		t.Fatalf("viewer label: %q", got)
	}
	if got := inventorySourceLabel(paths, true); got != "/etc/fleet/hosts.yml, deploy@10.0.0.5:2222," {
		t.Fatalf("operator label: %q", got)
	}
}

func TestServeInventoryHostVarsFromEveryDefinition(t *testing.T) {
	inv := &types.Inventory{
		Hosts: []types.Host{{Name: "web1", Vars: map[string]interface{}{"top": 1}}},
		Groups: map[string]*types.Group{"web": {Name: "web", Hosts: map[string]*types.Host{
			"web1": {Name: "web1", Vars: map[string]interface{}{"from_group": "x"}},
		}}},
	}
	hosts := inventoryHosts(inv)
	if len(hosts) != 1 || !slices.Equal(inventoryVarNames(hosts[0].Vars), []string{"from_group", "top"}) {
		t.Fatalf("var names: %+v", hosts)
	}
	if _, touched := inv.Hosts[0].Vars["from_group"]; touched {
		t.Fatal("the inventory itself must not change")
	}
}

func TestServeInventorySlowSourceDoesNotBlockOthers(t *testing.T) {
	s, err := newServer(&ServeConfig{DataDir: t.TempDir(), Retain: 10}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	release := make(chan struct{})
	s.inventoryLoad = func(paths []string) (*types.Inventory, error) {
		if paths[0] == "slow" {
			<-release
		}
		return &types.Inventory{Groups: map[string]*types.Group{}}, nil
	}
	go s.inventorySnapshot([]string{"slow"})
	time.Sleep(50 * time.Millisecond)
	done := make(chan struct{})
	go func() {
		s.inventorySnapshot([]string{"fast"})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("a slow source held up another one")
	}
	close(release)
}

func TestServeInventoryProvenance(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hosts.yml")
	groupPath := filepath.Join(dir, "group_vars", "web.yml")
	hostPath := filepath.Join(dir, "host_vars", "web1.yml")
	files := map[string]string{
		path: `all:
  vars:
    inherited: inventory-secret
  children:
    web:
      vars:
        shared: group-secret
      hosts:
        web1:
          ansible_host: 192.0.2.10
          ansible_password: connection-secret
          shared: host-secret
`,
		groupPath: "from_file: group-file-secret\n",
		hostPath:  "host_file: host-file-secret\n",
	}
	for path, body := range files {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	s, err := newServer(&ServeConfig{DataDir: filepath.Join(dir, "data"), Retain: 10,
		Auth: ServeAuth{UserHeader: "X-User", GroupsHeader: "X-Groups", Operators: []string{"admins"}},
		Jobs: []ServeJob{{Name: "job", Inventory: path}, {Name: "inline", Inventory: "deploy@192.0.2.8:2222,"}},
	}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	for _, operator := range []bool{true, false} {
		r := httptest.NewRequest("GET", "/api/inventory", nil)
		r.Header.Set("X-User", "alice")
		if operator {
			r.Header.Set("X-Groups", "admins")
		}
		w := httptest.NewRecorder()
		s.mux().ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatalf("GET: %d %s", w.Code, w.Body.String())
		}
		for _, secret := range []string{"inventory-secret", "group-secret", "host-secret", "connection-secret", "group-file-secret", "host-file-secret"} {
			if strings.Contains(w.Body.String(), secret) {
				t.Fatalf("leaked %s", secret)
			}
		}
		if !operator {
			for _, private := range []string{`"effective"`, `"var_sources"`, groupPath, hostPath, "deploy@"} {
				if strings.Contains(w.Body.String(), private) {
					t.Fatalf("viewer leaked %s: %s", private, w.Body.String())
				}
			}
			if !strings.Contains(w.Body.String(), `"source":"inline"`) {
				t.Fatal("viewer inline label missing")
			}
			continue
		}
		var sources []serveInventorySource
		if err := json.Unmarshal(w.Body.Bytes(), &sources); err != nil {
			t.Fatal(err)
		}
		found := false
		for _, source := range sources {
			if source.Source != path {
				continue
			}
			found = true
			if source.Error != "" || len(source.Hosts) != 1 || source.Hosts[0].Effective == nil {
				t.Fatalf("source: %+v", source)
			}
			got := make(map[string]inventory.VariableProvenance)
			for _, v := range *source.Hosts[0].Effective {
				got[v.Name] = v
			}
			for name, want := range map[string]inventory.VariableProvenance{
				"ansible_host": {Name: "ansible_host", From: "host", Source: path},
				"shared":       {Name: "shared", From: "host", Source: path},
				"inherited":    {Name: "inherited", From: "group:all", Source: path},
				"from_file":    {Name: "from_file", From: "group:web", Source: groupPath},
				"host_file":    {Name: "host_file", From: "host", Source: hostPath},
			} {
				if got[name] != want {
					t.Errorf("%s: %+v, want %+v", name, got[name], want)
				}
			}
			if _, ok := got["ansible_password"]; ok {
				t.Fatal("password moved out of execution vars")
			}
			for _, g := range source.Groups {
				if g.VarSources == nil {
					t.Fatalf("group missing var_sources: %+v", g)
				}
				if g.Name == "web" && (*g.VarSources)["from_file"] != groupPath {
					t.Fatalf("group sources: %+v", g.VarSources)
				}
			}
		}
		if !found {
			t.Fatal("inventory source missing")
		}
	}
}
