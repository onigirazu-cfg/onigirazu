set -e
test "$(stat -c '%a %U' /opt/onigirazu-e2e)" = "750 root"
test -f /opt/onigirazu-e2e/touched
