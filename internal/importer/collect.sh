# onigirazu import: describe what makes this host different from a fresh
# install. Read-only. One record per line: KIND<TAB>fields...
export LC_ALL=C
max_size=${MAX_SIZE:-1048576}
max_tree=${MAX_TREE:-300}
tmp=$(mktemp -d) || exit 1
trap 'rm -rf "$tmp"' EXIT
T=$(printf '\t')

os=unknown
if command -v dpkg-query >/dev/null 2>&1; then os=debian
elif command -v rpm >/dev/null 2>&1; then os=redhat; fi
( . /etc/os-release 2>/dev/null; printf 'OS\t%s\t%s\t%s\n' "$os" "${ID:-}" "${VERSION_ID:-}" )
# what a file may say about the host itself (templates replace it)
short=$(cat /proc/sys/kernel/hostname 2>/dev/null)
fqdn=$(hostname -f 2>/dev/null || echo "$short")
ip=$( (hostname -I 2>/dev/null || ip -4 -o addr show scope global 2>/dev/null | awk '{sub("/.*","",$4); print $4}') | awk '{print $1; exit}')
printf 'HOST\t%s\t%s\t%s\n' "$short" "$fqdn" "$ip"

# packages installed by hand, and every package file (for ownership)
case $os in
debian)
  apt-mark showmanual 2>/dev/null | sed "s/^/PKG$T/"
  cat /var/lib/dpkg/info/*.list 2>/dev/null | sort -u > "$tmp/owned"
  dpkg --verify 2>/dev/null | awk 'substr($1,3,1)=="5" && $2=="c" {print $3}' > "$tmp/changed"
  ;;
redhat)
  (dnf repoquery --userinstalled --qf '%{name}' 2>/dev/null || yum history userinstalled 2>/dev/null | tail -n +2) |
    grep -v '^$' | sed "s/^/PKG$T/"
  rpm -qa --qf '[%{FILENAMES}\n]' 2>/dev/null | sort -u > "$tmp/owned"
  rpm -Va --nomtime --nodeps 2>/dev/null | awk 'substr($1,3,1)=="5" && $2=="c" {print $3}' > "$tmp/changed"
  ;;
*) : > "$tmp/owned" ;;
esac
touch "$tmp/changed"
sed "s/^/CHANGED$T/" "$tmp/changed"

# services: unit file state and vendor preset, and the running ones
[ -d /run/systemd/system ] && printf 'SYSTEMD\tyes\n'
if command -v systemctl >/dev/null 2>&1; then
  systemctl list-unit-files --type=service --no-legend --no-pager 2>/dev/null |
    awk '{print "UNIT\t" $1 "\t" $2 "\t" ($3 == "" ? "-" : $3)}'
  systemctl list-units --type=service --state=active --no-legend --no-pager --plain 2>/dev/null |
    awk '{print "ACTIVE\t" $1}'
fi

getent passwd | sed "s/^/PASSWD$T/"
getent group | sed "s/^/GROUP$T/"
[ -r /etc/fstab ] && grep -v '^[[:space:]]*#' /etc/fstab | grep -v '^[[:space:]]*$' | sed "s/^/FSTAB$T/"
[ -L /etc/localtime ] && printf 'TZ\t%s\n' "$(readlink /etc/localtime)"

# files no package owns, under the places people configure
skip='^/etc/(shadow|gshadow|passwd|group|subuid|subgid)-?$|^/etc/ssh/ssh_host_|^/etc/machine-id$|^/etc/hostname$|^/etc/hosts$|^/etc/resolv\.conf$|^/etc/mtab$|^/etc/fstab$|^/etc/ld\.so\.cache$|^/etc/\.pwd\.lock$|^/etc/localtime$|^/etc/timezone$|^/etc/adjtime$|^/etc/mailname$|^/etc/ssl/certs/|^/etc/alternatives/|^/etc/rc[0-6S]\.d/|^/etc/systemd/(system|user)/.*\.(wants|requires)(/|$)|^/etc/security/opasswd$|^/etc/apparmor\.d/cache/|^/etc/pki/ca-trust/extracted/|^/etc/pki/tls/certs/ca-|^/etc/lvm/(archive|backup|cache)/|^/etc/udev/hwdb\.bin$|^/etc/cloud(/|$)|^/etc/netplan/50-cloud-init|^/etc/sudoers\.d/90-cloud-init|^/etc/ca-certificates\.conf$|^/etc/\.updated$|^/etc/selinux/.*/(active|contexts/files/file_contexts\.(bin|homedirs))|^/etc/ld\.so\.conf\.d/|^/etc/dconf/db/|^/etc/X11/|^/etc/fonts/conf\.d/|^/etc/os-release$|^/etc/motd$|^/etc/issue(\.net)?$|^/etc/environment$|^/etc/initramfs-tools/|^/etc/default/grub\.d/|^/etc/modprobe\.d/|^/etc/kernel/|^/etc/sysconfig/network-scripts/|^/etc/NetworkManager/system-connections/|^/etc/crypttab$|^/etc/subuid|^/etc/subgid'
roots=""
for r in /etc /opt /srv /usr/local /var/spool/cron; do [ -d "$r" ] && roots="$roots $r"; done
# big application trees are listed, not taken
for top in /opt/* /srv/* /usr/local/*; do
  [ -d "$top" ] || continue
  n=$(find "$top" -xdev \( -type f -o -type l \) 2>/dev/null | head -n $((max_tree + 1)) | wc -l)
  if [ "$n" -gt "$max_tree" ]; then printf 'TREE\t%s\n' "$top"; echo "$top" >> "$tmp/trees"; fi
done
touch "$tmp/trees"
# shellcheck disable=SC2086
find $roots -xdev \( -type f -o -type l -o -type d \) 2>/dev/null | sort > "$tmp/all"
comm -23 "$tmp/all" "$tmp/owned" | grep -Ev "$skip" > "$tmp/unowned"
if [ -s "$tmp/trees" ]; then
  grep -v -F -f "$tmp/trees" "$tmp/unowned" > "$tmp/unowned2" || true
  mv "$tmp/unowned2" "$tmp/unowned"
fi

emit() {  # path
  p=$1
  if [ -L "$p" ]; then
    t=$(readlink "$p")
    # systemctl enable/alias links are the services' business
    case "$p:$t" in /etc/systemd/*:/usr/lib/systemd/*|/etc/systemd/*:/lib/systemd/*|/etc/systemd/*:/dev/null) return ;; esac
    printf 'LINK\t%s\t%s\n' "$p" "$t"
  elif [ -d "$p" ]; then
    printf 'DIR\t%s\t%s\n' "$p" "$(stat -c '%a %U %G %u %g' "$p" | tr ' ' '\t')"
  elif [ -f "$p" ]; then
    size=$(stat -c %s "$p")
    if [ "$size" -gt "$max_size" ]; then printf 'BIG\t%s\t%s\n' "$p" "$size"; return; fi
    printf 'FILE\t%s\t%s\n' "$p" "$(stat -c '%a %U %G %u %g' "$p" | tr ' ' '\t')"
    printf 'DATA\t'; base64 -w0 < "$p"; echo
  fi
}
# unowned files, then the changed conffiles
while IFS= read -r p; do emit "$p"; done < "$tmp/unowned"
grep -Ev "$skip" "$tmp/changed" | while IFS= read -r p; do emit "$p"; done
echo END
