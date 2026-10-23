package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func loadTokenConfig(t *testing.T, auth string) *ServeConfig {
	t.Helper()
	cfg, err := loadTokenConfigError(t, auth)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func loadTokenConfigError(t *testing.T, auth string) (*ServeConfig, error) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "serve.yml")
	data := "data_dir: " + filepath.Join(dir, "data") + "\nauth:\n" + auth + "\njobs:\n  - {name: a, playbook: a.yml}\n  - {name: b, playbook: b.yml}\n"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	return loadServeConfig(path)
}

func tokenHTTP(s *server, method, path, token string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, nil)
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	s.mux().ServeHTTP(w, r)
	return w
}

func TestServeTokenConfigValidation(t *testing.T) {
	for _, tt := range []struct{ auth, want string }{
		{"  token: old\n  tokens: [{name: ToKeN, token: new}]", `auth.token and auth.tokens both define the name "token"`},
		{"  tokens: [{name: ci, token: x, role: admin}]", `auth.tokens: "ci": role must be operator or viewer`},
		{"  tokens: [{name: ci, token: x, jobs: [missing]}]", "unknown job"},
		{"  tokens: [{name: ci, token: x, expires: tomorrow}]", "invalid expires"},
		{"  tokens: [{name: ci, token: 'sha256:abcd'}]", "64 hex digits"},
		{"  tokens: [{name: ci, token: 'sha256:" + strings.Repeat("g", 64) + "'}]", "64 hex digits"},
	} {
		t.Run(tt.want+tt.auth, func(t *testing.T) {
			_, err := loadTokenConfigError(t, tt.auth)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("got %v, want %s", err, tt.want)
			}
		})
	}
	t.Setenv("SERVE_TEST_TOKEN", "literal${UNEXPANDED}")
	cfg := loadTokenConfig(t, "  token: ${SERVE_TEST_TOKEN}")
	if len(cfg.Auth.allTokens()) != 1 || cfg.Auth.Tokens[0].Token != "literal${UNEXPANDED}" {
		t.Fatalf("legacy folding: %+v", cfg.Auth)
	}
	s, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	w := tokenHTTP(s, "GET", "/api/me", "literal${UNEXPANDED}")
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"user":"token"`) || !strings.Contains(w.Body.String(), `"role":"operator"`) {
		t.Fatal(w.Code, w.Body.String())
	}
}

func TestServeTokenAccess(t *testing.T) {
	cfg := loadTokenConfig(t, "  tokens:\n    - {name: viewer, token: view, role: viewer}\n    - {name: ci, token: scoped, jobs: [a], groups: [team]}\n")
	cfg.Jobs[0].Operators = []string{"TOKEN:CI"}
	s, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		method, path, token string
		code                int
	}{
		{"GET", "/api/jobs", "view", 200},
		{"POST", "/api/jobs/a/run", "view", 403},
		{"POST", "/api/jobs/b/run", "scoped", 403},
		{"POST", "/api/pause", "scoped", 403},
		{"POST", "/api/resume", "scoped", 403},
		{"GET", "/api/pause", "scoped", 200},
		{"POST", "/api/inventory/refresh", "scoped", 403},
		{"GET", "/api/jobs/b/inventory", "scoped", 403},
		{"POST", "/api/jobs/a/run", "scoped", 202},
	} {
		w := tokenHTTP(s, tt.method, tt.path, tt.token)
		if w.Code != tt.code {
			t.Fatalf("%+v: %d %s", tt, w.Code, w.Body.String())
		}
	}
	if s.allowed(serveIdentity{User: "ci", Kind: "proxy", Operator: true}, &cfg.Jobs[0]) {
		t.Fatal("proxy matched token:ci")
	}
	w := tokenHTTP(s, "GET", "/api/me", "scoped")
	var me struct {
		Auth, Role   string
		Jobs, Groups []string
		Expires      *time.Time
	}
	if err := json.Unmarshal(w.Body.Bytes(), &me); err != nil {
		t.Fatal(err)
	}
	if me.Auth != "token" || me.Role != "operator" || len(me.Jobs) != 1 || me.Jobs[0] != "a" || me.Expires != nil {
		t.Fatalf("me: %s", w.Body.String())
	}
	actions := serveAuditEntries(s, 100)
	refused := 0
	for _, a := range actions {
		if a.Reason == "operators only" && !a.OK {
			refused++
		}
	}
	if refused != 4 {
		t.Fatalf("operator refusals: %d; %+v", refused, actions)
	}
}

func TestServeTokenHashExpiry(t *testing.T) {
	digest := sha256.Sum256([]byte("secret"))
	for _, expiry := range []string{"2027-01-01", "2027-01-01T12:34:56+02:00"} {
		t.Run(expiry, func(t *testing.T) {
			cfg := loadTokenConfig(t, fmt.Sprintf("  tokens: [{name: ci, token: 'sha256:%X', expires: '%s'}]", digest, expiry))
			s, err := newServer(cfg, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			end, err := parseServeTokenExpiry(expiry)
			if err != nil {
				t.Fatal(err)
			}
			s.now = func() time.Time { return end.Add(-time.Second) }
			w := tokenHTTP(s, "GET", "/api/me", "secret")
			if w.Code != 200 || !strings.Contains(w.Body.String(), `"expires":`) {
				t.Fatal(w.Code, w.Body.String())
			}
			var me struct{ Expires time.Time }
			if err := json.Unmarshal(w.Body.Bytes(), &me); err != nil || !me.Expires.Equal(end) {
				t.Fatalf("expiry: %s %v", w.Body.String(), err)
			}
			if w := tokenHTTP(s, "GET", "/api/jobs", "wrong"); w.Code != 401 {
				t.Fatal(w.Code)
			}
			metric := tokenHTTP(s, "GET", "/metrics", "")
			want := fmt.Sprintf("onigirazu_serve_token_expiry_timestamp_seconds{name=\"ci\"} %g", float64(end.Unix()))
			if !strings.Contains(metric.Body.String(), want) {
				t.Fatalf("missing %s in %s", want, metric.Body.String())
			}
			for _, at := range []time.Time{end, end.Add(time.Second)} {
				s.now = func() time.Time { return at }
				if w := tokenHTTP(s, "GET", "/api/jobs", "secret"); w.Code != 401 {
					t.Fatal(w.Code)
				}
			}
		})
	}
}

func TestServeTokenGenerator(t *testing.T) {
	now := func() time.Time { return time.Date(2026, 10, 23, 15, 0, 0, 0, time.UTC) }
	for _, expiry := range []string{"90d", "never", "2027-02-03", "2027-02-03T12:00:00+02:00"} {
		t.Run(expiry, func(t *testing.T) {
			value, snippet, err := generateServeToken(serveTokenOptions{name: "ci", role: "viewer", jobs: []string{"a", "b"}, groups: []string{"team", "yes", "a: b"}, expires: expiry}, bytes.NewReader(make([]byte, 32)), now)
			if err != nil {
				t.Fatal(err)
			}
			if !regexp.MustCompile(`^ozs_[A-Za-z0-9_-]{43}$`).MatchString(value) {
				t.Fatal(value)
			}
			cfg := loadTokenConfig(t, "  tokens:\n"+"  "+strings.ReplaceAll(strings.TrimSuffix(snippet, "\n"), "\n", "\n  "))
			tok := cfg.Auth.Tokens[0]
			if tok.Name != "ci" || tok.Role != "viewer" || len(tok.Jobs) != 2 || len(tok.Groups) != 3 || tok.Groups[2] != "a: b" {
				t.Fatalf("round trip: %+v", tok)
			}
			expected := expiry
			if expiry == "90d" {
				expected = "2027-01-21"
			}
			if expiry == "never" {
				expected = ""
				if strings.Contains(snippet, "expires:") {
					t.Fatal(snippet)
				}
			}
			if tok.Expires != expected {
				t.Fatalf("expires %s, want %s", tok.Expires, expected)
			}
			s, err := newServer(cfg, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			s.now = now
			if w := tokenHTTP(s, "GET", "/api/me", value); w.Code != 200 {
				t.Fatal(w.Code, w.Body.String())
			}
		})
	}
	for _, o := range []serveTokenOptions{{name: "BAD"}, {name: "ci", role: "admin"}, {name: "ci", expires: "bad"}, {name: "ci", expires: "999999999999999999999d"}} {
		if _, _, err := generateServeToken(o, bytes.NewReader(make([]byte, 32)), now); err == nil {
			t.Fatalf("accepted %+v", o)
		}
	}
	if _, _, err := generateServeToken(serveTokenOptions{name: "ci"}, bytes.NewReader(nil), now); err == nil {
		t.Fatal("ignored random failure")
	}
	cmd := newServeCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"token", "new", "--name", "ci", "--expires", "never"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Token for ci (shown once, store it in your secret manager):\nozs_") || !strings.Contains(out.String(), "token: sha256:") {
		t.Fatal(out.String())
	}
}

func TestServeTokenScopedDecisions(t *testing.T) {
	for _, action := range []string{"approve", "reject", "cancel"} {
		t.Run(action, func(t *testing.T) {
			cfg := loadTokenConfig(t, "  tokens: [{name: root, token: root}, {name: ci, token: scoped, jobs: [a]}]")
			for i := range cfg.Jobs {
				if action != "cancel" {
					cfg.Jobs[i].Approval = "required"
				}
			}
			s, err := newServer(cfg, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			for _, job := range []string{"a", "b"} {
				w := tokenHTTP(s, "POST", "/api/jobs/"+job+"/run", "root")
				var result struct {
					Run string `json:"run"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || w.Code != 202 || result.Run == "" {
					t.Fatalf("run: %d %s %v", w.Code, w.Body.String(), err)
				}
				w = tokenHTTP(s, "POST", "/api/runs/"+result.Run+"/"+action, "scoped")
				want := 200
				if job == "b" {
					want = 403
				}
				if w.Code != want {
					t.Fatalf("%s %s: %d %s", job, action, w.Code, w.Body.String())
				}
			}
		})
	}
}

