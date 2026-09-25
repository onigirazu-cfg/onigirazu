# Prepended to every case's setup.sh (runs as root on the host).

# need PKG...: install Debian/Ubuntu packages unless they are all installed.
# Refreshes stale package lists first and waits for the dpkg lock (apt-daily
# holds it on a freshly booted VM); a failed install ends the setup.
need() {
  dpkg -s "$@" >/dev/null 2>&1 && return 0
  if [ -z "$(find /var/lib/apt/lists -name '*Packages*' -mmin -120 2>/dev/null | head -1)" ]; then
    apt-get -o DPkg::Lock::Timeout=600 update -qq >/dev/null 2>&1 || true
  fi
  local out
  out="$(DEBIAN_FRONTEND=noninteractive apt-get -o DPkg::Lock::Timeout=600 install -y -qq "$@" 2>&1)" ||
    { echo "setup: cannot install $*: $(echo "$out" | tail -2 | paste -sd' ' -)" >&2; exit 1; }
}
