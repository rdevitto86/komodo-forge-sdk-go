#!/usr/bin/env bash
# coverage.sh — run tests with coverage profiling, enforce a minimum threshold,
#               print per-package breakdown, and generate an HTML report.
#
# Usage:
#   ./scripts/coverage.sh [--help]
#
# Environment variables:
#   COVERAGE_THRESHOLD   Minimum total coverage % required  (default: 80)
#   COVERAGE_OUT         Path for the coverage profile      (default: coverage.out)
#   COVERAGE_HTML        Path for the HTML report           (default: coverage.html)
#   TEST_PKGS            Package pattern                    (default: ./...)
#   TEST_TIMEOUT         Per-package timeout                (default: 5m)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=_lib.sh
source "${SCRIPT_DIR}/_lib.sh"

# ---------------------------------------------------------------------------
# Help
# ---------------------------------------------------------------------------
if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  echo "Usage: $(basename "$0") [--help]"
  echo ""
  echo "Runs tests with coverage, prints per-package breakdown, enforces a"
  echo "minimum total coverage threshold, and writes an HTML report."
  echo ""
  echo "Environment variables:"
  echo "  COVERAGE_THRESHOLD   Minimum total coverage % required  (default: 80)"
  echo "  COVERAGE_OUT         Coverage profile output path        (default: coverage.out)"
  echo "  COVERAGE_HTML        HTML report output path             (default: coverage.html)"
  echo "  TEST_PKGS            Package pattern                     (default: ./...)"
  echo "  TEST_TIMEOUT         Per-package timeout                 (default: 5m)"
  echo ""
  echo "Exit codes:"
  echo "  0  Tests passed and coverage threshold met"
  echo "  1  Tests failed or coverage below threshold"
  exit 0
fi

check_go_version 1 21

ROOT="$(repo_root)"
cd "$ROOT"

COVERAGE_THRESHOLD="${COVERAGE_THRESHOLD:-80}"
COVERAGE_OUT="${COVERAGE_OUT:-coverage.out}"
COVERAGE_HTML="${COVERAGE_HTML:-coverage.html}"
TEST_PKGS="${TEST_PKGS:-./...}"
TEST_TIMEOUT="${TEST_TIMEOUT:-5m}"

# ---------------------------------------------------------------------------
# Run tests with coverage
# ---------------------------------------------------------------------------
log_section "Running tests with coverage"
log_info "Packages:  ${TEST_PKGS}"
log_info "Threshold: ${COVERAGE_THRESHOLD}%"

set +e
go test \
  -race \
  -timeout "${TEST_TIMEOUT}" \
  -count=1 \
  -coverprofile="${COVERAGE_OUT}" \
  -covermode=atomic \
  "${TEST_PKGS}" 2>&1
TEST_EXIT=$?
set -e

if [[ "$TEST_EXIT" -ne 0 ]]; then
  log_error "Tests failed — coverage report not generated."
  exit 1
fi

if [[ ! -f "$COVERAGE_OUT" ]]; then
  log_error "Coverage profile not written — no testable packages found?"
  exit 1
fi

# ---------------------------------------------------------------------------
# Per-package breakdown
# ---------------------------------------------------------------------------
log_section "Per-package coverage"
go tool cover -func="${COVERAGE_OUT}" | grep -v "^total:" | \
  awk '{printf "  %-70s %s\n", $1, $3}' || true

# ---------------------------------------------------------------------------
# Total coverage
# ---------------------------------------------------------------------------
TOTAL=$(go tool cover -func="${COVERAGE_OUT}" | grep "^total:" | awk '{print $3}' | sed 's/%//')

echo ""
log_info "Total coverage: ${TOTAL}%  (threshold: ${COVERAGE_THRESHOLD}%)"

# ---------------------------------------------------------------------------
# HTML report
# ---------------------------------------------------------------------------
go tool cover -html="${COVERAGE_OUT}" -o "${COVERAGE_HTML}"
log_info "HTML report written to ${COVERAGE_HTML}"

# ---------------------------------------------------------------------------
# Threshold check
# ---------------------------------------------------------------------------
BELOW=$(awk -v total="$TOTAL" -v threshold="$COVERAGE_THRESHOLD" \
  'BEGIN { print (total+0 < threshold+0) ? "1" : "0" }')

if [[ "$BELOW" == "1" ]]; then
  log_error "Coverage ${TOTAL}% is below the required ${COVERAGE_THRESHOLD}%."
  exit 1
fi

log_success "Coverage ${TOTAL}% meets the required ${COVERAGE_THRESHOLD}%."
