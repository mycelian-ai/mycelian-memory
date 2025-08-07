#!/usr/bin/env bash
set -euo pipefail

THRESHOLD="${1:-78.0}"

echo "Running tests with coverage..."
go test -coverprofile=coverage.out ./...

TOTAL_PCT_STR=$(go tool cover -func=coverage.out | awk '/^total:/ {print $3}')
# strip trailing %
TOTAL_PCT=${TOTAL_PCT_STR%%%}

echo "Total coverage: ${TOTAL_PCT}% (threshold ${THRESHOLD}%)"

awk -v cov="${TOTAL_PCT}" -v thr="${THRESHOLD}" 'BEGIN { if (cov+0.0 < thr+0.0) { exit 1 } }'
echo "COVERAGE CHECK PASSED"

