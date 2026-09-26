set -e
test "$(cat /root/onigirazu-e2e-ir-main)" = port=9090
test "$(cat /root/onigirazu-e2e-ir-extra)" = "extra 8080"
