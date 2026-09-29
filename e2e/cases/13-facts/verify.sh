set -e
test "$(cat /root/onigirazu-e2e-facts)" = "True True blue"
. /etc/os-release
case "${ID_LIKE:-$ID}" in *debian*|ubuntu) family=Debian ;; *rhel*|*fedora*) family=RedHat ;; *) family=Linux ;; esac
test "$(cat /root/onigirazu-e2e-gathered)" = "$ID $VERSION_ID $family"
if [ -f /.dockerenv ]; then v=docker; else v=$(systemd-detect-virt 2>/dev/null || true); fi
case "$v" in ""|none) want="NA NA" ;; vmware) want="VMware guest" ;; *) want="$v guest" ;; esac
test "$(cat /root/onigirazu-e2e-virt)" = "$want"
