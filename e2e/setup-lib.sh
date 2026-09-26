# Prepended to every case's setup.sh (runs as root on the host).

# need PKG...: install Debian/Ubuntu packages unless they are all installed.
# Refreshes stale package lists first, waits for the dpkg lock (apt-daily
# holds it on a freshly booted VM) and retries once; a failed install ends the
# setup.
need() {
  dpkg -s "$@" >/dev/null 2>&1 && return 0
  if [ -z "$(find /var/lib/apt/lists -name '*Packages*' -mmin -120 2>/dev/null | head -1)" ]; then
    apt-get -o DPkg::Lock::Timeout=600 update -qq >/dev/null 2>&1 || true
  fi
  local out
  # one retry after a fresh update: mirrors and the network fail now and then
  out="$(DEBIAN_FRONTEND=noninteractive apt-get -o DPkg::Lock::Timeout=600 install -y -qq "$@" 2>&1)" || {
    sleep 10
    apt-get -o DPkg::Lock::Timeout=600 update -qq >/dev/null 2>&1 || true
    out="$(DEBIAN_FRONTEND=noninteractive apt-get -o DPkg::Lock::Timeout=600 install -y -qq "$@" 2>&1)"
  } || { echo "setup: cannot install $*: $(echo "$out" | tail -2 | paste -sd' ' -)" >&2; exit 1; }
}

# need_docker: Docker, pulling Docker Hub images through Google's mirror
# (Docker Hub limits anonymous pulls per IP, and every e2e VM shares one)
need_docker() {
  command -v docker >/dev/null || need docker.io  # the image may ship Docker CE
  local f=/etc/docker/daemon.json
  grep -q mirror.gcr.io "$f" 2>/dev/null && return 0
  mkdir -p /etc/docker
  python3 - "$f" <<'PY'
import json, os, sys
path = sys.argv[1]
conf = json.load(open(path)) if os.path.exists(path) and os.path.getsize(path) else {}
conf.setdefault("registry-mirrors", []).append("https://mirror.gcr.io")
json.dump(conf, open(path, "w"), indent=2)
PY
  systemctl restart docker
}
