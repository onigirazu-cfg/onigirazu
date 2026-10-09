#!/bin/bash
# Runs detached as root on the prepared VM (the e2e session that started it is
# killed here): removes the build access and the VM's identity the way the
# golden image does, so every clone is new, then powers off.
set -u
sleep 3
pkill -KILL -u e2e || true
userdel -rf e2e >/dev/null 2>&1 || true
rm -f /etc/sudoers.d/90-e2e /etc/ssh/sshd_config.d/90-e2e.conf
# the first-boot units of the golden image run again on each clone
rm -rf /var/lib/e2e-access
systemctl enable e2e-access.service regenerate-ssh-host-keys.service >/dev/null 2>&1 || true
rm -f /etc/ssh/ssh_host_*
truncate -s 0 /etc/machine-id
rm -f /var/lib/dbus/machine-id && ln -sf /etc/machine-id /var/lib/dbus/machine-id
systemctl stop netbird >/dev/null 2>&1 && rm -rf /var/lib/netbird/* || true
# guest customization of the build VM wrote its own netplan file; clones get
# their address by DHCP from the image's own config, on any interface, with
# the MAC as client id (26.04's installer config has another name and went
# with the rest)
find /etc/netplan -name '*.yaml' ! -name 00-installer-config.yaml -delete
cat > /etc/netplan/01-e2e-dhcp.yaml <<'NETPLAN'
network:
  version: 2
  ethernets:
    e2e-all:
      match:
        name: "en*"
      dhcp4: true
      dhcp-identifier: mac
      # a reboot keeps the address: networkd would release the lease on the
      # way down and a VM of a parallel run could take it (99-reboot failed
      # "unreachable" with 8 runs in flight); dhcp-release.sh releases it
      dhcp4-overrides:
        send-release: false
NETPLAN
chmod 600 /etc/netplan/01-e2e-dhcp.yaml
cloud-init clean --logs --seed >/dev/null 2>&1 || true
apt-get clean
rm -rf /tmp/* /var/tmp/*
journalctl --vacuum-time=1s >/dev/null 2>&1 || true
find /var/log -type f -name '*.log' -exec truncate -s 0 {} \;
sync; fstrim -a >/dev/null 2>&1 || true; sync
systemctl poweroff
