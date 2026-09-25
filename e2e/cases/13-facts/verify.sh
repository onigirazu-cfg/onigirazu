set -e
test "$(cat /root/onigirazu-e2e-facts)" = "true true blue"
. /etc/os-release
case "${ID_LIKE:-$ID}" in *debian*|ubuntu) family=Debian ;; *rhel*|*fedora*) family=RedHat ;; *) family=Linux ;; esac
test "$(cat /root/onigirazu-e2e-gathered)" = "$ID $VERSION_ID $family"
