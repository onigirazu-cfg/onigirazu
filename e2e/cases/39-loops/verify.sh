set -e
. /etc/os-release
d=/root/onigirazu-e2e-loop
test -d "$d/a" && test -d "$d/b"
test "$(cat "$d/x.txt")" = "1/2 x $ID"
test "$(cat "$d/y.txt")" = "2/2 y $ID"
test "$(cat "$d/results")" = 2
test -d "$d/when-keep"
test ! -e "$d/when-skip"
