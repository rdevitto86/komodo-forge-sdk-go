#!/usr/bin/env bash
# test.sh — run the full test suite with the race detector enabled.
#
# Usage:
#   ./scripts/test.sh [--help]
#
# Environment variables:
#   TEST_FLAGS    Extra flags passed directly to `go test`
#                 (e.g. TEST_FLAGS="-short" or TEST_FLAGS="-run TestFoo")
#   TEST_TIMEOUT  Per-package test timeout   (default: 5m)
#   TEST_PKGS     Package pattern to test    (default: ./...)
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
  echo "Runs go test ./... with the race detector."
  echo ""
  echo "Environment variables:"
  echo "  TEST_FLAGS    Extra flags for go test  (e.g. '-short', '-run TestFoo')"
  echo "  TEST_TIMEOUT  Per-package timeout       (default: 5m)"
  echo "  TEST_PKGS     Package pattern           (default: ./...)"
  echo ""
  echo "Exit codes:"
  echo "  0  All tests passed"
  echo "  1  One or more tests failed"
  exit 0
fi

check_go_version 1 21

ROOT="$(repo_root)"
cd "$ROOT"

TEST_TIMEOUT="${TEST_TIMEOUT:-5m}"
TEST_PKGS="${TEST_PKGS:-./...}"
# shellcheck disable=SC2086
TEST_FLAGS="${TEST_FLAGS:-}"

# ---------------------------------------------------------------------------
# Run
# ---------------------------------------------------------------------------
log_section "Running tests"
log_info "Packages: ${TEST_PKGS}"
log_info "Timeout:  ${TEST_TIMEOUT}"
[[ -n "$TEST_FLAGS" ]] && log_info "Flags:    ${TEST_FLAGS}"

set +e
# shellcheck disable=SC2086
go test \
  -race \
  -timeout "${TEST_TIMEOUT}" \
  -count=1 \
  ${TEST_FLAGS} \
  "${TEST_PKGS}" \
  2>&1 | tee /tmp/moxtox_test_output.txt
EXIT_CODE=${PIPESTATUS[0]}
set -e

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
echo ""
PASSED=$(grep -c "^--- PASS"  /tmp/moxtox_test_output.txt 2>/dev/null || true)
FAILED=$(grep -c "^--- FAIL"  /tmp/moxtox_test_output.txt 2>/dev/null || true)
SKIPPED=$(grep -c "^--- SKIP" /tmp/moxtox_test_output.txt 2>/dev/null || true)

log_section "Test summary"
log_info "Passed:  ${PASSED}"
log_info "Skipped: ${SKIPPED}"

if [[ "$EXIT_CODE" -ne 0 ]]; then
  log_error "Failed:  ${FAILED}"
  log_error "Tests FAILED — see output above."
  exit 1
fi

log_success "All tests passed."
