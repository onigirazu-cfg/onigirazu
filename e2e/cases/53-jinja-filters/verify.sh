set -e
f=/root/onigirazu-e2e-filters
grep -qx 'admins=alice' $f
grep -qx 'names=ALICE BOB' $f
grep -qx 'json={"host":"0.0.0.0","port":8080}' $f
grep -qx 'mode=debug' $f
grep -qx 'file=app.conf' $f
grep -qx 'b64=aGk=' $f
grep -qx "id=$(grep -o '^id=[^-]*' $f | cut -d= -f2)-8080" $f
test "$(grep -c '^key=' $f)" = 2
test "$(sed -n 's/^key=//p' $f | tr '\n' ' ')" = "host port "
test "$(cat /root/onigirazu-e2e-filters-when)" = ok
