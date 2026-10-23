package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func storedTokenHTTP(s *server, method, path, token, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	s.mux().ServeHTTP(w, r)
	return w
}
func createStoredToken(t *testing.T, s *server, caller, name, role, jobs string) string {
	t.Helper()
	w := storedTokenHTTP(s, "POST", "/api/tokens", caller, fmt.Sprintf(`{"name":%q,"role":%q,"jobs":%s}`, name, role, jobs))
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var result struct{ Token string }
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || result.Token == "" {
		t.Fatal(w.Body.String(), err)
	}
	return result.Token
}
func TestServeTokensStoreAndLock(t *testing.T) {
	cfg := &ServeConfig{DataDir: t.TempDir()}
	value, record, err := makeStoredServeToken(serveTokenOptions{name: "ci", groups: []string{"ops"}}, "cli:test", time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = addServeToken(cfg, record); err != nil {
		t.Fatal(err)
	}
	tokens, err := readServeTokens(cfg.DataDir)
	if err != nil || len(tokens) != 1 || tokens[0].SHA256 != record.SHA256 || tokens[0].Groups[0] != "ops" {
		t.Fatal(tokens, err)
	}
	path := filepath.Join(cfg.DataDir, "tokens.json")
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), value) {
		t.Fatal("stored secret")
	}
	st, _ := os.Stat(path)
	if st.Mode().Perm() != 0600 {
		t.Fatal(st.Mode())
	}
	unlock, err := lockServeTokens(cfg.DataDir)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, err := addServeToken(cfg, record); done <- err }()
	select {
	case err := <-done:
		t.Fatal("lock did not serialize writer", err)
	case <-time.After(100 * time.Millisecond):
	}
	unlock()
	if err := <-done; err != errServeTokenDuplicate {
		t.Fatal(err)
	}
	lock := filepath.Join(cfg.DataDir, "tokens.lock")
	if err = os.WriteFile(lock, nil, 0600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-31 * time.Second)
	if err = os.Chtimes(lock, old, old); err != nil {
		t.Fatal(err)
	}
	unlock, err = lockServeTokens(cfg.DataDir)
	if err != nil {
		t.Fatal(err)
	}
	unlock()
	if _, err = os.Stat(lock); !os.IsNotExist(err) {
		t.Fatal(err)
	}
}
func TestServeTokensAPI(t *testing.T) {
	cfg := loadTokenConfig(t, "  tokens: [{name: root, token: root}, {name: limited, token: limited, jobs: [a]}, {name: other, token: other, jobs: [a]}, {name: viewer, token: viewer, role: viewer}]")
	s, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	value := createStoredToken(t, s, "limited", "ci", "viewer", `["a"]`)
	if w := tokenHTTP(s, "GET", "/api/me", value); w.Code != 200 || !strings.Contains(w.Body.String(), `"role":"viewer"`) {
		t.Fatal(w.Code, w.Body.String())
	}
	tokens, err := readServeTokens(cfg.DataDir)
	if err != nil {
		t.Fatal(err)
	}
	w := tokenHTTP(s, "GET", "/api/tokens", "viewer")
	raw := w.Body.String()
	if w.Code != 200 || strings.Contains(raw, value) || strings.Contains(raw, tokens[0].SHA256) || strings.Contains(raw, "sha256") {
		t.Fatal(w.Code, raw)
	}
	if !strings.Contains(raw, `"source":"store"`) || !strings.Contains(raw, `"can_create":false`) {
		t.Fatal(raw)
	}
	for _, tt := range []struct {
		caller, body string
		code         int
	}{
		{"viewer", `{"name":"elevated","role":"operator"}`, 403},
		{"limited", `{"name":"broad"}`, 403},
		{"limited", `{"name":"broad","jobs":["b"]}`, 403},
		{"root", `{"name":"bad","jobs":["missing"]}`, 400},
		{"root", `{"name":"bad","groups":[]}`, 400},
		{"root", `{"name":"bad","groups":null}`, 400},
		{"root", `{"name":"ci"}`, 409},
		{"root", `{"name":"root"}`, 409},
		{"root", `{"name":"BAD"}`, 400},
	} {
		w := storedTokenHTTP(s, "POST", "/api/tokens", tt.caller, tt.body)
		if w.Code != tt.code {
			t.Fatalf("%s: %d %s", tt.body, w.Code, w.Body.String())
		}
	}
	for _, tt := range []struct {
		caller, name string
		code         int
	}{{"other", "ci", 403}, {"limited", "ci", 204}, {"root", "root", 409}, {"root", "missing", 404}} {
		w := tokenHTTP(s, "DELETE", "/api/tokens/"+tt.name, tt.caller)
		if w.Code != tt.code {
			t.Fatal(tt, w.Code, w.Body.String())
		}
	}
	if w := tokenHTTP(s, "GET", "/api/me", value); w.Code != 401 {
		t.Fatal(w.Code, w.Body.String())
	}
	audit, err := os.ReadFile(filepath.Join(cfg.DataDir, "audit.log"))
	if err != nil || !bytes.Contains(audit, []byte(`"action":"token_create"`)) || !bytes.Contains(audit, []byte(`"action":"token_revoke"`)) || !bytes.Contains(audit, []byte(`"ok":false`)) {
		t.Fatal(string(audit), err)
	}
}
func TestServeTokensEnableAuth(t *testing.T) {
	s, err := newServer(&ServeConfig{DataDir: t.TempDir()}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	w := storedTokenHTTP(s, "POST", "/api/tokens", "", `{"name":"first"}`)
	if w.Code != 409 || !strings.Contains(w.Body.String(), "confirm with enable_auth") {
		t.Fatal(w.Code, w.Body.String())
	}
	w = storedTokenHTTP(s, "POST", "/api/tokens", "", `{"name":"first","enable_auth":true}`)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var result struct{ Token string }
	json.Unmarshal(w.Body.Bytes(), &result)
	if w := tokenHTTP(s, "GET", "/api/tokens", result.Token); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := tokenHTTP(s, "GET", "/api/me", ""); w.Code != 401 {
		t.Fatal(w.Code)
	}
}
func TestServeTokensReloadAndUsage(t *testing.T) {
	cfg := loadTokenConfig(t, "  tokens: [{name: root, token: root}]")
	var out bytes.Buffer
	s, err := newServer(cfg, &out)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	s.now = func() time.Time { return now }
	// Establish the poll time before an external writer replaces the store.
	tokenHTTP(s, "GET", "/api/me", "root")
	value, record, err := makeStoredServeToken(serveTokenOptions{name: "external"}, "cli:test", s.now)
	if err != nil {
		t.Fatal(err)
	}
	if err = writeServeTokenJSON(filepath.Join(cfg.DataDir, "tokens.json"), serveTokenStore{Tokens: []storedServeToken{record}}); err != nil {
		t.Fatal(err)
	}
	now = now.Add(3 * time.Second)
	if w := tokenHTTP(s, "GET", "/api/me", value); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	id, err := s.identify(httptest.NewRequest("GET", "/", nil))
	if err == nil {
		t.Fatal(id)
	}
	request := httptest.NewRequest("GET", "/", nil)
	request.Header.Set("Authorization", "Bearer "+value)
	id, err = s.identify(request)
	if err != nil || id.Source != "store" || id.matches([]string{"external"}) || !id.matches([]string{"token:external"}) {
		t.Fatal(id, err)
	}
	// A bad reload keeps the last good set and logs the same error once.
	if err = os.WriteFile(filepath.Join(cfg.DataDir, "tokens.json"), []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		now = now.Add(3 * time.Second)
		if w := tokenHTTP(s, "GET", "/api/me", value); w.Code != 200 {
			t.Fatal(w.Code)
		}
	}
	if strings.Count(out.String(), "serve: tokens:") != 1 {
		t.Fatal(out.String())
	}
	now = now.Add(time.Minute)
	tokenHTTP(s, "GET", "/api/me", value)
	usagePath := filepath.Join(cfg.DataDir, "token-usage.json")
	data, err := os.ReadFile(usagePath)
	if err != nil || !bytes.Contains(data, []byte("external")) || !bytes.Contains(data, []byte("root")) {
		t.Fatal(string(data), err)
	}
	st, _ := os.Stat(usagePath)
	now = now.Add(time.Second)
	tokenHTTP(s, "GET", "/api/me", value)
	st2, _ := os.Stat(usagePath)
	if !st.ModTime().Equal(st2.ModTime()) {
		t.Fatal("usage written too often")
	}
	s.tokens.Lock()
	usage := s.tokenUsageLocked()
	s.tokens.Unlock()
	s.saveTokenUsage(usage)
	data, _ = os.ReadFile(usagePath)
	if !bytes.Contains(data, []byte(now.Format(time.RFC3339Nano))) {
		t.Fatal(string(data))
	}
	// Restore the valid store and check startup usage loading.
	writeServeTokenJSON(filepath.Join(cfg.DataDir, "tokens.json"), serveTokenStore{Tokens: []storedServeToken{record}})
	restarted, err := newServer(cfg, io.Discard)
	if err != nil || restarted.tokens.usage["external"].IsZero() {
		t.Fatal(err)
	}
}
func TestServeTokensCLI(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "serve.yml")
	dataDir := filepath.Join(dir, "data")
	if err := os.WriteFile(path, []byte("data_dir: "+dataDir+"\nauth:\n  tokens: [{name: root, token: secret}]\njobs: [{name: a, playbook: a.yml}]\n"), 0600); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) (string, error) {
		cmd := newServeCmd()
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(io.Discard)
		cmd.SetArgs(append([]string{"token"}, args...))
		err := cmd.Execute()
		return out.String(), err
	}
	out, err := run("new", "-f", path, "--name", "ci", "--groups", "ops", "--jobs", "a")
	if err != nil || !strings.Contains(out, "ozs_") || !strings.Contains(out, "Saved to "+filepath.Join(dataDir, "tokens.json")) {
		t.Fatal(out, err)
	}
	records, err := readServeTokens(dataDir)
	if err != nil || len(records) != 1 || !strings.HasPrefix(records[0].CreatedBy, "cli:") {
		t.Fatal(records, err)
	}
	out, err = run("list", "-f", path)
	if err != nil || !strings.Contains(out, "LAST USED") || !strings.Contains(out, "ci") || !strings.Contains(out, "config") || !strings.Contains(out, "store") {
		t.Fatal(out, err)
	}
	if _, err = run("new", "-f", path, "--name", "root"); err == nil {
		t.Fatal("config duplicate accepted")
	}
	if _, err = run("revoke", "root", "-f", path); err != errServeTokenConfig {
		t.Fatal(err)
	}
	if _, err = run("revoke", "ci", "-f", path); err != nil {
		t.Fatal(err)
	}
	records, err = readServeTokens(dataDir)
	if err != nil || len(records) != 0 {
		t.Fatal(records, err)
	}
	if _, err = run("new", "-f", filepath.Join(dir, "missing"), "--name", "printed", "--print"); err != nil {
		t.Fatal(err)
	}
}

