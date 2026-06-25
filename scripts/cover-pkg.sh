#!/usr/bin/env bash
# cover-pkg.sh enforces a per-package statement coverage floor for every
# package under internal/ that has test files.
#
# Exemptions (per AGENTS.md):
#   - cmd/ and the root package: only internal/... packages are evaluated, so
#     they are never inspected here (os.Exit paths are covered by E2E tests).
#   - internal/config/testutil and any other package without _test.go files:
#     packages with no tests cannot be measured and are skipped.
#
# The threshold defaults to 80% and can be overridden via COVERAGE_THRESHOLD_PKG.
set -euo pipefail

THRESHOLD="${COVERAGE_THRESHOLD_PKG:-80}"
PROFILE="${COVERAGE_FILE:-coverage.out}"

# Run the full suite once with coverage instrumentation. Per-package
# percentages are read from the command output; the profile is left on disk
# for reuse by other tooling (Codecov, the total-coverage gate, etc.).
if ! test_output="$(go test -covermode=set -coverprofile="${PROFILE}" ./... 2>&1)"; then
  printf '%s\n' "${test_output}"
  echo "FAIL: go test failed; cannot evaluate per-package coverage"
  exit 1
fi
printf '%s\n' "${test_output}"
echo "----------------------------------------"

# Collect internal/ packages that actually have test files. Packages without
# tests cannot be measured and are exempt.
testable_pkgs="$(go list -f '{{if or .TestGoFiles .XTestGoFiles}}{{.ImportPath}}{{end}}' ./internal/...)"

failed=0
checked=0
for pkg in ${testable_pkgs}; do
  # Extract the coverage percentage reported for this exact package. The
  # package import path is matched as a whole whitespace-delimited field to
  # avoid partial-path collisions, then the value after "coverage:" is read.
  pct="$(printf '%s\n' "${test_output}" | awk -v p="${pkg}" '{
    for (i = 1; i <= NF; i++) {
      if ($i == p) {
        for (j = i; j <= NF; j++) {
          if ($j == "coverage:") { gsub(/%/, "", $(j + 1)); print $(j + 1); exit }
        }
      }
    }
  }')"

  if [ -z "${pct}" ]; then
    echo "WARN: no coverage reported for ${pkg} (skipped)"
    continue
  fi

  checked=$((checked + 1))
  if awk -v v="${pct}" -v t="${THRESHOLD}" 'BEGIN { exit (v + 0 < t + 0) ? 0 : 1 }'; then
    echo "FAIL: ${pkg} coverage ${pct}% is below ${THRESHOLD}% threshold"
    failed=1
  else
    echo "PASS: ${pkg} coverage ${pct}%"
  fi
done

echo "----------------------------------------"
if [ "${checked}" -eq 0 ]; then
  echo "ERROR: no internal/ packages with tests were evaluated"
  exit 1
fi

if [ "${failed}" -ne 0 ]; then
  echo "FAIL: one or more internal/ packages are below ${THRESHOLD}% coverage"
  exit 1
fi

echo "PASS: all ${checked} internal/ packages meet the ${THRESHOLD}% per-package threshold"
