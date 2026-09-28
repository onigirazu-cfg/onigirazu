#!/usr/bin/env bash
# Runs the S3 managed state tests against a throwaway single-node Garage in
# Docker (Garage ignores If-None-Match, the lock must hold anyway).
# Usage: scripts/garage-s3-test.sh [garage image]
set -euo pipefail
image="${1:-dxflrs/garage:v2.3.0}"
name="onigirazu-garage-test"
work="$(mktemp -d)"
trap 'docker rm -f "$name" >/dev/null 2>&1 || true; rm -rf "$work"' EXIT

cat > "$work/garage.toml" <<CONF
metadata_dir = "/var/lib/garage/meta"
data_dir = "/var/lib/garage/data"
db_engine = "sqlite"
replication_factor = 1
rpc_bind_addr = "[::]:3901"
rpc_public_addr = "127.0.0.1:3901"
rpc_secret = "$(openssl rand -hex 32)"
[s3_api]
s3_region = "garage"
api_bind_addr = "[::]:3900"
root_domain = ".s3.garage.localhost"
CONF

docker rm -f "$name" >/dev/null 2>&1 || true
docker run -d --name "$name" -p 127.0.0.1:3900:3900 -v "$work/garage.toml:/etc/garage.toml:ro" "$image" >/dev/null
garage() { docker exec "$name" /garage "$@" 2>/dev/null; }
for _ in $(seq 30); do garage status >/dev/null && break; sleep 1; done
node="$(garage status | grep -Eo '^[0-9a-f]{16}' | head -1)"
garage layout assign -z dc1 -c 1G "$node" >/dev/null
garage layout apply --version 1 >/dev/null
garage bucket create onigirazu-test >/dev/null
keys="$(garage key create onigirazu-test)"
garage bucket allow --read --write --owner onigirazu-test --key onigirazu-test >/dev/null

AWS_ACCESS_KEY_ID="$(awk -F': *' '/Key ID/{print $2}' <<<"$keys")"
AWS_SECRET_ACCESS_KEY="$(awk -F': *' '/Secret key/{print $2}' <<<"$keys")"
export AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY
ONIGIRAZU_S3_TEST_ENDPOINT=127.0.0.1:3900 ONIGIRAZU_S3_TEST_BUCKET=onigirazu-test ONIGIRAZU_S3_TEST_REGION=garage \
  go test -count=1 -run TestS3Store -v ./internal/managed/ | tee "$work/test.log"
# a skipped test would pass silently
grep -q -- '--- PASS: TestS3Store' "$work/test.log"
