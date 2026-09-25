command -v docker >/dev/null || DEBIAN_FRONTEND=noninteractive apt-get install -y docker.io >/dev/null 2>&1
docker compose version >/dev/null 2>&1 || DEBIAN_FRONTEND=noninteractive apt-get install -y docker-compose-v2 >/dev/null 2>&1
docker compose -p onigirazu-e2e down >/dev/null 2>&1 || true
