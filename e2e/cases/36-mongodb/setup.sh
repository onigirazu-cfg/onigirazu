# MongoDB is not packaged for Ubuntu: run the server in Docker and put a
# mongosh wrapper on PATH. mongo:8 refuses kernels >= 6.19 (SERVER-121912).
need docker.io
docker rm -f onigirazu-e2e-mongo >/dev/null 2>&1
docker run -d --name onigirazu-e2e-mongo mongo:7 >/dev/null
printf '#!/bin/sh\nexec docker exec -i onigirazu-e2e-mongo mongosh "$@"\n' > /usr/local/bin/mongosh
chmod +x /usr/local/bin/mongosh
for _ in $(seq 30); do mongosh --quiet --eval 'db.runCommand({ping: 1}).ok' 2>/dev/null | grep -qx 1 && break; sleep 2; done