func TestServeTokensReloadMetadataAndExpiry(t *testing.T) {
	cfg := loadTokenConfig(t, "  tokens: [{name: root, token: root}]")
	s, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	s.now = func() time.Time { return now }
	value, record, err := makeStoredServeToken(serveTokenOptions{name: "ci", expires: "never"}, "root", s.now)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(cfg.DataDir, "tokens.json")
	write := func() {
		t.Helper()
		if err := writeServeTokenJSON(path, serveTokenStore{Tokens: []storedServeToken{record}}); err != nil {
			t.Fatal(err)
		}
	}
	write()
	now = now.Add(3 * time.Second)
	if w := tokenHTTP(s, "GET", "/api/me", value); w.Code != 200 {
		t.Fatal(w.Code)
	}
	// Same size, different mtime: changing the hash revokes the old value.
	st, _ := os.Stat(path)
	record.SHA256 = strings.Repeat("0", 64)
	write()
	changed := st.ModTime().Add(time.Second)
	if err := os.Chtimes(path, changed, changed); err != nil {
		t.Fatal(err)
	}
	if w := tokenHTTP(s, "GET", "/api/me", value); w.Code != 200 {
		t.Fatal("poll ran before two seconds", w.Code)
	}
	now = now.Add(3 * time.Second)
	if w := tokenHTTP(s, "GET", "/api/me", value); w.Code != 401 {
		t.Fatal(w.Code)
	}
	// Same mtime, different size: metadata still reloads.
	st, _ = os.Stat(path)
	record.CreatedBy = "a different creator"
	write()
	if err := os.Chtimes(path, st.ModTime(), st.ModTime()); err != nil {
		t.Fatal(err)
	}
	now = now.Add(3 * time.Second)
	w := tokenHTTP(s, "GET", "/api/tokens", "root")
	if !strings.Contains(w.Body.String(), "a different creator") {
		t.Fatal(w.Body.String())
	}
	expired := now.Add(-time.Second)
	record.ExpiresAt = &expired
	write()
	now = now.Add(3 * time.Second)
	w = tokenHTTP(s, "GET", "/api/tokens", "root")
	if !strings.Contains(w.Body.String(), `"expired":true`) {
		t.Fatal(w.Body.String())
	}
	// a store whose tokens all expired keeps authentication on
	cfg.Auth = ServeAuth{}
	if !s.authEnabled() {
		t.Fatal("expired store tokens opened the server")
	}
	if w := tokenHTTP(s, "GET", "/api/me", ""); w.Code != 401 {
		t.Fatal(w.Code)
	}
}

