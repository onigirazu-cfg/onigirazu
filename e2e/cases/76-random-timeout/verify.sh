set -e
grep -Eq '^[0-9]{1,2} 3 \* \* \* root /bin/true$' /etc/cron.d/onigirazu-e2e-random
