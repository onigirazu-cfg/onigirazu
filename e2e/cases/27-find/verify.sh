set -e
test "$(cat /root/onigirazu-e2e-find)" = "2 /opt/onigirazu-e2e-find/sub/old 3"
