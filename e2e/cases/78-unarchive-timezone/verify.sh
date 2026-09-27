set -e
test "$(readlink -f /etc/localtime)" = /usr/share/zoneinfo/Europe/Madrid
test "$(cat /opt/onigirazu-e2e-ua/app/VERSION)" = v1
test "$(cat /opt/onigirazu-e2e-ua/app/conf/app.ini)" = port=8080
test "$(stat -c %U /opt/onigirazu-e2e-ua/app/conf/app.ini)" = nobody
test "$(cat /etc/hostname)" = "$(cat /opt/onigirazu-e2e-ur/hostname)"
