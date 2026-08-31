#!/usr/bin/env bash
# Fail if statement coverage is below the given threshold.
#
# usage: scripts/coverage.sh <threshold-percent> <coverage-profile>
set -euo pipefail

threshold=${1:-95}
profile=${2:-coverage.out}

total=$(go tool cover -func="$profile" | awk '/^total:/ {print $3}' | tr -d '%')
if [ -z "$total" ]; then
  echo "coverage: no total in $profile" >&2
  exit 1
fi

echo "coverage: ${total}% (threshold ${threshold}%)"
awk -v got="$total" -v want="$threshold" 'BEGIN { exit !(got >= want) }' || {
  echo "coverage ${total}% is below the ${threshold}% threshold" >&2
  exit 1
}
