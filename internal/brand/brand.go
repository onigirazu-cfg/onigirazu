// Package brand holds the onigirazu mark, favicon and color tokens shared by
// the serve web UI and the standalone HTML reports, so every page carries the
// same look. internal/cli/serve_ui.html keeps literal copies; a test there
// checks they match these values.
package brand

import "html"

// Logo colors: five indigo cells in the grid and one salmon cell that drifted.
const (
	LogoLight   = "#2b4c8c"
	LogoDark    = "#7f9fe0"
	LogoDrift   = "#ef6a4c"
	AccentLight = "#2b4c8c"
	AccentDark  = "#9db8f0"
)

// FaviconLink is the <link rel="icon"> tag: a data URI, so pages stay offline.
// Its own style follows the system theme.
const FaviconLink = `<link rel="icon" type="image/svg+xml" href="data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 32 32'%3E%3Cstyle%3Erect{fill:%232b4c8c}rect.d{fill:%23ef6a4c}@media (prefers-color-scheme:dark){rect{fill:%237f9fe0}}%3C/style%3E%3Crect x='2' y='7' width='11.5' height='11.5' rx='2.5'/%3E%3Crect x='2' y='19.5' width='11.5' height='11.5' rx='2.5'/%3E%3Crect x='15' y='19.5' width='11.5' height='11.5' rx='2.5'/%3E%3Crect class='d' x='18.5' y='1' width='11.5' height='11.5' rx='2.5'/%3E%3C/svg%3E">`

// Mark is the header logo SVG; it reads its colors from --logo and --logo-drift.
const Mark = `<svg viewBox="0 0 32 32" aria-hidden="true"><g fill="var(--logo)"><rect x="3" y="7" width="7" height="7" rx="1.5"/><rect x="12.5" y="7" width="7" height="7" rx="1.5"/><rect x="3" y="16.5" width="7" height="7" rx="1.5"/><rect x="12.5" y="16.5" width="7" height="7" rx="1.5"/><rect x="22" y="16.5" width="7" height="7" rx="1.5"/></g><rect x="23.5" y="3" width="7" height="7" rx="1.5" fill="var(--logo-drift)"/></svg>`

// HeaderOpen and HeaderClose wrap the escaped report name in the
// "mark onigirazu / report" line; html/template reports put them in the
// template text with the name action between them, so the template escapes it.
const (
	HeaderOpen  = `<div class="onz-head">` + Mark + `<b>onigirazu</b><i>/</i><span>`
	HeaderClose = `</span></div>`
)

// Header returns the header line for the top of a report, escaping the name.
func Header(report string) string {
	return HeaderOpen + html.EscapeString(report) + HeaderClose
}
