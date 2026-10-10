#!/usr/bin/env bash
# A listener on the runner gets an Alertmanager alert for this host and runs
# fix.yml with --limit on it; the file the playbook writes proves the run.
set -euo pipefail
port=$((20000 + RANDOM % 10000))
work=$(mktemp -d); trap 'kill $pid 2>/dev/null || true; rm -rf "$work"' EXIT
cp fix.yml "$work/"
cat > "$work/listen.yml" <<EOF
listen: "127.0.0.1:$port"
sources:
  - {name: alerts, type: alertmanager, token: e2e-token}
rules:
  - name: disk
    source: alerts
    when: "event.status == 'firing' and event.labels.alertname == 'DiskFull'"
    playbook: fix.yml
    inventory: $INVENTORY
    limit: "{{ event.alerts | map(attribute='labels.instance') | join(',') }}"
    extra_vars: {alertname: "{{ event.labels.alertname }}", severity: "{{ event.labels.severity }}"}
    throttle: 1m
EOF
"$BIN" listen --config-file "$work/listen.yml" > "$work/log" 2>&1 &
pid=$!
for _ in $(seq 1 50); do curl -fs "http://127.0.0.1:$port/healthz" >/dev/null 2>&1 && break; sleep 0.2; done
alert="{\"status\":\"firing\",\"alerts\":[{\"labels\":{\"alertname\":\"DiskFull\",\"instance\":\"$HOST\",\"severity\":\"warning\"}}]}"
# a wrong token is refused, a HighLoad alert matches nothing, the DiskFull one runs
curl -s -o /dev/null -w '%{http_code}\n' -X POST -H 'Content-Type: application/json' -d "$alert" "http://127.0.0.1:$port/alerts" | grep -qx 401
curl -fs -X POST -H 'Authorization: Bearer e2e-token' -H 'Content-Type: application/json' \
  -d "${alert/DiskFull/HighLoad}" "http://127.0.0.1:$port/alerts" | grep -q '"matched":0'
curl -fs -X POST -H 'Authorization: Bearer e2e-token' -H 'Content-Type: application/json' -d "$alert" "http://127.0.0.1:$port/alerts" | grep -q '"queued":\["disk"\]'
# the throttle holds the second one
curl -fs -X POST -H 'Authorization: Bearer e2e-token' -H 'Content-Type: application/json' -d "$alert" "http://127.0.0.1:$port/alerts" | grep -q '"throttled":\["disk"\]'
for _ in $(seq 1 120); do grep -q "listen: disk: ok" "$work/log" && break; sleep 1; done
grep -q "listen: disk: ok, 1 changed, 0 failed" "$work/log" || { cat "$work/log"; exit 1; }
curl -fs "http://127.0.0.1:$port/metrics" | grep -q 'onigirazu_listen_runs_total{result="ok",rule="disk"} 1'
got=$(ssh -i "$KEY" -o BatchMode=yes -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR "e2e@$HOST_IP" 'sudo cat /etc/onigirazu-e2e-listen' 2>/dev/null)
test "$got" = "DiskFull warning"
echo "note: alert -> listener -> fix.yml on $HOST: $got"
