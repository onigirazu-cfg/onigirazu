set -e
mongosh --quiet onigirazu_e2e --eval 'db.getUser("onigirazu_e2e").roles[0].role' | grep -qx readWrite
mongosh --quiet onigirazu_e2e --eval 'db.getCollectionNames().includes("_init")' | grep -qx true
