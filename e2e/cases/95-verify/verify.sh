set -e
test -f /etc/onigirazu-e2e-verify.conf
getent group e2everify >/dev/null
