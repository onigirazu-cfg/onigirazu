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
# guest customization of the build VM wrote its own netplan file; clones get theirs
find /etc/netplan -name '*.yaml' ! -name 00-installer-config.yaml -delete
cloud-init clean --logs --seed >/dev/null 2>&1 || true
apt-get clean
rm -rf /tmp/* /var/tmp/*
journalctl --vacuum-time=1s >/dev/null 2>&1 || true
find /var/log -type f -name '*.log' -exec truncate -s 0 {} \;
sync; fstrim -a >/dev/null 2>&1 || true; sync
systemctl poweroff
