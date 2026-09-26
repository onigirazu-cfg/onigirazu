set -e
test "$(sed -n 1p /root/onigirazu-e2e-lp)" = "from a file"
test "$(sed -n 2p /root/onigirazu-e2e-lp)" = piped
test "$(sed -n 3p /root/onigirazu-e2e-lp)" = 1
test "$(getent shadow onigirazu-e2e-lp | cut -d: -f2)" = '$6$saltsalt$TVLlQcbpFVof5W3Yz4DTP6gRstiNuHwwTt6GLc1E5n0U0aDehy0S5knV8wiOQSpT0Y77vwPZN.Pq.H91p5hVO1'
