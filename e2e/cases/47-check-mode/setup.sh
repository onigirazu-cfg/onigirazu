rm -f /root/onigirazu-e2e-check-*
userdel onigirazu-e2e-check 2>/dev/null; groupdel onigirazu-e2e-check 2>/dev/null
dpkg -s cowsay >/dev/null 2>&1 && DEBIAN_FRONTEND=noninteractive apt-get -o DPkg::Lock::Timeout=600 remove -y -qq cowsay >/dev/null 2>&1
systemctl start cron 2>/dev/null
rm -f /etc/cron.d/onigirazu-e2e-check
sysctl -w vm.swappiness=60 >/dev/null 2>&1
true