func TestServeTokensJobOperators(t *testing.T) {
	cfg := loadTokenConfig(t, "  tokens: [{name: root, token: root}]")
	s, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	value := createStoredToken(t, s, "root", "ci", "operator", `[]`)
	r := httptest.NewRequest("GET", "/api/me", nil)
	r.Header.Set("Authorization", "Bearer "+value)
	id, err := s.identify(r)
	if err != nil {
		t.Fatal(err)
	}
	job := &ServeJob{Name: "a", Operators: []string{"ci"}}
	if s.allowed(id, job) {
		t.Fatal("store token matched bare name")
	}
	job.Operators = []string{"TOKEN:CI"}
	if !s.allowed(id, job) {
		t.Fatal("store token did not match token:name")
	}
	config := serveIdentity{User: "ci", Kind: "token", Source: "config", Operator: true}
	job.Operators = []string{"ci"}
	if !s.allowed(config, job) {
		t.Fatal("config name no longer matches")
	}
	if _, err := addServeToken(cfg, storedServeToken{Name: "ROOT"}); err != errServeTokenDuplicate {
		t.Fatal("case insensitive duplicate", err)
	}
}

func TestServeTokensFailClosed(t *testing.T) {
	cfg := loadTokenConfig(t, "")
	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(cfg.DataDir, "tokens.json")
	if err := os.WriteFile(path, []byte("{broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := newServer(cfg, io.Discard); err == nil {
		t.Fatal("started with an unreadable tokens.json")
	}
	if err := os.WriteFile(path, []byte(`{"tokens":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if w := tokenHTTP(s, "GET", "/api/jobs", ""); w.Code != 401 {
		t.Fatalf("empty store: %d", w.Code)
	}
}

func TestServeTokensAnonymousRaceAndExpiryClamp(t *testing.T) {
	cfg := loadTokenConfig(t, "")
	s, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	anonymous := serveIdentity{User: "anonymous", Operator: true, Kind: "none", Role: "operator"}
	post := func(id serveIdentity, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		s.tokensHandler(w, httptest.NewRequest("POST", "/api/tokens", strings.NewReader(body)), id)
		return w
	}
	if w := post(anonymous, `{"name":"first","enable_auth":true}`); w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	// a second request identified while auth was still off
	if w := post(anonymous, `{"name":"second","enable_auth":true}`); w.Code != 401 {
		t.Fatal(w.Code, w.Body.String())
	}
	soon := s.now().Add(time.Hour)
	short := serveIdentity{User: "short", Operator: true, Kind: "token", Role: "operator", Expires: &soon}
	w := post(short, `{"name":"successor","expires":"never"}`)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var got struct {
		ExpiresAt *time.Time `json:"expires_at"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || got.ExpiresAt == nil || !got.ExpiresAt.Equal(soon) {
		t.Fatal(w.Body.String())
	}
}

func TestServeTokensDelegation(t *testing.T) {
	cfg := loadTokenConfig(t, "  tokens: [{name: root, token: root}]\n  user_header: X-User\n  groups_header: X-Groups\n  operators: [admin, tier0]")
	cfg.Jobs[0].Operators = []string{"tier0"}
	cfg.Jobs[1].Operators = []string{"admin"}
	cfg.Jobs = append(cfg.Jobs, ServeJob{Name: "c", Playbook: "c.yml"})
	s, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	admin := serveIdentity{User: "alice", Groups: []string{"admin"}, Operator: true, Kind: "proxy", Role: "operator"}
	create := func(body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		s.tokensHandler(w, httptest.NewRequest("POST", "/api/tokens", strings.NewReader(body)), admin)
		return w
	}
	// a job the creator cannot run is refused
	if w := create(`{"name":"bad","jobs":["a"]}`); w.Code != 403 || !strings.Contains(w.Body.String(), `job \"a\"`) {
		t.Fatal(w.Code, w.Body.String())
	}
	// no jobs: everything the creator can run, the restricted ones recorded
	w := create(`{"name":"ci"}`)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var got struct {
		Token     string
		Delegated []string
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || len(got.Delegated) != 1 || got.Delegated[0] != "b" {
		t.Fatal(w.Body.String())
	}
	for _, tt := range []struct {
		job  string
		code int
	}{{"b", 202}, {"c", 202}, {"a", 403}} {
		if rw := tokenHTTP(s, "POST", "/api/jobs/"+tt.job+"/run", got.Token); rw.Code != tt.code {
			t.Fatalf("%s: %d %s", tt.job, rw.Code, rw.Body.String())
		}
	}
	// a token made by that token cannot pick up more than it has
	tok := serveIdentity{User: "ci", Operator: true, Kind: "token", Source: "store", Role: "operator", Delegated: []string{"b"}}
	w = httptest.NewRecorder()
	s.tokensHandler(w, httptest.NewRequest("POST", "/api/tokens", strings.NewReader(`{"name":"child","jobs":["a"]}`)), tok)
	if w.Code != 403 {
		t.Fatal(w.Code, w.Body.String())
	}
}
