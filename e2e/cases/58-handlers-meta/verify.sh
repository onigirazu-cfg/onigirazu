set -e
test "$(cat /root/onigirazu-e2e-hm-after-pre)" = run
test "$(cat /root/onigirazu-e2e-hm-after-flush | wc -l)" = 2
test "$(wc -l < /root/onigirazu-e2e-hm-log)" = 2
test ! -e /root/onigirazu-e2e-hm-never
