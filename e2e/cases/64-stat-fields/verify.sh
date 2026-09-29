set -e
test "$(sed -n 1p /root/onigirazu-e2e-stat-out)" = "0640 root f572d396fae9206628714fb2ce00f72e94f2258f True"
test "$(sed -n 2p /root/onigirazu-e2e-stat-out)" = "True /root/onigirazu-e2e-stat-file"
