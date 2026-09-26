set -e
test "$(cat /root/onigirazu-e2e-wd-http)" = 80
test "$(cat /root/onigirazu-e2e-wd-https)" = 443
test "$(cat /root/onigirazu-e2e-d2i-80)" = http
test "$(cat /root/onigirazu-e2e-d2i-443)" = https
test "$(cat /root/onigirazu-e2e-in)" = yes
test ! -e /root/onigirazu-e2e-notin
