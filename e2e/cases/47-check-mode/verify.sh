set -e
test ! -e /root/onigirazu-e2e-check-copy
test ! -e /root/onigirazu-e2e-check-command
test "$(cat /root/onigirazu-e2e-check-report)" = "true true true true"
! id onigirazu-e2e-check 2>/dev/null
! getent group onigirazu-e2e-check
! dpkg -s cowsay 2>/dev/null | grep -q '^Status: install ok installed'
systemctl is-active --quiet cron
