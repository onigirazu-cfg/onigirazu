command -v podman >/dev/null || DEBIAN_FRONTEND=noninteractive apt-get install -y podman >/dev/null 2>&1
podman rm -f onigirazu-e2e-podman >/dev/null 2>&1 || true
