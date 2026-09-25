set -e
tar -tzf /root/onigirazu-e2e.tar.gz | grep -q 'a.txt'
tar -tzf /root/onigirazu-e2e.tar.gz | grep -q 'b.txt'
