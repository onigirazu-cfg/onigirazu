set -e
test "$(cat /etc/onigirazu-e2e-sensor.conf)" = 'sensor v1'
test "$(cat /etc/onigirazu-e2e-role)" = sensor
! dpkg -s tree >/dev/null 2>&1
