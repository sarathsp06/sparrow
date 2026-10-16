#!/usr/bin/env sh
# Core statement coverage from one or more Go cover profiles.
#
#   scripts/coverage.sh [-min PERCENT] [-v] profile.out...
#
# Profiles are merged block by block (a block counts as covered if any profile
# covered it), so unit and integration runs made with the same -coverpkg add
# up. "Core" is the server and its libraries: internal/... and pkg/...
# Generated tracing wrappers (*_otel.go) are left out. Exits 1 when total
# coverage is below -min.
set -eu

min=0
verbose=0
while [ $# -gt 0 ]; do
  case "$1" in
    -min) min="$2"; shift 2 ;;
    -v) verbose=1; shift ;;
    --) shift; break ;;
    -*) echo "unknown flag $1" >&2; exit 2 ;;
    *) break ;;
  esac
done
[ $# -gt 0 ] || { echo "usage: $0 [-min PERCENT] [-v] profile.out..." >&2; exit 2; }

awk -v min="$min" -v verbose="$verbose" '
  FNR == 1 && /^mode:/ { next }
  {
    split($1, loc, ":")
    file = loc[1]
    if (file !~ /^github\.com\/sarathsp06\/sparrow\/(internal|pkg)\//) next
    if (file ~ /_otel\.go$/) next
    stmts[$1] = $2
    if ($3 > 0) hit[$1] = 1
  }
  END {
    for (b in stmts) {
      split(b, loc, ":")
      pkg = loc[1]
      sub(/\/[^\/]*$/, "", pkg)
      sub(/^github\.com\/sarathsp06\/sparrow\//, "", pkg)
      total[pkg] += stmts[b]; all += stmts[b]
      if (b in hit) { covered[pkg] += stmts[b]; done += stmts[b] }
    }
    if (verbose) {
      for (p in total)
        printf "%6.1f%%  %5d stmts  %s\n", 100 * covered[p] / total[p], total[p], p | "sort -n"
      close("sort -n")
    }
    pct = all ? 100 * done / all : 0
    printf "core coverage: %.1f%% of %d statements\n", pct, all
    if (pct + 0 < min + 0) {
      printf "below the %s%% minimum\n", min
      exit 1
    }
  }
' "$@"
