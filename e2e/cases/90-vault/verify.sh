set -e
test "$(cat /root/onigirazu-e2e-vault-vars)" = "s3cret abc from-inline"
test "$(cat /root/onigirazu-e2e-vault-key)" = "copied secret"
test "$(cat /root/onigirazu-e2e-vault-conf)" = "port=8080 password=s3cret"
