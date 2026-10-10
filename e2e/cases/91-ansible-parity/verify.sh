set -e
test "$(cat /etc/onigirazu-e2e-parity)" = '0 2 HELLO WORLD 8080 e2e [1,2,3]'
test "$(cat /etc/onigirazu-e2e-greet)" = 'long hello world'
