set -e
docker image inspect alpine:3.20 >/dev/null
test "$(docker inspect -f '{{.State.Running}}' onigirazu-e2e)" = true
test "$(docker inspect -f '{{.HostConfig.NanoCpus}} {{.HostConfig.Memory}}' onigirazu-e2e)" = "500000000 67108864"
docker inspect -f '{{.Config.Env}}' onigirazu-e2e-env | grep -q STAGE=two
