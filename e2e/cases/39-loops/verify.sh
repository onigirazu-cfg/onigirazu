set -e
. /etc/os-release
d=/root/onigirazu-e2e-loop
test -d "$d/a" && test -d "$d/b"
test "$(cat "$d/x.txt")" = "1/2 x $ID"
test "$(cat "$d/y.txt")" = "2/2 y $ID"
test "$(cat "$d/results")" = 2
test -d "$d/when-keep"
test ! -e "$d/when-skip"
case "$ID" in ubuntu) fam=debian ;; *) fam=redhat ;; esac
test "$(cat "$d/nested")" = "$fam"
test "$(cat "$d/braces")" = "{{ .Id }}"
test "$(cat "$d/with")" = "$(printf '%s\n' 'combinations 4' 'nested 1-a' 'nested 1-b' 'nested 2-a' 'nested 2-b' 'together a-1' 'together b-2' 'sub ann-k1' 'sub ann-k2' 'sub bob-k3' 'indexed 0-x' 'indexed 1-y')"
