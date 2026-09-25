command -v mysqld >/dev/null || command -v mariadbd >/dev/null || DEBIAN_FRONTEND=noninteractive apt-get install -y mariadb-server >/dev/null 2>&1
systemctl start mariadb 2>/dev/null || systemctl start mysql
mysql -e "DROP DATABASE IF EXISTS onigirazu_e2e; DROP USER IF EXISTS 'onigirazu_e2e'@'localhost';"
