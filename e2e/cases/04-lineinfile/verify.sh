set -e
test "$(grep -c '^timeout=30$' /etc/onigirazu-e2e-lines.conf)" = 1
! grep -q '^debug=' /etc/onigirazu-e2e-lines.conf
