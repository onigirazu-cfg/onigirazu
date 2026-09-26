set -e
test ! -e /root/onigirazu-e2e-check-copy
test ! -e /root/onigirazu-e2e-check-command
test "$(cat /root/onigirazu-e2e-check-report)" = "true true true true true true true true true true true"
! id onigirazu-e2e-check 2>/dev/null
! getent group onigirazu-e2e-check
! dpkg -s cowsay 2>/dev/null | grep -q '^Status: install ok installed'
systemctl is-active --quiet cron
test "$(sysctl -n vm.swappiness)" != 7
test ! -e /etc/cron.d/onigirazu-e2e-check
test ! -e /opt/onigirazu-e2e-check-clone
test ! -e /root/onigirazu-e2e-check-readme
test ! -e /etc/systemd/system/onigirazu-e2e-check.service
! grep -q onigirazu-e2e-check /etc/fstab
! findmnt /mnt/onigirazu-e2e-check >/dev/null
