set -e
# as Ansible stores it: relative to the parent of the archived directory
tar -tzf /root/onigirazu-e2e.tar.gz | grep -qx 'onigirazu-e2e-arch/a.txt'
tar -tzf /root/onigirazu-e2e.tar.gz | grep -qx 'onigirazu-e2e-arch/b.txt'
! tar -tzf /root/onigirazu-e2e.tar.gz | grep -q '^opt/'
