# Appended to setup-lib.sh and run as root on a VM cloned from a golden image:
# installs what the cases' setup.sh would install, so e2e VMs cloned from the
# resulting base template skip it. Cases still purge what they install.

# A fresh clone runs apt-daily and unattended-upgrades: stop them, then fetch
# complete package lists (need() tolerates a failed update; the image must not)
systemctl stop apt-daily.timer apt-daily-upgrade.timer apt-daily.service apt-daily-upgrade.service unattended-upgrades.service >/dev/null 2>&1 || true
for try in 1 2 3; do
  apt-get -o DPkg::Lock::Timeout=600 update -qq >/dev/null 2>&1 && break
  [ "$try" = 3 ] && { echo "prepare: apt-get update keeps failing" >&2; exit 1; }
  sleep 20
done
# on a failed install, show where apt looks (the golden images may use a mirror)
trap 'rc=$?; [ $rc = 0 ] || { echo "apt sources:" >&2; grep -rhv "^#" /etc/apt/sources.list /etc/apt/sources.list.d/ 2>/dev/null | grep -v "^$" | head -20 >&2; apt-get update 2>&1 | tail -15 >&2; }' EXIT
need git ufw mariadb-server postgresql podman
need_docker
docker compose version >/dev/null 2>&1 || need docker-compose-v2

pull() {  # tool image: Docker Hub fails now and then
  local try
  for try in 1 2 3; do "$1" pull -q "$2" >/dev/null 2>&1 && return 0; sleep 10; done
  echo "prepare: cannot pull $2 with $1" >&2; exit 1
}
pull docker mongo:7
pull docker alpine:3.20
pull podman docker.io/library/alpine:3.20

# Guest customization of a clone renames the image's netplan file and writes
# its own DHCP config without "dhcp-identifier: mac"; on 26.04 the initramfs
# and networkd then lease with different client ids and the address changes
# on reboot. A networkd drop-in survives the customization.
mkdir -p /etc/systemd/network/10-netplan-ens192.network.d
printf '[DHCPv4]\nClientIdentifier=mac\n' > /etc/systemd/network/10-netplan-ens192.network.d/10-client-id.conf

# A pre-warmed VM (e2e/pool.sh) gets its run's key after it booted: a timer
# applies guestinfo.e2e_authorized_key and e2e_hostname whenever they change
cat > /usr/local/sbin/e2e-access-refresh.sh <<'EOS'
#!/bin/bash
set -u
command -v vmware-rpctool >/dev/null || exit 0
key="$(vmware-rpctool 'info-get guestinfo.e2e_authorized_key' 2>/dev/null || true)"
[ -n "$key" ] || exit 0
[ "$(cat /home/e2e/.ssh/authorized_keys 2>/dev/null)" = "$key" ] && exit 0
rm -rf /var/lib/e2e-access
/usr/local/sbin/e2e-access.sh
EOS
chmod +x /usr/local/sbin/e2e-access-refresh.sh
cat > /etc/systemd/system/e2e-access-refresh.service <<'EOS'
[Unit]
Description=Apply a changed e2e access key from guestinfo
ConditionVirtualization=vmware
[Service]
Type=oneshot
ExecStart=/usr/local/sbin/e2e-access-refresh.sh
EOS
cat > /etc/systemd/system/e2e-access-refresh.timer <<'EOS'
[Unit]
Description=Check guestinfo for a changed e2e access key
[Timer]
OnBootSec=15s
OnUnitActiveSec=5s
AccuracySec=1s
[Install]
WantedBy=timers.target
EOS
systemctl enable e2e-access-refresh.timer >/dev/null 2>&1

# The database cases start their server; idle VMs do not need them
systemctl disable --now mariadb postgresql >/dev/null 2>&1 || true
trap - EXIT
echo "prepared: $(dpkg -l | grep -c '^ii') packages, $(docker images -q | wc -l) docker images"
