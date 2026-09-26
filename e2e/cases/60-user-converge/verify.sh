set -e
e=$(getent passwd onigirazu-e2e-uc)
test "$(echo "$e" | cut -d: -f7)" = /bin/bash
test "$(echo "$e" | cut -d: -f5)" = second
test "$(id -Gn onigirazu-e2e-uc | tr ' ' '\n' | sort -u | tr '\n' ' ')" = "adm onigirazu-e2e-uc "
test "$(id -gn onigirazu-e2e-uc2)" = onigirazu-e2e-uc
