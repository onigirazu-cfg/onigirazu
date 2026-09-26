set -e
h=$(cat /root/onigirazu-e2e-taskvars | sed 's/^tier-//')
test "$(cat /root/onigirazu-e2e-env)" = "hello $h"
test "$(cat /root/onigirazu-e2e-lc-x)" = 0
test "$(cat /root/onigirazu-e2e-lc-y)" = 1
test "$(cat /root/onigirazu-e2e-wi-p)" = p
test "$(cat /root/onigirazu-e2e-wi-q)" = q
test "$(cat /root/onigirazu-e2e-nolog)" = e2e-s3cr3t
