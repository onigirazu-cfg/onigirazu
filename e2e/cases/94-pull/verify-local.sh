# the host runs onigirazu itself: a local repository with a playbook, pull
# once, then a second pull with --only-on-change does nothing, and
# --drift-only after a commit exits 2
set -e
s() { ssh -i "$KEY" -o BatchMode=yes -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR "e2e@$HOST_IP" "$@"; }
scp -q -i "$KEY" -o BatchMode=yes -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR "$BIN" "e2e@$HOST_IP:/tmp/onigirazu-pull"
s 'set -e; rm -rf /tmp/pullrepo /tmp/pullco /tmp/onigirazu-e2e-pulled
mkdir -p /tmp/pullrepo && cd /tmp/pullrepo && git init -q -b main
printf -- "- hosts: all\n  tasks:\n    - copy: {dest: /tmp/onigirazu-e2e-pulled, content: \"v1\\\\n\"}\n" > site.yml
git -c user.email=e@e -c user.name=e add -A && git -c user.email=e@e -c user.name=e commit -q -m v1
/tmp/onigirazu-pull pull --repo /tmp/pullrepo --playbook site.yml --dir /tmp/pullco | grep -q "^pull: ok, 1 task"
test "$(cat /tmp/onigirazu-e2e-pulled)" = v1
/tmp/onigirazu-pull pull --repo /tmp/pullrepo --playbook site.yml --dir /tmp/pullco --only-on-change | grep -q "nothing to do"
printf -- "- hosts: all\n  tasks:\n    - copy: {dest: /tmp/onigirazu-e2e-pulled, content: \"v2\\\\n\"}\n" > site.yml
git -c user.email=e@e -c user.name=e commit -qam v2
rc=0; /tmp/onigirazu-pull pull --repo /tmp/pullrepo --playbook site.yml --dir /tmp/pullco --drift-only >/dev/null || rc=$?
test "$rc" = 2
test "$(cat /tmp/onigirazu-e2e-pulled)" = v1
echo "note: pull once, only-on-change, drift-only all as expected"'
