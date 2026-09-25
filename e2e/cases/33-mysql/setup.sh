need mariadb-server
systemctl start mariadb 2>/dev/null || systemctl start mysql
mysql -e "DROP DATABASE IF EXISTS onigirazu_e2e; DROP USER IF EXISTS 'onigirazu_e2e'@'localhost';"
