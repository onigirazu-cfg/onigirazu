set -e
findmnt -n -o FSTYPE /mnt/onigirazu-e2e | grep -qx tmpfs
grep -q '/mnt/onigirazu-e2e' /etc/fstab
