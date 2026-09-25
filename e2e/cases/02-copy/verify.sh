set -e
test "$(cat /etc/onigirazu-e2e/copy.conf)" = "copied by onigirazu"
test "$(stat -c '%a %U %G' /etc/onigirazu-e2e/copy.conf)" = "640 root root"
