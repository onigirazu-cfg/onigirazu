set -e
test "$(readlink /root/onigirazu-e2e-fl-sym)" = /root/onigirazu-e2e-fl-target
test "$(readlink /root/onigirazu-e2e-fl-sym2)" = /root/onigirazu-e2e-fl-target
test /root/onigirazu-e2e-fl-hard -ef /root/onigirazu-e2e-fl-target
test ! -L /root/onigirazu-e2e-fl-hard
test "$(stat -c %a /root/onigirazu-e2e-fl-target)" = 600
test "$(stat -c %U /root/onigirazu-e2e-fl-tree/a/b/f)" = nobody
test "$(stat -c %a /root/onigirazu-e2e-fl-tree/a/b/f)" = 750
test "$(stat -c %U /root/onigirazu-e2e-fl-tree)" = nobody
test ! -e /root/onigirazu-e2e-fl-missing
test "$(cat /root/onigirazu-e2e-fl-missing-result)" = true
