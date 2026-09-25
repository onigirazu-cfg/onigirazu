set -e
test -f /opt/onigirazu-e2e-src/go.mod
test "$(git -C /opt/onigirazu-e2e-src describe --tags)" = v1.63.1
