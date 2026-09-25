set -e
docker image inspect alpine:3.20 >/dev/null
test "$(docker inspect -f '{{.State.Running}}' onigirazu-e2e)" = true
