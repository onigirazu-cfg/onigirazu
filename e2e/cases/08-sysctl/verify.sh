set -e
test "$(sysctl -n vm.swappiness)" = 13
