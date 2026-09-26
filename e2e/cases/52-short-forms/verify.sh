set -e
test "$(stat -c %a /root/onigirazu-e2e-sf-dir)" = 750
test -f /root/onigirazu-e2e-sf-dir/made
test "$(cat /root/onigirazu-e2e-sf-pwd)" = /root/onigirazu-e2e-sf-dir
test "$(cat /root/onigirazu-e2e-sf-msg)" = "hello world"
