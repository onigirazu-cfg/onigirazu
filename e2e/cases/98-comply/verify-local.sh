#!/usr/bin/env bash
# comply against this host: the bundled profiles run, the JSON report has
# every control, the log_martians control fails (the playbook turned it off)
# and --fail-on filters it, a custom profile file works too
set -euo pipefail
work=$(mktemp -d); trap 'rm -rf "$work"' EXIT
"$BIN" comply list | grep -q '^linux-baseline '
"$BIN" comply show ssh | grep -q 'id: ssh-1'
set +e
"$BIN" comply --profile linux-baseline -i "$INVENTORY" --limit "$HOST" --format json --output "$work/r.json" >/dev/null 2>"$work/err"
rc=$?
set -e
[ "$rc" = 1 ] || { echo "linux-baseline must exit 1 (log_martians is off), got $rc"; cat "$work/err"; exit 1; }
python3 - "$work/r.json" "$HOST" <<'PY'
import json, sys
r = json.load(open(sys.argv[1])); h = r["hosts"][sys.argv[2]]
assert len(h["controls"]) == 23, len(h["controls"])
bad = {c["id"] for c in h["controls"] if not c["ok"]}
assert "net-6" in bad, bad
assert all(c["remediation"] for c in h["controls"])
assert r["errors"] in (None, {}), r["errors"]
print(f"note: linux-baseline {h['passed']}/{h['passed']+h['failed']} pass on {sys.argv[2]}, failing: {sorted(bad)}")
PY
# only high+ failures count: net-6 is low, so unless something worse fails this passes
set +e
"$BIN" comply --profile linux-baseline -i "$INVENTORY" --limit "$HOST" --fail-on critical --format markdown > "$work/r.md" 2>&1
rc=$?
set -e
[ "$rc" = 0 ] || { echo "no critical control may fail on the e2e image"; cat "$work/r.md"; exit 1; }
grep -q '^## Compliance: linux-baseline' "$work/r.md"
# a custom profile from a file, filtered by tag
cat > "$work/own.yml" <<EOF
title: own
controls:
  - id: hostname
    title: the host knows its name
    severity: high
    tags: [identity]
    check: {command: {cmd: hostname, stdout: ["/^e2e-/"]}}
    remediation: hostnamectl
  - id: nope
    title: never runs (filtered out)
    tags: [other]
    check: {file: {path: /no/such/file}}
    remediation: none
EOF
"$BIN" comply --profile "$work/own.yml" -i "$INVENTORY" --limit "$HOST" --control-tags identity --format json | python3 -c "
import json,sys; r=json.load(sys.stdin); c=r['hosts']['$HOST']['controls']; assert len(c)==1 and c[0]['ok'], c"
