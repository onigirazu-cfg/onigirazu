set -e
grep -q 'MIT License' /root/onigirazu-e2e-LICENSE
test "$(stat -c '%a' /root/onigirazu-e2e-LICENSE)" = 600
