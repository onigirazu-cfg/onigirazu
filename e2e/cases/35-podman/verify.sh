set -e
test "$(podman inspect -f '{{.State.Running}}' onigirazu-e2e-podman)" = true
