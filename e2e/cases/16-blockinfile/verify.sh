set -e
f=/etc/onigirazu-e2e-block.conf
grep -qx 'keep=me' $f
grep -qx 'alpha=1' $f
grep -qx 'beta=2' $f
test "$(grep -c 'alpha=1' $f)" = 1
grep -q 'BEGIN' $f
grep -q 'END' $f
