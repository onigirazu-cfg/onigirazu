set -e
test "$(sed -n 1p /root/onigirazu-e2e-sf-out)" = "0640 root f572d396fae9206628714fb2ce00f72e94f2258f true"
test "$(sed -n 2p /root/onigirazu-e2e-sf-out)" = "true /root/onigirazu-e2e-sf-file"
