need_docker
docker compose version >/dev/null 2>&1 || need docker-compose-v2
docker compose -p onigirazu-e2e down >/dev/null 2>&1 || true