func TestServeTokenOtherIdentities(t *testing.T) {
	for _, tt := range []struct {
		auth                     ServeAuth
		user, groups, kind, role string
	}{
		{kind: "none", role: "operator"},
		{auth: ServeAuth{UserHeader: "X-User", GroupsHeader: "X-Groups", Operators: []string{"ops"}}, user: "ci", kind: "proxy", role: "viewer"},
		{auth: ServeAuth{UserHeader: "X-User", GroupsHeader: "X-Groups", Operators: []string{"ops"}}, user: "ci", groups: "ops", kind: "proxy", role: "operator"},
		{auth: ServeAuth{Token: "legacy"}, kind: "token", role: "operator"},
	} {
		s, err := newServer(&ServeConfig{DataDir: t.TempDir(), Auth: tt.auth}, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		r := httptest.NewRequest("GET", "/api/me", nil)
		r.Header.Set("X-User", tt.user)
		r.Header.Set("X-Groups", tt.groups)
		if tt.auth.Token != "" {
			r.Header.Set("Authorization", "Bearer legacy")
		}
		w := httptest.NewRecorder()
		s.mux().ServeHTTP(w, r)
		var me map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &me); err != nil || w.Code != 200 {
			t.Fatal(w.Code, w.Body.String(), err)
		}
		if me["auth"] != tt.kind || me["role"] != tt.role || me["jobs"] != nil || me["expires"] != nil {
			t.Fatal(me)
		}
	}
}

func TestServeTokenMetricEscaping(t *testing.T) {
	cfg := &ServeConfig{DataDir: t.TempDir(), Auth: ServeAuth{Tokens: []ServeToken{{Name: "a\"b\\c\nd", Token: "x", Expires: "2027-01-01"}}}}
	s, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	w := tokenHTTP(s, "GET", "/metrics", "")
	if !strings.Contains(w.Body.String(), `onigirazu_serve_token_expiry_timestamp_seconds{name="a\"b\\c\nd"}`) {
		t.Fatal(w.Body.String())
	}
}
