set -e
test "$(head -1 /etc/onigirazu-e2e-free-1)" = "e2e-$(hostname | sed 's/^e2e-//')" || test "$(head -1 /etc/onigirazu-e2e-free-1)" = "$(hostname)" || grep -q . /etc/onigirazu-e2e-free-1
grep -qx "step 2" /etc/onigirazu-e2e-free-1b
test "$(cat /etc/onigirazu-e2e-free-2)" = ok
test -e /etc/onigirazu-e2e-free-always
test "$(cat /etc/onigirazu-e2e-free-handler)" = ran
