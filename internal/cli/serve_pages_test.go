package cli

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServePagesConfig(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "pages"), 0o700); err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(dir, "serve.yml")
	for _, tc := range []struct {
		name, ui string
		bad      bool
	}{
		{"relative", "pages_dir: pages\n  links: [{title: Map, href: /pages/map.html}]", false},
		{"external", "links: [{title: Map, href: https://example.com/map}]", false},
		{"missing", "pages_dir: missing", true},
		{"file", "pages_dir: serve.yml", true},
		{"network", "links: [{title: Map, href: '//evil'}]", true},
		{"script", "links: [{title: Map, href: 'javascript:x'}]", true},
		{"empty title", "links: [{title: '', href: /pages/map.html}]", true},
		{"blank title", "links: [{title: '  ', href: /pages/map.html}]", true},
		{"empty href", "links: [{title: Map, href: ''}]", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := os.WriteFile(config, []byte("ui:\n  "+tc.ui+"\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			cfg, err := loadServeConfig(config)
			if tc.bad {
				if err == nil || !strings.Contains(err.Error(), "ui.") {
					t.Fatalf("expected UI config error, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if tc.name == "relative" && cfg.UI.PagesDir != filepath.Join(dir, "pages") {
				t.Fatalf("pages_dir = %q", cfg.UI.PagesDir)
			}
		})
	}
}

func TestServePages(t *testing.T) {
	dir := t.TempDir()
	pages := filepath.Join(dir, "pages")
	for _, name := range []string{"", "empty", "indexed", ".private"} {
		if err := os.MkdirAll(filepath.Join(pages, name), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	for name, content := range map[string]string{
		"pages/map.html": "map content", "pages/indexed/index.html": "index content",
		"pages/.hidden": "hidden content", "pages/.private/map.html": "hidden content",
		"serve.yml": "outside secret",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(filepath.Join(dir, "serve.yml"), filepath.Join(pages, "escape")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(dir, filepath.Join(pages, "escape-dir")); err != nil {
		t.Fatal(err)
	}
	cfg := &ServeConfig{DataDir: filepath.Join(dir, "data"), Auth: ServeAuth{Token: "tok"},
		UI: ServeUI{PagesDir: pages, Links: []ServeLink{{Title: "Map", Href: "/pages/map.html"}}}}
	s, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.pagesRoot.Close() })
	handler := s.mux()
	for _, tc := range []struct {
		name, method, target, token, accept string
		status                              int
		body                                string
	}{
		{"unauthorized", "GET", "/pages/map.html", "", "", 401, `{"error":"unauthorized"}`},
		{"html unauthorized", "GET", "/pages/map.html", "", "text/html", 401, "Sign in to onigirazu serve first"},
		{"file", "GET", "/pages/map.html", "tok", "", 200, "map content"},
		{"head", "HEAD", "/pages/map.html", "tok", "", 200, ""},
		{"parent", "GET", "/pages/../serve.yml", "tok", "", 404, ""},
		{"encoded parent", "GET", "/pages/%2e%2e/serve.yml", "tok", "", 404, ""},
		{"encoded slash", "GET", "/pages/%2e%2e%2fserve.yml", "tok", "", 404, ""},
		{"symlink", "GET", "/pages/escape", "tok", "", 404, ""},
		{"directory symlink", "GET", "/pages/escape-dir/serve.yml", "tok", "", 404, ""},
		{"empty directory", "GET", "/pages/empty/", "tok", "", 404, ""},
		{"index", "GET", "/pages/indexed/", "tok", "", 200, "index content"},
		{"hidden", "GET", "/pages/.hidden", "tok", "", 404, ""},
		{"hidden directory", "GET", "/pages/.private/map.html", "tok", "", 404, ""},
		{"post", "POST", "/pages/map.html", "tok", "", 405, ""},
		{"redirect", "GET", "/pages", "tok", "", 301, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, tc.target, nil)
			if tc.token != "" {
				r.Header.Set("Authorization", "Bearer "+tc.token)
			}
			r.Header.Set("Accept", tc.accept)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if w.Code != tc.status || !strings.Contains(w.Body.String(), tc.body) || strings.Contains(w.Body.String(), "outside secret") {
				t.Fatalf("status %d, body %q", w.Code, w.Body.String())
			}
			if tc.method == "HEAD" && w.Body.Len() != 0 {
				t.Fatalf("HEAD body = %q", w.Body.String())
			}
			if tc.name == "unauthorized" && !strings.Contains(w.Header().Get("Content-Type"), "application/json") {
				t.Fatal("missing JSON content type")
			}
			if tc.name == "html unauthorized" && (!strings.Contains(w.Header().Get("Content-Type"), "text/html") || !strings.Contains(w.Body.String(), `href="/"`)) {
				t.Fatal("missing HTML content type or sign-in link")
			}
			if tc.name == "redirect" && w.Header().Get("Location") != "/pages/" {
				t.Fatal("missing redirect location")
			}
			for header, value := range map[string]string{"X-Content-Type-Options": "nosniff", "Cache-Control": "no-cache", "Referrer-Policy": "same-origin"} {
				if w.Header().Get(header) != value {
					t.Errorf("%s = %q", header, w.Header().Get(header))
				}
			}
		})
	}
	r := httptest.NewRequest("GET", "/api/me", nil)
	r.Header.Set("Authorization", "Bearer tok")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	var me struct{ Links []ServeLink }
	if err := json.Unmarshal(w.Body.Bytes(), &me); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || len(me.Links) != 1 || me.Links[0] != cfg.UI.Links[0] {
		t.Fatalf("/api/me: %d %s", w.Code, w.Body.String())
	}
}

func TestServePagesUnsetAndStartup(t *testing.T) {
	cfg := &ServeConfig{DataDir: t.TempDir()}
	s, err := newServer(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	s.mux().ServeHTTP(w, httptest.NewRequest("GET", "/pages/x", nil))
	if w.Code != 404 {
		t.Fatalf("pages unset: %d", w.Code)
	}
	cfg.UI.PagesDir = filepath.Join(t.TempDir(), "missing")
	if _, err := newServer(cfg, io.Discard); err == nil {
		t.Fatal("expected startup error for missing pages root")
	}
}
