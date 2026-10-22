package cli

import (
	"strings"
	"testing"

	"github.com/onigirazu-cfg/onigirazu/internal/brand"
)

// serve_ui.html keeps literal copies of the brand so it stays a plain HTML
// file; this keeps them equal to internal/brand, which the reports use.
func TestServeUIMatchesBrand(t *testing.T) {
	for name, want := range map[string]string{
		"favicon":     brand.FaviconLink,
		"mark":        brand.Mark,
		"logo light":  "--logo:" + brand.LogoLight + ";--logo-drift:" + brand.LogoDrift,
		"logo dark":   "--logo:" + brand.LogoDark + ";--logo-drift:" + brand.LogoDrift,
		"accent":      "--accent:" + brand.AccentLight + ";",
		"accent dark": "--accent:" + brand.AccentDark + ";",
	} {
		if !strings.Contains(serveUI, want) {
			t.Errorf("serve_ui.html %s differs from internal/brand: want %q", name, want)
		}
	}
}
