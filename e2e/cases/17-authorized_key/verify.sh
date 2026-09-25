set -e
test "$(grep -c 'e2e@onigirazu' /root/.ssh/authorized_keys)" = 1
test "$(stat -c '%a %U' /root/.ssh/authorized_keys)" = "600 root"
