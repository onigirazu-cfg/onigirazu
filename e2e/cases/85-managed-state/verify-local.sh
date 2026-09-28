# apply records what the playbook manages; with tasks removed, plan lists
# them: created ones to remove, an adopted file to put back
set -e
"$BIN" state resources playbook.yml > res
grep -E "^$HOST +file +/tmp/e2e-ms/created +created" res
grep -E "^$HOST +file +/tmp/e2e-ms/existing +adopted" res
grep -E "^$HOST +user +e2ems +created" res
cp playbook.yml v1.yml
trap 'cp v1.yml playbook.yml' EXIT
cp v2.yml playbook.yml
"$BIN" plan playbook.yml -i "$INVENTORY" --limit "$HOST" > out 2>&1
grep -F -- "- file /tmp/e2e-ms/created on $HOST: remove" out
grep -F -- "< file /tmp/e2e-ms/existing on $HOST: put back" out
grep -F -- "- user e2ems on $HOST: remove" out
! grep -F "/tmp/e2e-ms/kept on" out
# plan leaves the state alone
"$BIN" state resources playbook.yml | grep -E "^$HOST +file +/tmp/e2e-ms/created +created +created file"
