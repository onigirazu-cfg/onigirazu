set -e
cat /etc/onigirazu-e2e-221-result
test "$(sed -n 1p /etc/onigirazu-e2e-221-result)" = "2 True False"
test "$(sed -n 2p /etc/onigirazu-e2e-221-result)" = "True True"
test "$(sed -n 3p /etc/onigirazu-e2e-221-result)" = "plain text none True"
test "$(sed -n 4p /etc/onigirazu-e2e-221-result)" = "True apt"
grep -q '^Enabled: no' /etc/apt/sources.list.d/onigirazu-e2e.sources
