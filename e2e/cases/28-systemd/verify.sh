set -e
test -f /etc/systemd/system/onigirazu-e2e.service
systemctl is-active --quiet onigirazu-e2e.service
systemctl is-enabled --quiet onigirazu-e2e.service
