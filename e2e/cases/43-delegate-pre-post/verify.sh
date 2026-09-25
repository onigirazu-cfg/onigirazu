set -e
test -s /root/onigirazu-e2e-pre
test -s /root/onigirazu-e2e-post
if [ "$(hostname)" = e2e-u2404 ]; then
  test "$(cat /root/onigirazu-e2e-delegated-u2404)" = "from u2404"
  test "$(cat /root/onigirazu-e2e-delegated-u2604)" = "from u2604"
else
  ! ls /root/onigirazu-e2e-delegated-* 2>/dev/null
fi
