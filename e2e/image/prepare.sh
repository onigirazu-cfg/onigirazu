# Appended to setup-lib.sh and run as root on a VM cloned from a golden image:
# installs what the cases' setup.sh would install, so e2e VMs cloned from the
# resulting base template skip it. Cases still purge what they install.

need git ufw mariadb-server postgresql podman
need_docker
docker compose version >/dev/null 2>&1 || need docker-compose-v2

pull() {  # tool image: Docker Hub fails now and then
  local try
  for try in 1 2 3; do "$1" pull -q "$2" >/dev/null 2>&1 && return 0; sleep 10; done
  echo "prepare: cannot pull $2 with $1" >&2; exit 1
}
pull docker mongo:7
pull docker alpine:3.20
pull podman docker.io/library/alpine:3.20

# The database cases start their server; idle VMs do not need them
systemctl disable --now mariadb postgresql >/dev/null 2>&1 || true
echo "prepared: $(dpkg -l | grep -c '^ii') packages, $(docker images -q | wc -l) docker images"
