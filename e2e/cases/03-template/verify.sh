set -e
test "$(cat /etc/onigirazu-e2e/app.conf)" = "port = 8080"
test "$(stat -c '%a' /etc/onigirazu-e2e/app.conf)" = "644"
