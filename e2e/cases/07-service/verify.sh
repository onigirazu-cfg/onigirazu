set -e
systemctl is-active --quiet cron
systemctl is-enabled --quiet cron
