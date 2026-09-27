set -e
test "$(cat /root/onigirazu-e2e-tb.conf; echo x)" = "$(printf '[app]\nport=80\nport=443\nfirst=none\nx')"
