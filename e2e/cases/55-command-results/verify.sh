set -e
test "$(cat /root/onigirazu-e2e-cr-eq)" = match
test "$(cat /root/onigirazu-e2e-cr-rc)" = "3|out|err"
test "$(cat /root/onigirazu-e2e-cr-pwd)" = /tmp
test "$(cat /root/onigirazu-e2e-cr-literal)" = "a;b"
