set -e
test "$(cat /root/onigirazu-e2e-include-a)" = "a runtime"
test "$(cat /root/onigirazu-e2e-include-b)" = "b runtime"
test "$(cat /root/onigirazu-e2e-include-nested)" = nested
test ! -e /root/onigirazu-e2e-include-skipped
