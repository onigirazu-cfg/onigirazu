set -e
test "$(cat /root/onigirazu-e2e-kw-count)" = 3
test "$(cat /root/onigirazu-e2e-kw-handler)" = fired
