rm -rf /root/onigirazu-e2e-mode-*
if crontab -l -u root >/tmp/onigirazu-e2e-cron 2>/dev/null; then
  sed -i '/# Onigirazu: onigirazu-e2e-numeric/,+1d' /tmp/onigirazu-e2e-cron
  crontab -u root /tmp/onigirazu-e2e-cron
fi
rm -f /tmp/onigirazu-e2e-cron
