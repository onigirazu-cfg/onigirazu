set -e
test "$(cat /root/onigirazu-e2e-lazy)" = "port 22 22"
test ! -e /root/onigirazu-e2e-lazy-wrong
