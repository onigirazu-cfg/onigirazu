# shellcheck shell=bash disable=SC2154 # TF_VAR_* come from the caller
# Sourced by run.sh and image/build.sh. Needs GOVC_* exported, TF_VAR_library,
# TF_VAR_datacenter, TF_VAR_folder and E2E_IMAGES ("key=family ...").
# Sets images_json {"key": "<vSphere template name>", ...} and golden_json
# {"key": "<golden [latest] item>"}. With E2E_BASE=1 a key uses the e2e base
# template built from its current golden item (image/build.sh), when one exists.

resolve_images() {
  local items pair key family name base
  items="$(govc library.info -json "/$TF_VAR_library/*")"
  images_json="{" golden_json="{"
  for pair in $E2E_IMAGES; do
    key="${pair%%=*}" family="${pair#*=}"
    name="$(jq -r --arg f "$family-" '
      [ (if type == "array" then . else [.] end)[]
        | select(.name | startswith($f)) | select((.description // "") | contains("[latest]")) | .name ] | first // empty' <<<"$items")"
    [ -n "$name" ] || { echo "error: no [latest] item for $family in $TF_VAR_library" >&2; return 1; }
    golden_json+="\"$key\":\"$name\","
    base=""
    if [ "${E2E_BASE:-}" = 1 ]; then
      base="$(govc find "/$TF_VAR_datacenter/vm/$TF_VAR_folder" -type m -name "e2e-base-$key-$name-*" 2>/dev/null | sort | tail -1)"
      base="${base##*/}"
    fi
    echo "$key: ${base:-$name}${base:+ (e2e base of $name)}"
    images_json+="\"$key\":\"${base:-$name}\","
  done
  images_json="${images_json%,}}" golden_json="${golden_json%,}}"
}
