set -e
! dpkg -s tree >/dev/null 2>&1
! id onigirazu-e2e-rbu >/dev/null 2>&1
systemctl is-active -q cron
