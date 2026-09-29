set -e
f=/root/onigirazu-e2e-av
. /etc/os-release
case "$ID" in ubuntu|debian) fam=Debian ;; rocky|rhel|centos|almalinux|fedora) fam=RedHat ;; *) fam=unknown ;; esac
grep -qx "family=$fam" $f
grep -qx 'same=True' $f
grep -Eqx "dist=(Ubuntu|Rocky|Debian|AlmaLinux|CentOS|RedHat|Fedora)" $f
grep -qx "major=${VERSION_ID%%.*}" $f
grep -qx "hostname=$(hostname -s)" $f
grep -qx 'self=8080' $f
grep -q '^all=[1-9]' $f
grep -qx 'inall=True' $f
test "$(sed -n 's/^all=//p' $f)" = "$(sed -n 's/^ips=//p' $f)"
