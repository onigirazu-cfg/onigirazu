set -e
mysql -N -e "SHOW DATABASES LIKE 'onigirazu_e2e'" | grep -qx onigirazu_e2e
mysql -uonigirazu_e2e -pe2e-Passw0rd -e "USE onigirazu_e2e; CREATE TABLE IF NOT EXISTS t (i INT);"
