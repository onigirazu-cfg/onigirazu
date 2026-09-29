set -e
crontab -l -u root | grep -q '^17 3 \* \* \* /bin/true'
crontab -l -u root | grep -qx '5 4 \* \* \* /bin/echo onigirazu-e2e-unmanaged'
test "$(crontab -l -u root | grep -c '^#Ansible: onigirazu_e2e$\|^# Onigirazu: onigirazu_e2e$')" = 1
