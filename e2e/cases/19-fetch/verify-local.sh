set -e
f=$(find fetched -type f -name shadow | head -1)
test -n "$f"
grep -q '^root:' "$f"
