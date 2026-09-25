# Prepended to every case's setup.sh (runs as root on the host).

# need PKG...: install Debian/Ubuntu packages unless they are all installed.
# Refreshes stale package lists first; a failed install ends the setup.
need() {
  dpkg -s "$@" >/dev/null 2>&1 && return 0
  if [ -z "$(find /var/lib/apt/lists -name '*Packages*' -mmin -120 2>/dev/null | head -1)" ]; then
    apt-get update -qq >/dev/null 2>&1 || true
  fi
  DEBIAN_FRONTEND=noninteractive apt-get install -y -qq "$@" >/dev/null 2>&1 ||
    { echo "setup: cannot install $*" >&2; exit 1; }
}
