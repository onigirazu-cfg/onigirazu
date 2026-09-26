set -e
test -e /root/onigirazu-e2e-block-before
test ! -e /root/onigirazu-e2e-block-skipped
test "$(cat /root/onigirazu-e2e-block-rescue)" = "rescued Fails"
test "$(cat /root/onigirazu-e2e-block-always)" = always
test ! -e /root/onigirazu-e2e-block-when
test "$(cat /root/onigirazu-e2e-block-outer)" = outer
test "$(cat /root/onigirazu-e2e-block-after)" = after
