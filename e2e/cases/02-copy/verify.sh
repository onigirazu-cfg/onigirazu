set -e
test "$(cat /etc/onigirazu-e2e/copy.conf)" = "copied by onigirazu"
test "$(stat -c '%a %U %G' /etc/onigirazu-e2e/copy.conf)" = "640 root root"
test "$(cat /etc/onigirazu-e2e/tree-contents/top.txt)" = top
test "$(cat /etc/onigirazu-e2e/tree-contents/sub/deep.txt)" = deep
test "$(stat -c %a /etc/onigirazu-e2e/tree-contents/sub/deep.txt)" = 644
test "$(cat /etc/onigirazu-e2e/tree-dir/tree/sub/deep.txt)" = deep
test "$(cat /etc/onigirazu-e2e/new-dir/top.txt)" = top
