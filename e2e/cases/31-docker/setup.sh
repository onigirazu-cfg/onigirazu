command -v docker >/dev/null || DEBIAN_FRONTEND=noninteractive apt-get install -y docker.io >/dev/null 2>&1
docker rm -f onigirazu-e2e >/dev/null 2>&1 || true
