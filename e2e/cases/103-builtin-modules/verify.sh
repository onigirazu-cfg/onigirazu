set -e
test "$(cat /etc/onigirazu-e2e-assembled)" = "$(printf 'one\n# ---\ntwo')"
grep -q 'e2e-host.example ssh-ed25519' /root/.ssh/known_hosts
sel="$(dpkg --get-selections tree 2>/dev/null | awk '{print $2}')"; [ -z "$sel" ] || test "$sel" = install
grep -q '^\[onigirazu-e2e\]' /tmp/onigirazu-e2e-repos/onigirazu-e2e.repo
grep -q '^enabled=0' /tmp/onigirazu-e2e-repos/onigirazu-e2e.repo
test "$(cat /etc/onigirazu-e2e-raw)" = raw-ok
cat /etc/onigirazu-e2e-builtin
test "$(sed -n 1p /etc/onigirazu-e2e-builtin)" = True
test "$(sed -n 2p /etc/onigirazu-e2e-builtin)" = "True True"
test "$(sed -n 3p /etc/onigirazu-e2e-builtin)" = 1
