need postgresql
systemctl start postgresql
sudo -u postgres psql -qc 'DROP DATABASE IF EXISTS onigirazu_e2e' -c 'DROP ROLE IF EXISTS onigirazu_e2e' >/dev/null
