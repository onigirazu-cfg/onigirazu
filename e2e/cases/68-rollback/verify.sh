set -e
test "$(cat /root/onigirazu-e2e-rb-conf)" = version=1
test "$(stat -c %a /root/onigirazu-e2e-rb-conf)" = 644
test "$(stat -c %a /root/onigirazu-e2e-rb-dir)" = 755
