package brand

import "strings"

// ReportCSS is the one stylesheet of the standalone HTML reports, in the
// spirit of the serve web UI: zinc neutrals, the indigo accent, status
// colors, light and dark through prefers-color-scheme, and print rules.
// Reports keep only their content markup and use these classes:
//
//	main.onz-page                 page container
//	h1.onz-title, p.onz-sub       title and the muted line under it
//	div.onz-stats > div.onz-stat  summary chips: <b>count</b><span>label</span>,
//	                              with ok, warn, bad or changed on the chip
//	section.onz-card > h2         a titled block (a host, a group of results)
//	div.onz-scroll > table.onz-table   tables; td.num for numbers
//	span.onz-badge                status badge with ok, warn, bad, changed or muted
//	pre.onz-diff > span.add|del|hunk|hdr   diff lines
//	footer.onz-foot               "Generated ..." line
const ReportCSS = `:root{color-scheme:light dark;--bg:#fafafa;--card:#fff;--fg:#18181b;--muted:#63636e;--line:#e4e4e7;--hover:#f4f4f5;
--accent:` + AccentLight + `;--logo:` + LogoLight + `;--logo-drift:` + LogoDrift + `;
--ok:#187446;--ok-bg:#e5f5ec;--warn:#936013;--warn-bg:#fff2d5;--bad:#b42318;--bad-bg:#fee9e8;--changed:#245dbe;--changed-bg:#e9f0ff;
--add:#e0f4e8;--del:#fbe5e5;--hunk:#e9f0ff}
@media screen and (prefers-color-scheme:dark){:root{--bg:#09090b;--card:#18181b;--fg:#fafafa;--muted:#a1a1aa;--line:#2e2e33;--hover:#232327;
--accent:` + AccentDark + `;--logo:` + LogoDark + `;
--ok:#78d4a0;--ok-bg:#16382a;--warn:#f4c277;--warn-bg:#3a2b17;--bad:#ff8f87;--bad-bg:#422324;--changed:#9dbbff;--changed-bg:#1c2b4a;
--add:#163629;--del:#421f25;--hunk:#1c2b4a}}
*{box-sizing:border-box}
body{margin:0;background:var(--bg);color:var(--fg);font:14px/1.5 system-ui,-apple-system,"Segoe UI",Roboto,sans-serif}
a{color:var(--accent)}
.onz-page{max-width:1100px;margin:0 auto;padding:20px 16px 32px}
.onz-head{display:flex;gap:8px;align-items:center;margin:0 0 16px;font-size:15px;line-height:1.3;min-width:0}
.onz-head svg{width:24px;height:24px;flex-shrink:0}
.onz-head b{font-weight:650;flex-shrink:0}
.onz-head i{font-style:normal;color:var(--muted);flex-shrink:0}
.onz-head span{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.onz-title{margin:0;font-size:22px;line-height:1.25;font-weight:650;overflow-wrap:anywhere}
.onz-sub{margin:4px 0 0;color:var(--muted)}
h2{margin:0;font-size:15px;font-weight:600}
h3{margin:12px 0 6px;font-size:14px;font-weight:600}
.onz-stats{display:flex;flex-wrap:wrap;gap:8px;margin:16px 0 20px}
.onz-stat{display:flex;flex-direction:column;min-width:96px;padding:8px 12px;border:1px solid var(--line);border-radius:8px;background:var(--card)}
.onz-stat b{font-size:20px;line-height:1.2;font-weight:650;font-variant-numeric:tabular-nums}
.onz-stat span{color:var(--muted);font-size:12px}
.onz-stat.ok b{color:var(--ok)}.onz-stat.warn b{color:var(--warn)}.onz-stat.bad b{color:var(--bad)}.onz-stat.changed b{color:var(--changed)}
.onz-card{margin:0 0 16px;border:1px solid var(--line);border-radius:8px;background:var(--card);overflow:hidden}
.onz-card>h2{display:flex;flex-wrap:wrap;gap:8px;align-items:center;padding:10px 14px;border-bottom:1px solid var(--line)}
.onz-card>.onz-body{padding:10px 14px}
.onz-card>.onz-body+.onz-body{border-top:1px solid var(--line)}
.onz-card>.onz-scroll .onz-table th,.onz-card>.onz-scroll .onz-table td{padding-left:14px}
.onz-scroll{overflow-x:auto;max-width:100%}
.onz-table{width:100%;border-collapse:collapse;font-size:13px}
.onz-table th,.onz-table td{padding:7px 10px;text-align:left;vertical-align:top;border-bottom:1px solid var(--line)}
.onz-table th{color:var(--muted);font-size:12px;font-weight:550;white-space:nowrap;background:var(--card)}
.onz-table tr:last-child td{border-bottom:0}
.onz-table td.num,.onz-table th.num{text-align:right;font-variant-numeric:tabular-nums;white-space:nowrap}
.onz-card>.onz-table,.onz-card>.onz-scroll>.onz-table{margin:0}
main>.onz-scroll{border:1px solid var(--line);border-radius:8px;background:var(--card);margin:0 0 16px}
.onz-badge{display:inline-block;padding:1px 7px;border-radius:6px;font-size:11px;font-weight:550;line-height:1.6;white-space:nowrap;background:var(--hover);color:var(--muted)}
.onz-badge.ok{background:var(--ok-bg);color:var(--ok)}.onz-badge.warn{background:var(--warn-bg);color:var(--warn)}
.onz-badge.bad{background:var(--bad-bg);color:var(--bad)}.onz-badge.changed{background:var(--changed-bg);color:var(--changed)}
.ok{color:var(--ok)}.warn{color:var(--warn)}.bad{color:var(--bad)}.muted{color:var(--muted)}
.onz-diff{margin:6px 0 0;padding:6px 0;overflow-x:auto;border:1px solid var(--line);border-radius:6px;background:var(--bg);font:12px/1.45 ui-monospace,SFMono-Regular,Menlo,monospace}
.onz-diff span{display:block;padding:0 8px;white-space:pre}
.onz-diff .add{background:var(--add)}.onz-diff .del{background:var(--del)}.onz-diff .hunk{background:var(--hunk)}.onz-diff .hdr{color:var(--muted)}
.onz-foot{margin-top:24px;color:var(--muted);font-size:12px}
@media print{:root{color-scheme:light}body{background:#fff}.onz-page{max-width:none;padding:0}.onz-card,.onz-stat{break-inside:avoid}.onz-scroll{overflow:visible}.onz-diff{white-space:pre-wrap}.onz-diff span{white-space:pre-wrap}}`

// ReportHead is everything a report puts in <head> besides its <title>:
// charset, viewport, the favicon and ReportCSS.
const ReportHead = `<meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">` + "\n" + FaviconLink + "\n<style>" + ReportCSS + "</style>"

// Tone maps a status word from any report to one of the fixed badge and
// chip modifiers: ok, warn, bad, changed or muted. Unknown words are muted,
// so the result is always safe in a class attribute.
func Tone(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "ok", "success", "succeeded", "healthy", "pass", "passed", "fixed", "in sync", "in_sync":
		return "ok"
	case "warning", "warn", "drift", "drifting", "detected", "pending", "medium", "high", "skipped", "unreachable":
		return "warn"
	case "failure", "failed", "fail", "error", "critical", "bad", "timed_out":
		return "bad"
	case "changed", "running", "queued", "would change":
		return "changed"
	}
	return "muted"
}
