#!/usr/bin/env bash
# Prints the e2e cases a change needs, from the files changed since BASE:
#   "all"  - the change can affect any case (engine, harness, dependencies)
#   ""     - no case (docs, packaging, unit tests only)
#   list   - case directory names, one per line
# A module file internal/modules/<name>.go selects the cases whose playbooks
# use the <name> module; a module file no case uses (shared code) selects all.
# Usage: e2e/select-cases.sh BASE [HEAD]
set -euo pipefail

base="${1:?base ref}" head="${2:-HEAD}"
cd "$(dirname "$0")/.."
cases_dir=e2e/cases

selected=()
all=""
while IFS= read -r f; do
  case "$f" in
    *.md | docs/* | examples/* | scripts/* | e2e/image/* | docker/* | docker-compose*.yml | Makefile | LICENSE* | \
      .goreleaser.yml | .golangci.yml | .gitignore | *_test.go | testdata/* | */testdata/*) ;;
    .github/workflows/e2e.yml) all=1 ;;
    .github/*) ;;
    e2e/cases/*)
      c="${f#e2e/cases/}"; c="${c%%/*}"
      [ -d "$cases_dir/$c" ] && selected+=("$c") ;;
    internal/modules/*.go)
      name="$(basename "$f" .go)"
      users="$(grep -rlE "^[[:space:]]*-?[[:space:]]*((ansible\.builtin|community\.[a-z_]+|onigirazu)\.)?$name:" \
        "$cases_dir" --include='*.yml' 2>/dev/null | cut -d/ -f3 | sort -u || true)"
      if [ -n "$users" ]; then
        while IFS= read -r c; do selected+=("$c"); done <<<"$users"
      else
        all=1
      fi ;;
    *) all=1 ;;
  esac
done < <(git diff --name-only "$base...$head")

if [ -n "$all" ]; then echo all; exit 0; fi
[ "${#selected[@]}" -gt 0 ] && printf '%s\n' "${selected[@]}" | sort -u
exit 0
