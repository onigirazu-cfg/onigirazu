# the failed host got no further; every other host ran both plays
set -e
if [ -e /root/onigirazu-e2e-leave-failed ]; then
  test ! -e /root/onigirazu-e2e-leave-after
  test ! -e /root/onigirazu-e2e-leave-next
else
  test "$(cat /root/onigirazu-e2e-leave-after)" = "went on"
  test "$(cat /root/onigirazu-e2e-leave-next)" = next
fi
