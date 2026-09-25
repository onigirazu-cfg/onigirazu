set -e
printf '# managed by onigirazu\n10.0.0.1 app1 primary\n10.0.0.2 app2\nretries=3\ntimeout=30\n' | diff - /etc/onigirazu-e2e-hosts
test "$(cat /etc/onigirazu-e2e-backends)" = "app1,app2"
