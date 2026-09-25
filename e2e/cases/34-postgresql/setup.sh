command -v psql >/dev/null || DEBIAN_FRONTEND=noninteractive apt-get install -y postgresql >/dev/null 2>&1
systemctl start postgresql
sudo -u postgres psql -qc 'DROP DATABASE IF EXISTS onigirazu_e2e' -c 'DROP ROLE IF EXISTS onigirazu_e2e' >/dev/null
