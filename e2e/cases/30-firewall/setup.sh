command -v ufw >/dev/null || DEBIAN_FRONTEND=noninteractive apt-get install -y ufw >/dev/null 2>&1
ufw delete allow 18080/tcp >/dev/null 2>&1 || true
