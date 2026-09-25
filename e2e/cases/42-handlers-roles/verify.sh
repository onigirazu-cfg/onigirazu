set -e
test "$(cat /etc/onigirazu-e2e-motd)" = "managed by onigirazu"
test "$(cat /root/onigirazu-e2e-handler)" = 1
test "$(cat /root/onigirazu-e2e-role-handler)" = 1
test ! -e /root/onigirazu-e2e-must-not-run
