set -e
test ! -e /root/onigirazu-e2e-check-copy
test ! -e /root/onigirazu-e2e-check-command
test "$(cat /root/onigirazu-e2e-check-report)" = true
