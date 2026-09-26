set -e
test "$(cat /root/onigirazu-e2e-ar-assert)" = "True|too few packages" || test "$(cat /root/onigirazu-e2e-ar-assert)" = "true|too few packages"
test "$(cat /root/onigirazu-e2e-ar-conf)" = "listen 80 reuseport;
server_name new.example.com;
listen 8080 reuseport;"
