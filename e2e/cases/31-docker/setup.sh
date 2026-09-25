command -v docker >/dev/null || need docker.io  # the image may ship Docker CE
docker rm -f onigirazu-e2e >/dev/null 2>&1 || true
