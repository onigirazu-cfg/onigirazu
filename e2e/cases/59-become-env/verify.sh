set -e
test "$(cat /root/onigirazu-e2e-be-users)" = "e2e|root"
test "$(cat /root/onigirazu-e2e-be-play)" = play
test "$(cat /root/onigirazu-e2e-be-block)" = "play block from-block-vars"
