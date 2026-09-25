set -e
test "$(sudo -u postgres psql -tAc "SELECT 1 FROM pg_database WHERE datname='onigirazu_e2e'")" = 1
PGPASSWORD=e2e-Passw0rd psql -h 127.0.0.1 -U onigirazu_e2e -d onigirazu_e2e -tAc 'SELECT 1' | grep -qx 1
