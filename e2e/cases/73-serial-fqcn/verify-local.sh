# serial: 1 -> the busy intervals of the hosts do not overlap (1 s for clock skew)
set -e
"$BIN" run all -m command -a "cat /tmp/onigirazu-e2e-serial" -i "$INVENTORY" -o json 2>/dev/null |
  jq -r '.results[].message' | paste - - | sort -n > intervals
test "$(wc -l < intervals)" -ge 2
awk 'NR > 1 && $1 + 1 < prev_end { exit 1 } { prev_end = $2 }' intervals
