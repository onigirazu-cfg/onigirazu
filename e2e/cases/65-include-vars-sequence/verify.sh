set -e
test "$(cat /root/onigirazu-e2e-iv-vars)" = "yes-file 3 from-role-vars-dir"
test "$(cat /root/onigirazu-e2e-iv-s01)" = s01
test "$(cat /root/onigirazu-e2e-iv-s02)" = s02
