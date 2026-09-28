set -e
test "$(cat /root/onigirazu-e2e-uid)" = 0
test "$(cat /root/onigirazu-e2e-omit)" = ok
test "$(cat /root/onigirazu-e2e-braces)" = '{{ .Field }}'
