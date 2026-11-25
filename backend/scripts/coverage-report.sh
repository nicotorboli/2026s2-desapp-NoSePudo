#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: coverage-report.sh [options] profile...

Merges one or more Go coverage profiles and prints a per-package summary.

Options:
  --out FILE     write the merged profile to FILE
  --html FILE    render the merged profile as HTML with go tool cover
  --min PERCENT  exit 1 when total statement coverage is below PERCENT
  -h, --help     show this message
EOF
}

html=""
out=""
min=""
profiles=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    --out)
      [[ $# -ge 2 ]] || { echo "[ERROR] --out needs a value" >&2; exit 2; }
      out="$2"
      shift 2
      ;;
    --html)
      [[ $# -ge 2 ]] || { echo "[ERROR] --html needs a value" >&2; exit 2; }
      html="$2"
      shift 2
      ;;
    --min)
      [[ $# -ge 2 ]] || { echo "[ERROR] --min needs a value" >&2; exit 2; }
      min="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    -*)
      echo "[ERROR] unknown option: $1" >&2
      usage >&2
      exit 2
      ;;
    *)
      profiles+=("$1")
      shift
      ;;
  esac
done

if [[ ${#profiles[@]} -eq 0 ]]; then
  echo "[ERROR] at least one coverage profile is required" >&2
  usage >&2
  exit 2
fi

if [[ -n "$min" ]] && ! [[ "$min" =~ ^[0-9]+([.][0-9]+)?$ ]]; then
  echo "[ERROR] --min must be a number between 0 and 100" >&2
  exit 2
fi

mode=""
for profile in "${profiles[@]}"; do
  if [[ ! -f "$profile" ]]; then
    echo "[ERROR] coverage profile not found: $profile" >&2
    exit 2
  fi

  first="$(head -n 1 "$profile")"
  case "$first" in
    mode:*) current="${first#mode: }" ;;
    *)
      echo "[ERROR] not a Go coverage profile: $profile" >&2
      exit 2
      ;;
  esac

  if [[ -z "$mode" ]]; then
    mode="$current"
  elif [[ "$mode" != "$current" ]]; then
    echo "[ERROR] coverage modes differ: $mode and $current" >&2
    exit 2
  fi
done

merged_tmp=""
table_tmp="$(mktemp)"
cleanup() {
  rm -f "$table_tmp"
  if [[ -n "$merged_tmp" ]]; then
    rm -f "$merged_tmp"
  fi
}
trap cleanup EXIT

if [[ -n "$out" ]]; then
  target="$out"
else
  merged_tmp="$(mktemp)"
  target="$merged_tmp"
fi

{
  printf 'mode: %s\n' "$mode"
  awk 'FNR == 1 { next } { num[$1] = $2; count[$1] += $3 } END { for (key in num) printf "%s %s %s\n", key, num[key], count[key] }' "${profiles[@]}" | LC_ALL=C sort
} > "$target"

blocks="$(awk 'NR > 1 { n++ } END { print n + 0 }' "$target")"
if [[ "$blocks" -eq 0 ]]; then
  echo "[ERROR] the merged profile has no coverage blocks" >&2
  exit 2
fi

if [[ -t 1 && -z "${NO_COLOR:-}" ]]; then
  bold=$'\033[1m'
  dim=$'\033[2m'
  reset=$'\033[0m'
else
  bold=""
  dim=""
  reset=""
fi

module="${MODULE:-}"
if [[ -z "$module" ]]; then
  module="$(go list -m 2>/dev/null || true)"
fi
if [[ -n "$module" ]]; then
  module="$module/"
fi

awk -v module="$module" '
  /^mode:/ { next }
  {
    pkg = $1
    sub(/\/[^\/]*$/, "", pkg)
    total[pkg] += $2
    if ($3 + 0 > 0) covered[pkg] += $2
    grand_total += $2
    if ($3 + 0 > 0) grand_covered += $2
  }
  END {
    for (pkg in total) {
      if (total[pkg] == 0) continue
      name = pkg
      if (module != "" && index(pkg, module) == 1) name = substr(pkg, length(module) + 1)
      printf "%.1f\t%d\t%d\t%s\n", 100 * covered[pkg] / total[pkg], covered[pkg], total[pkg], name
    }
    printf "__TOTAL__\t%d\t%d\tTOTAL\n", grand_covered, grand_total
  }
' "$target" > "$table_tmp"

read -r _total_marker total_covered total_statements _total_label < <(grep '^__TOTAL__' "$table_tmp")
total_percent="$(awk -v covered="$total_covered" -v statements="$total_statements" 'BEGIN { if (statements > 0) printf "%.1f", 100 * covered / statements; else printf "0.0" }')"

printf '%sCoverage · statements%s\n' "$bold" "$reset"

make_bar() {
  local percent="$1"
  local filled
  local bar=""
  local cell

  filled="$(awk -v p="$percent" 'BEGIN { n = int(p / 10); if (n < 0) n = 0; if (n > 10) n = 10; print n }')"
  for ((cell = 0; cell < 10; cell++)); do
    if (( cell < filled )); then
      bar="$bar#"
    else
      bar="$bar-"
    fi
  done

  printf '%s' "$bar"
}

sort -t$'\t' -k1,1n -k4,4 "$table_tmp" | while IFS=$'\t' read -r percent covered statements name; do
  if [[ "$name" == "TOTAL" ]]; then
    continue
  fi

  display="$name"
  if (( ${#display} > 38 )); then
    display="...${display: -35}"
  fi

  printf '  %-38.38s %6s%%  %s  %d/%d\n' "$display" "$percent" "$(make_bar "$percent")" "$covered" "$statements"
done

printf '  %s%-38s %6s%%  %s  %d/%d%s\n' \
  "$bold" "TOTAL" "$total_percent" "$(make_bar "$total_percent")" "$total_covered" "$total_statements" "$reset"

if [[ -n "$out" ]]; then
  printf '  %smerged  %s%s\n' "$dim" "$out" "$reset"
fi

if [[ -n "$html" ]]; then
  go tool cover -html="$target" -o "$html" >/dev/null
  printf '  %shtml    %s%s\n' "$dim" "$html" "$reset"
fi

if [[ -n "$min" ]]; then
  if ! awk -v percent="$total_percent" -v minimum="$min" 'BEGIN { exit !(percent + 0 >= minimum + 0) }'; then
    echo "[ERROR] total coverage ${total_percent}% is below the required ${min}%" >&2
    exit 1
  fi
fi
