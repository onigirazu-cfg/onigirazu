set -e
crontab -l -u root | grep -q '^17 3 \* \* \* /bin/true'
