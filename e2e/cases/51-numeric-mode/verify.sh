set -e
test "$(stat -c %a /root/onigirazu-e2e-mode-copy)" = 600
test "$(stat -c %a /root/onigirazu-e2e-mode-dir)" = 750
test "$(stat -c %a /root/onigirazu-e2e-mode-tpl)" = 640
