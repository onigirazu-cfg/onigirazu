set -e
test "$(cat /root/onigirazu-e2e-rf-tpl)" = port=9090
test "$(cat /root/onigirazu-e2e-rf-file)" = static
test "$(cat /root/onigirazu-e2e-rf-dep)" = from-dependency
