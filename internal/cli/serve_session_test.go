package cli

import (
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func sessionHTTP(s *server, method, path, body string, cookie *http.Cookie, headers map[string]string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if cookie != nil {
		r.AddCookie(cookie)
	}
	for key, value := range headers {
		r.Header.Set(key, value)
	}
	w := httptest.NewRecorder()
	s.mux().ServeHTTP(w, r)
	return w
}

func TestServeSessionSignIn(t *testing.T) {
	cfg := loadTokenConfig(t, "  tokens: [{name: ci, token: secret, role: viewer, jobs: [a], expires: '2027-01-01'}]")
	s, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 12, 31, 12, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	for _, remember := range []bool{false, true} {
		w := sessionHTTP(s, "POST", "/api/session", fmt.Sprintf(`{"token":"secret","remember":%t}`, remember), nil, nil)
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
		cookies := w.Result().Cookies()
		if len(cookies) != 1 {
			t.Fatal(cookies)
		}
		cookie := cookies[0]
		if cookie.Name != serveSessionCookie || cookie.Value != "secret" || cookie.Path != "/" || !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode || cookie.Secure || !cookie.Expires.IsZero() {
			t.Fatalf("cookie: %+v", cookie)
		}
		want := 0
		if remember {
			want = 12 * 60 * 60
		}
		if cookie.MaxAge != want {
			t.Fatalf("MaxAge = %d, want %d", cookie.MaxAge, want)
		}
		if !remember && strings.Contains(w.Header().Get("Set-Cookie"), "Max-Age") {
			t.Fatal(w.Header())
		}
		var signedIn, me map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &signedIn); err != nil {
			t.Fatal(err)
		}
		meResponse := sessionHTTP(s, "GET", "/api/me", "", cookie, nil)
		if err := json.Unmarshal(meResponse.Body.Bytes(), &me); err != nil {
			t.Fatal(err)
		}
		if meResponse.Code != 200 || fmt.Sprint(signedIn) != fmt.Sprint(me) || me["via"] != "cookie" {
			t.Fatal(signedIn, me)
		}
		if w := sessionHTTP(s, "GET", "/api/jobs", "", cookie, nil); w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	actions := serveAuditEntries(s, 100)
	if len(actions) != 2 || !actions[0].OK || actions[0].Action != "sign_in" || actions[0].User != "ci" {
		t.Fatal(actions)
	}
	now = now.Add(12 * time.Hour)
	if w := sessionHTTP(s, "POST", "/api/session", `{"token":"secret"}`, nil, nil); w.Code != 401 || !strings.Contains(w.Body.String(), "unknown or expired token") {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := sessionHTTP(s, "GET", "/api/jobs", "", &http.Cookie{Name: serveSessionCookie, Value: "secret"}, nil); w.Code != 401 {
		t.Fatal(w.Code)
	}
}

func TestServeSessionRememberAndSecure(t *testing.T) {
	s, err := newServer(&ServeConfig{DataDir: t.TempDir(), Auth: ServeAuth{Token: "secret", TrustedProxies: []string{"10.0.0.0/8"}}}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	s.cfg.Auth.trusted, err = parseTrustedProxies(s.cfg.Auth.TrustedProxies)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, peer, proto string
		tls, secure       bool
	}{
		{name: "plain", peer: "10.0.0.1:123"},
		{name: "TLS", peer: "192.0.2.1:123", tls: true, secure: true},
		{name: "trusted", peer: "10.0.0.1:123", proto: "https", secure: true},
		{name: "untrusted", peer: "192.0.2.1:123", proto: "https"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/api/session", strings.NewReader(`{"token":"secret","remember":true}`))
			r.RemoteAddr = tc.peer
			r.Header.Set("X-Forwarded-Proto", tc.proto)
			if tc.tls {
				r.TLS = &tls.ConnectionState{}
			}
			w := httptest.NewRecorder()
			s.mux().ServeHTTP(w, r)
			if w.Code != 200 {
				t.Fatal(w.Code, w.Body.String())
			}
			cookie := w.Result().Cookies()[0]
			if cookie.Secure != tc.secure || cookie.MaxAge != 30*24*60*60 {
				t.Fatalf("cookie: %+v", cookie)
			}
			r.Method = "DELETE"
			r.AddCookie(cookie)
			w = httptest.NewRecorder()
			s.mux().ServeHTTP(w, r)
			if w.Code != 204 {
				t.Fatal(w.Code, w.Body.String())
			}
			cleared := w.Result().Cookies()[0]
			if cleared.MaxAge != -1 || cleared.Value != "" || cleared.Secure != cookie.Secure || cleared.Path != "/" || !cleared.HttpOnly || cleared.SameSite != http.SameSiteStrictMode || !strings.Contains(w.Header().Get("Set-Cookie"), "Max-Age=0") {
				t.Fatalf("cleared: %+v", cleared)
			}
		})
	}
	actions := serveAuditEntries(s, 100)
	count := 0
	for _, a := range actions {
		if a.Action == "sign_out" && a.OK && a.User == "token" {
			count++
		}
	}
	if count != 4 {
		t.Fatal(actions)
	}
	if w := sessionHTTP(s, "DELETE", "/api/session", "", nil, nil); w.Code != 204 {
		t.Fatal(w.Code)
	}
}

func TestServeSessionRateLimit(t *testing.T) {
	s, err := newServer(&ServeConfig{DataDir: t.TempDir(), Auth: ServeAuth{Token: "secret", UserHeader: "X-User"}}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	s.now = func() time.Time { return now }
	for i := 0; i < 10; i++ {
		w := sessionHTTP(s, "POST", "/api/session", `{"token":"do-not-log"}`, nil, map[string]string{"X-User": "proxy-user"})
		if w.Code != 401 {
			t.Fatal(i, w.Code, w.Body.String())
		}
	}
	w := sessionHTTP(s, "POST", "/api/session", `{"token":"secret"}`, nil, nil)
	if w.Code != 429 || !strings.Contains(w.Body.String(), "too many attempts, try again in a minute") {
		t.Fatal(w.Code, w.Body.String())
	}
	actions := serveAuditEntries(s, 100)
	if len(actions) != 11 {
		t.Fatal(actions)
	}
	for _, a := range actions {
		if a.Action != "sign_in" || a.OK || a.Addr != "192.0.2.1" || a.User != "" {
			t.Fatal(a)
		}
	}
	data, err := os.ReadFile(filepath.Join(s.cfg.DataDir, "audit.log"))
	if err != nil || strings.Contains(string(data), "do-not-log") {
		t.Fatal(string(data), err)
	}
	r := httptest.NewRequest("POST", "/api/session", strings.NewReader(`{"token":"secret"}`))
	r.RemoteAddr = "192.0.2.2:123"
	w = httptest.NewRecorder()
	s.mux().ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	now = now.Add(61 * time.Second)
	if w := sessionHTTP(s, "POST", "/api/session", `{"token":"secret"}`, nil, map[string]string{"X-User": "proxy-user"}); w.Code != 200 || !strings.Contains(w.Body.String(), `"auth":"token"`) {
		t.Fatal(w.Code, w.Body.String())
	}
	if len(s.sessions.failures) != 0 {
		t.Fatal(s.sessions.failures)
	}
}

func TestServeSessionConcurrentFailures(t *testing.T) {
	s, err := newServer(&ServeConfig{DataDir: t.TempDir(), Auth: ServeAuth{Token: "secret"}}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	codes := make(chan int, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			codes <- sessionHTTP(s, "POST", "/api/session", `{"token":"wrong"}`, nil, nil).Code
		}()
	}
	wg.Wait()
	close(codes)
	counts := map[int]int{}
	for code := range codes {
		counts[code]++
	}
	if counts[401] != 10 || counts[429] != 10 {
		t.Fatal(counts)
	}
}

func TestServeSessionCSRFAndPrecedence(t *testing.T) {
	cfg := loadTokenConfig(t, "  user_header: X-User\n  tokens: [{name: root, token: root}, {name: viewer, token: view, role: viewer}]")
	s, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	cookie := &http.Cookie{Name: serveSessionCookie, Value: "root"}
	for _, tc := range []struct {
		name    string
		headers map[string]string
		code    int
	}{
		{"cross-site", map[string]string{"Sec-Fetch-Site": "cross-site"}, 403},
		{"same-site", map[string]string{"Sec-Fetch-Site": "same-site"}, 403},
		{"origin", map[string]string{"Origin": "https://other.test"}, 403},
		{"null origin", map[string]string{"Origin": "null"}, 403},
		{"empty origin", map[string]string{"Origin": ""}, 403},
		{"same-origin", map[string]string{"Origin": "http://example.com", "Sec-Fetch-Site": "same-origin"}, 200},
		{"absent", nil, 403},
		{"origin only", map[string]string{"Origin": "http://example.com"}, 200},
		{"bearer", map[string]string{"Authorization": "Bearer root", "Sec-Fetch-Site": "cross-site", "Origin": "https://other.test"}, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := sessionHTTP(s, "POST", "/api/pause", `{}`, cookie, tc.headers)
			if w.Code != tc.code {
				t.Fatal(w.Code, w.Body.String())
			}
			if tc.code == 403 && !strings.Contains(w.Body.String(), "cross-site request") {
				t.Fatal(w.Body.String())
			}
		})
	}
	if w := sessionHTTP(s, "DELETE", "/api/session", "", cookie, map[string]string{"Sec-Fetch-Site": "cross-site"}); w.Code != 403 || len(w.Result().Cookies()) != 0 {
		t.Fatal(w.Code, w.Header())
	}
	for _, tc := range []struct{ header, value, user, via string }{
		{"root", "view", "root", "header"},
		{"wrong", "root", "root", "cookie"},
		{"", "wrong", "proxy-user", ""},
	} {
		w := sessionHTTP(s, "GET", "/api/me", "", &http.Cookie{Name: serveSessionCookie, Value: tc.value}, map[string]string{"Authorization": "Bearer " + tc.header, "X-User": "proxy-user"})
		var me struct{ User, Via string }
		if err := json.Unmarshal(w.Body.Bytes(), &me); err != nil || w.Code != 200 || me.User != tc.user || me.Via != tc.via {
			t.Fatal(w.Code, w.Body.String(), err)
		}
	}
}

func TestServeSessionStoreAndPages(t *testing.T) {
	cfg := loadTokenConfig(t, "  token: root")
	cfg.UI.PagesDir = t.TempDir()
	if err := os.WriteFile(filepath.Join(cfg.UI.PagesDir, "map.html"), []byte("map content"), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.pagesRoot.Close() })
	value := createStoredToken(t, s, "root", "ci", "viewer", `[]`)
	w := sessionHTTP(s, "POST", "/api/session", fmt.Sprintf(`{"token":%q}`, value), nil, nil)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	cookie := w.Result().Cookies()[0]
	for _, path := range []string{"/api/jobs", "/pages/map.html"} {
		if w := sessionHTTP(s, "GET", path, "", cookie, nil); w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	if w := storedTokenHTTP(s, "DELETE", "/api/tokens/ci", "root", ""); w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, path := range []string{"/api/jobs", "/pages/map.html"} {
		if w := sessionHTTP(s, "GET", path, "", cookie, nil); w.Code != 401 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
}

func TestServeSessionHashedToken(t *testing.T) {
	digest := sha256.Sum256([]byte("secret"))
	cfg := loadTokenConfig(t, fmt.Sprintf("  tokens: [{name: ci, token: 'sha256:%x'}]", digest))
	s, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if w := sessionHTTP(s, "POST", "/api/session", `{"token":"secret"}`, nil, nil); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
}

func TestServeSessionUI(t *testing.T) {
	if strings.Contains(serveUI, "prompt(") {
		t.Fatal("native prompt remains")
	}
	start, end := strings.Index(serveUI, "async function migrateSession()"), strings.Index(serveUI, "// Pages under")
	if start < 0 || end <= start {
		t.Fatal("missing migration")
	}
	if strings.Contains(serveUI[:start]+serveUI[end:], "serveToken") {
		t.Fatal("token storage outside migration")
	}
	for _, part := range []string{`type="password" autocomplete="off"`, `Remember on this device`, `Too many attempts`, `/api/session`} {
		if !strings.Contains(serveUI, part) {
			t.Fatal("missing", part)
		}
	}
}

func TestServeSessionStoredExpiry(t *testing.T) {
	cfg := &ServeConfig{DataDir: t.TempDir()}
	now := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
	value, record, err := makeStoredServeToken(serveTokenOptions{name: "ci", expires: "2026-10-24T12:00:00Z"}, "cli:test", func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	if _, err := addServeToken(cfg, record); err != nil {
		t.Fatal(err)
	}
	s, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	s.now = func() time.Time { return now }
	body := fmt.Sprintf(`{"token":%q,"remember":true}`, value)
	w := sessionHTTP(s, "POST", "/api/session", body, nil, nil)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	cookie := w.Result().Cookies()[0]
	if cookie.MaxAge != 24*60*60 {
		t.Fatal(cookie.MaxAge)
	}
	now = *record.ExpiresAt
	if w := sessionHTTP(s, "GET", "/api/jobs", "", cookie, nil); w.Code != 401 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := sessionHTTP(s, "POST", "/api/session", body, nil, nil); w.Code != 401 {
		t.Fatal(w.Code, w.Body.String())
	}
}

func TestServeSessionProxyClientsAndLoginCSRF(t *testing.T) {
	s, err := newServer(&ServeConfig{DataDir: t.TempDir(), Auth: ServeAuth{Token: "secret", TrustedProxies: []string{"192.0.2.5"}}}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	s.cfg.Auth.trusted, err = parseTrustedProxies(s.cfg.Auth.TrustedProxies)
	if err != nil {
		t.Fatal(err)
	}
	h := s.mux()
	signIn := func(peer, forwarded, token string, headers map[string]string) int {
		r := httptest.NewRequest("POST", "/api/session", strings.NewReader(`{"token":"`+token+`"}`))
		r.RemoteAddr = peer
		if forwarded != "" {
			r.Header.Set("X-Forwarded-For", forwarded)
		}
		for k, v := range headers {
			r.Header.Set(k, v)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Code
	}
	// one client behind the proxy fails ten times; another one still signs in
	for range 10 {
		signIn("192.0.2.5:1", "198.51.100.7, 198.51.100.1", "wrong", nil)
	}
	if code := signIn("192.0.2.5:1", "198.51.100.1", "wrong", nil); code != 429 {
		t.Fatalf("limited client: %d", code)
	}
	if code := signIn("192.0.2.5:1", "198.51.100.2", "secret", nil); code != 200 {
		t.Fatalf("other client: %d", code)
	}
	// an untrusted peer cannot pick its bucket with the header
	for range 10 {
		signIn("203.0.113.9:1", "198.51.100.50", "wrong", nil)
	}
	if code := signIn("203.0.113.9:1", "198.51.100.51", "secret", nil); code != 429 {
		t.Fatalf("untrusted peer: %d", code)
	}
	if code := signIn("192.0.2.5:1", "198.51.100.3", "secret", map[string]string{"Sec-Fetch-Site": "cross-site"}); code != 403 {
		t.Fatalf("cross-site sign-in: %d", code)
	}
	if code := signIn("192.0.2.5:1", "198.51.100.3", "secret", map[string]string{"Origin": "https://evil.example.com"}); code != 403 {
		t.Fatalf("foreign origin sign-in: %d", code)
	}
}
