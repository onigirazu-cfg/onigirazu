set -e
me="$(hostname)"; me="${me#e2e-}"
# play 2 ran on the alias of this host, in a group of all aliases
read -r name alias count < /root/onigirazu-e2e-added
test "$name" = "$me-alias"
test "$alias" = "$me"
test "$count" -ge 1
# play 3 ran on this host through its group_by group under grouped
grep -q "by_$me" /root/onigirazu-e2e-grouped
grep -q grouped /root/onigirazu-e2e-grouped
