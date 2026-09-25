set -e
docker compose -p onigirazu-e2e ps --status running --services | grep -qx sleeper
