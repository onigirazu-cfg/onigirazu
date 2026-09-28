# apply records what the playbook manages; with tasks removed, plan lists
# them and apply removes the created ones and puts the adopted file back
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

"$BIN" apply playbook.yml -i "$INVENTORY" --limit "$HOST" --no-destroy > out 2>&1
"$BIN" state resources playbook.yml | grep -E "^$HOST +file +/tmp/e2e-ms/created .*orphan: destroy"

"$BIN" apply playbook.yml -i "$INVENTORY" --limit "$HOST" > out 2>&1
grep -F "removed file /tmp/e2e-ms/created on $HOST" out
grep -F "put back file /tmp/e2e-ms/existing on $HOST" out
grep -F "removed user e2ems on $HOST" out
! "$BIN" state resources playbook.yml | grep -E "^$HOST +(file +/tmp/e2e-ms/created|user)"
"$BIN" apply check.yml -i "$INVENTORY" --limit "$HOST" > out 2>&1

# back to the first playbook for the idempotency run
cp v1.yml playbook.yml
"$BIN" apply playbook.yml -i "$INVENTORY" --limit "$HOST" > out 2>&1
