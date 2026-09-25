set -e
. /etc/os-release
test "$(cat /root/onigirazu-e2e-cond-release)" = "$VERSION_ID"
test "$(cat /root/onigirazu-e2e-cond-true)" = yes
test ! -e /root/onigirazu-e2e-cond-false
test "$(cat /root/onigirazu-e2e-cond-failed)" = false
test "$(cat /root/onigirazu-e2e-cond-until)" = 3
