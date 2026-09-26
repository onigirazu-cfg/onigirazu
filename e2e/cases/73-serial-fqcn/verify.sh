set -e
test "$(cat /root/onigirazu-e2e-fqcn)" = v1
test "$(cat /root/onigirazu-e2e-fqcn-handler)" = handled
test "$(wc -l < /tmp/onigirazu-e2e-serial)" = 2
