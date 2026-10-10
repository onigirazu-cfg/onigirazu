#!/usr/bin/env bash
# dhcp-release.sh KEY IP...: each VM gives its DHCP lease back before the hard
# power-off (a powered-off VM never does, and the pool of the e2e network ran
# out of addresses). The image keeps its lease across reboots (send-release
# off, seal.sh): a runtime drop-in turns the release on, then the link goes
# down and systemd-networkd sends the RELEASE; it is done detached, the SSH
# session dies with the link.
key=$1; shift
for ip in "$@"; do
  [ -n "$ip" ] && [ "$ip" != null ] || continue
  # shellcheck disable=SC2016  # expands on the VM
  timeout 10 ssh -i "$key" -o BatchMode=yes -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null \
    -o LogLevel=ERROR -o ConnectTimeout=5 "e2e@$ip" \
    'sudo -n nohup sh -c "for n in /run/systemd/network/*.network; do mkdir -p \$n.d; printf \"[DHCPv4]\\nSendRelease=yes\\n\" > \$n.d/zz-release.conf; done; networkctl reload; sleep 1; for i in /sys/class/net/en*; do networkctl down \${i##*/}; done" >/dev/null 2>&1 &' &
done
wait
sleep 3  # the RELEASE leaves before the power-off
