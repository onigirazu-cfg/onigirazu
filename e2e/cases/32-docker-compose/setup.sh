command -v docker >/dev/null || need docker.io  # the image may ship Docker CE
docker compose version >/dev/null 2>&1 || need docker-compose-v2
docker compose -p onigirazu-e2e down >/dev/null 2>&1 || true
