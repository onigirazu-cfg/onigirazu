set -e
grep -q 'BEGIN PGP PUBLIC KEY BLOCK' /etc/apt/keyrings/onigirazu-e2e.asc
test "$(cat /etc/apt/sources.list.d/onigirazu-e2e.list)" = "deb [trusted=yes] file:/opt/onigirazu-e2e-repo ./"
apt-get update -qq
