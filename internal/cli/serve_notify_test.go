package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestServeNotifyEmptyEnvironment(t *testing.T) {
	const x = "SERVE_TEST_EMPTY_HOOK"
	const y = "SERVE_TEST_OTHER_HOOK"
	for _, tc := range []struct {
		name, notify string
		serverWide   bool
		env          map[string]string
		wantURLs     []string
		wantWarnings []string
		wantError    string
	}{
		{
			name: "unset job target", notify: `[{url: "${` + x + `}"}]`,
			wantWarnings: []string{"serve: notify target 1 of job patch-wave disabled: " + x + " is not set"},
		},
		{
			name: "empty job target", notify: `[{url: "$` + x + `"}]`, env: map[string]string{x: ""},
			wantWarnings: []string{"serve: notify target 1 of job patch-wave disabled: " + x + " is empty"},
		},
		{
			name: "whitespace expansion", notify: `[" ${` + x + `} "]`, env: map[string]string{x: " \t"},
			wantWarnings: []string{"serve: notify target 1 of job patch-wave disabled: " + x + " is empty"},
		},
		{
			name: "unset server target", notify: `["${` + x + `}"]`, serverWide: true,
			wantWarnings: []string{"serve: notify target 1 (server-wide) disabled: " + x + " is not set"},
		},
		{
			name: "retains next target", notify: `["${` + x + `}", "https://notify.test/kept"]`,
			wantURLs:     []string{"https://notify.test/kept"},
			wantWarnings: []string{"serve: notify target 1 of job patch-wave disabled: " + x + " is not set"},
		},
		{
			name: "original target number", notify: `["https://notify.test/kept", "${` + x + `}"]`,
			wantURLs:     []string{"https://notify.test/kept"},
			wantWarnings: []string{"serve: notify target 2 of job patch-wave disabled: " + x + " is not set"},
		},
		{
			name: "multiple variables and repeated reference", notify: `["${` + x + `}${` + y + `}${` + x + `}"]`, env: map[string]string{y: ""},
			wantWarnings: []string{"serve: notify target 1 of job patch-wave disabled: " + x + " is not set, " + y + " is empty"},
		},
		{
			name: "one warning per disabled target", notify: `["${` + x + `}", "${` + y + `}"]`, env: map[string]string{y: ""},
			wantWarnings: []string{
				"serve: notify target 1 of job patch-wave disabled: " + x + " is not set",
				"serve: notify target 2 of job patch-wave disabled: " + y + " is empty",
			},
		},
		{name: "literal empty", notify: `[{url: ""}]`, wantError: "target 1: URL is empty"},
		{name: "literal whitespace", notify: `[{url: "  "}]`, wantError: "target 1: URL is empty"},
		{name: "missing url", notify: `[{format: json}]`, wantError: "target 1: URL is empty"},
		{name: "partially expanded", notify: `["https://${` + x + `}/x"]`, wantError: x + " is not set"},
		{name: "disabled bad format", notify: `[{url: "${` + x + `}", format: xml}]`, wantError: `target 1: format "xml"`},
		{name: "disabled bad event", notify: `[{url: "${` + x + `}", on: [typo]}]`, wantError: `target 1: unknown event "typo"`},
		{
			name: "nonempty variable", notify: `["${` + x + `}"]`, env: map[string]string{x: "https://notify.test/secret"},
			wantURLs: []string{"https://notify.test/secret"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, name := range []string{x, y} {
				t.Setenv(name, "") // Restore the original environment at cleanup, even when unset below.
				if err := os.Unsetenv(name); err != nil {
					t.Fatal(err)
				}
			}
			for name, value := range tc.env {
				t.Setenv(name, value)
			}
			config := "jobs:\n  - name: patch-wave\n    playbook: patch.yml\n"
			if tc.serverWide {
				config += "notify: " + tc.notify + "\n"
			} else {
				config += "    notify: " + tc.notify + "\n"
			}
			path := filepath.Join(t.TempDir(), "serve.yml")
			if err := os.WriteFile(path, []byte(config), 0o600); err != nil {
				t.Fatal(err)
			}
			cfg, err := loadServeConfig(path)
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("error = %v, want %q", err, tc.wantError)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			targets := cfg.Jobs[0].Notify
			if tc.serverWide {
				targets = cfg.Notify
			}
			var urls []string
			for _, target := range targets {
				urls = append(urls, target.URL)
				if target.Format != "mattermost" {
					t.Errorf("target format = %q, want mattermost", target.Format)
				}
			}
			if !slices.Equal(urls, tc.wantURLs) {
				t.Errorf("URLs = %q, want %q", urls, tc.wantURLs)
			}
			if !slices.Equal(cfg.warnings, tc.wantWarnings) {
				t.Errorf("warnings = %q, want %q", cfg.warnings, tc.wantWarnings)
			}
			var stderr bytes.Buffer
			cfg.printWarnings(&stderr)
			wantOutput := ""
			for _, warning := range tc.wantWarnings {
				wantOutput += warning + "\n"
			}
			if stderr.String() != wantOutput {
				t.Errorf("stderr = %q, want %q", stderr.String(), wantOutput)
			}
		})
	}
}
