#!/usr/bin/env bash
set -euo pipefail

if ! command -v jq >/dev/null 2>&1; then
  echo "[ERROR] jq is required to format test output." >&2
  echo "        Install it with your package manager (for example: apt install jq)." >&2
  exit 2
fi

input="$(mktemp)"
trap 'rm -f "$input"' EXIT
cat > "$input"

if [[ ! -s "$input" ]]; then
  echo "[ERROR] no test output received on stdin." >&2
  echo "        Usage: go test -json ... | $0" >&2
  exit 2
fi

if [[ -t 1 && -z "${NO_COLOR:-}" ]]; then
  green=$'\033[32m'
  red=$'\033[31m'
  yellow=$'\033[33m'
  bold=$'\033[1m'
  reset=$'\033[0m'
else
  green=""
  red=""
  yellow=""
  bold=""
  reset=""
fi

module="${MODULE:-}"
if [[ -z "$module" ]]; then
  module="$(go list -m 2>/dev/null || true)"
fi
label="${LABEL:-tests}"
slowest="${SLOWEST:-5}"
race_note=""
if [[ "${RACE:-0}" == "1" ]]; then
  race_note=" · race"
fi

printf '%sNoSePudo · backend · %s%s%s\n' "$bold" "$label" "$race_note" "$reset"

declare -A tests_total=()
declare -A tests_failed=()

while IFS=$'\t' read -r pkg count failed; do
  tests_total["$pkg"]="$count"
  tests_failed["$pkg"]="$failed"
done < <(jq -s -r '
  [ .[] | select((.Action == "pass" or .Action == "fail" or .Action == "skip") and .Test != null and ((.Test | contains("/")) | not)) ]
  | group_by(.Package)
  | .[] | [
      .[0].Package,
      (length | tostring),
      ([ .[] | select(.Action == "fail") ] | length | tostring)
    ] | @tsv
' "$input")

print_failures() {
  local pkg="$1"
  local test

  while IFS= read -r test; do
    printf '        %s\n' "$test"

    local body
    body="$(jq -s -r --arg p "$pkg" --arg t "$test" '
      [ .[] | select(.Action == "output" and .Package == $p and .Test == $t) | .Output ]
      | join("")
    ' "$input" | grep -vE '^(=== (RUN|PAUSE|CONT)|--- (PASS|FAIL|SKIP))' || true)"

    if [[ -n "$body" ]]; then
      local lines
      lines="$(printf '%s' "$body" | awk 'END { print NR }')"
      printf '%s\n' "$body" | head -n 40 | sed 's/^/          /'
      if (( lines > 40 )); then
        printf '          ... %d more lines\n' "$((lines - 40))"
      fi
    fi
  done < <(jq -s -r --arg p "$pkg" '
    [ .[] | select(.Action == "fail" and .Test != null and .Package == $p) | .Test ]
    | unique
    | .[]
  ' "$input")
}

print_package_output() {
  local pkg="$1"
  local body

  body="$(jq -s -r --arg p "$pkg" '
    [ .[] | select(.Action == "output" and .Package == $p and .Test == null) | .Output ]
    | join("")
  ' "$input" | grep -vE '^(FAIL$|FAIL[[:space:]]|ok[[:space:]])' | sed '/^$/d' || true)"

  if [[ -n "$body" ]]; then
    printf '%s\n' "$body" | sed 's/^/          /'
  fi
}

packages_passed=0
packages_failed=0
packages_skipped=0
test_passed_total=0
test_failed_total=0
slowest_package=""
slowest_package_elapsed=0

while IFS=$'\t' read -r action pkg elapsed; do
  name="$pkg"
  if [[ -n "$module" ]]; then
    name="${pkg#"$module"/}"
  fi
  if (( ${#name} > 38 )); then
    name="...${name: -35}"
  fi

  case "$action" in
    pass)
      status="PASS"
      color="$green"
      packages_passed=$((packages_passed + 1))
      ;;
    fail)
      status="FAIL"
      color="$red"
      packages_failed=$((packages_failed + 1))
      ;;
    *)
      status="SKIP"
      color="$yellow"
      packages_skipped=$((packages_skipped + 1))
      ;;
  esac

  counts=""
  if [[ -n "${tests_total[$pkg]:-}" && "${tests_total[$pkg]}" -gt 0 ]]; then
    if [[ "${tests_total[$pkg]}" == "1" ]]; then
      counts="1 test"
    else
      counts="${tests_total[$pkg]} tests"
    fi
    if [[ "${tests_failed[$pkg]:-0}" -gt 0 ]]; then
      if [[ "${tests_failed[$pkg]}" == "1" ]]; then
        counts="$counts, 1 test failed"
      else
        counts="$counts, ${tests_failed[$pkg]} tests failed"
      fi
      test_failed_total=$((test_failed_total + ${tests_failed[$pkg]}))
    fi
    test_passed_total=$((test_passed_total + ${tests_total[$pkg]} - ${tests_failed[$pkg]:-0}))
  fi

  if [[ -n "$counts" ]]; then
    printf '  %s%s%s  %-38.38s %6.2fs  %s\n' "$color" "$status" "$reset" "$name" "$elapsed" "$counts"
  else
    printf '  %s%s%s  %-38.38s %6.2fs\n' "$color" "$status" "$reset" "$name" "$elapsed"
  fi

  if awk -v a="$elapsed" -v b="$slowest_package_elapsed" 'BEGIN { exit !(a + 0 > b + 0) }'; then
    slowest_package="$name"
    slowest_package_elapsed="$elapsed"
  fi

  if [[ "$action" == "fail" ]]; then
    print_failures "$pkg"
    print_package_output "$pkg"
  fi
done < <(jq -s -r '
  [ .[] | select((.Action == "pass" or .Action == "fail" or .Action == "skip") and .Test == null) ]
  | sort_by(.Package)
  | .[] | [.Action, .Package, (.Elapsed // 0)] | @tsv
' "$input")

total_tests=$((test_passed_total + test_failed_total))
if (( total_tests == 1 )); then
  total_label="1 test"
else
  total_label="$total_tests tests"
fi

printf '\n  %d passed · %d failed · %d skipped · %s' \
  "$packages_passed" "$packages_failed" "$packages_skipped" "$total_label"
if (( test_failed_total > 0 )); then
  if (( test_failed_total == 1 )); then
    printf ' (1 test failed)'
  else
    printf ' (%d tests failed)' "$test_failed_total"
  fi
fi
if [[ -n "$slowest_package" ]]; then
  printf ' · slowest package %s %.2fs' "$slowest_package" "$slowest_package_elapsed"
fi
printf '\n'

if (( slowest > 0 )); then
  slow_list="$(jq -s -r --argjson n "$slowest" '
    [ .[] | select((.Action == "pass" or .Action == "fail") and .Test != null and ((.Test | contains("/")) | not)) | select((.Elapsed // 0) > 0) ]
    | sort_by(-.Elapsed)
    | .[:$n]
    | .[] | [.Test, (.Elapsed | tostring)] | @tsv
  ' "$input")"

  if [[ -n "$slow_list" ]]; then
    printf '  slowest  '
    first=1
    while IFS=$'\t' read -r test elapsed; do
      if (( first == 0 )); then
        printf ' · '
      fi
      printf '%s %ss' "$test" "$elapsed"
      first=0
    done <<< "$slow_list"
    printf '\n'
  fi
fi

printf '\n'

if (( packages_failed > 0 )); then
  exit 1
fi
