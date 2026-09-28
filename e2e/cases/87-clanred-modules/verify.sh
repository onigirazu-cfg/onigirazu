set -e
grep -qx '\[domain/e2e.example\]' /etc/onigirazu-e2e.ini
grep -qx 'dyndns_update = True' /etc/onigirazu-e2e.ini
test "$(stat -c %a /etc/onigirazu-e2e.ini)" = 640
/opt/onigirazu-e2e-venv/bin/pip show six | grep -qx 'Version: 1.16.0'
if [ -f /etc/debian_version ]; then ufw show added | grep -q 22022; fi
