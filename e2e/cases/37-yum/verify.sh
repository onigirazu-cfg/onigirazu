set -e
command -v rpm >/dev/null || exit 0
rpm -q tree
