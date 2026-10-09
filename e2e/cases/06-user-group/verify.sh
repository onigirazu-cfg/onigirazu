set -e
test "$(getent group e2egrp | cut -d: -f3)" = 4242
test "$(getent passwd e2euser | cut -d: -f3,4,7)" = "4242:4242:/bin/bash"
test "$(getent shadow e2epw | cut -d: -f2 | cut -c1-10)" = '$6$e2esalt$'
