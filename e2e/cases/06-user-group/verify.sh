set -e
test "$(getent group e2egrp | cut -d: -f3)" = 4242
test "$(getent passwd e2euser | cut -d: -f3,4,7)" = "4242:4242:/bin/bash"
