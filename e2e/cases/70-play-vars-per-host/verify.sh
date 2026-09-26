set -e
# exactly one file, named after the host whose name it contains
test "$(ls /root/onigirazu-e2e-pv-*.txt | wc -l)" = 1
f=$(ls /root/onigirazu-e2e-pv-*.txt)
test "$f" = "/root/onigirazu-e2e-pv-$(cat "$f").txt"
