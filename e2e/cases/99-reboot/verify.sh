set -e
# /run is a tmpfs: the marker is gone after a real reboot
test ! -e /run/onigirazu-e2e-before-reboot
