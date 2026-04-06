#!/usr/bin/env bash
# format.sh — run gofmt and goimports across all Go source files.
#
# In CI mode (CI=true), runs in check-only mode and exits non-zero if any
# files would be changed, without modifying them.
#
# Usage:
#   ./scripts/format.sh [--help]
#
# Environment variables:
#   CI   When set to "true", check-only mode — no files are modified
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
  echo "Formats all Go source files using gofmt and goimports."
  echo ""
  echo "Local mode:  rewrites files in place."
  echo "CI mode:     check-only — exits non-zero if any files need formatting."
  echo ""
  echo "Environment variables:"
  echo "  CI   Set to 'true' to enable check-only mode (default: false)"
  echo ""
  echo "Exit codes:"
  echo "  0  All files are correctly formatted (or were successfully formatted)"
  echo "  1  Files need formatting (CI mode), or a tool failed"
  exit 0
fi

check_go_version 1 21

ROOT="$(repo_root)"
cd "$ROOT"

require_tool goimports "go install golang.org/x/tools/cmd/goimports@latest"

CI_MODE="${CI:-false}"
OVERALL_EXIT=0

# ---------------------------------------------------------------------------
# Collect Go files (exclude vendor if present)
# ---------------------------------------------------------------------------
GO_FILES=$(find . -name "*.go" \
  ! -path "*/vendor/*" \
  ! -path "*/.git/*" \
  2>/dev/null | sort)

if [[ -z "$GO_FILES" ]]; then
  log_warn "No Go files found."
  exit 0
fi

# ---------------------------------------------------------------------------
# gofmt
# ---------------------------------------------------------------------------
log_section "gofmt"

if [[ "$CI_MODE" == "true" ]]; then
  # shellcheck disable=SC2086
  UNFORMATTED=$(echo "$GO_FILES" | xargs gofmt -l 2>/dev/null || true)
  if [[ -n "$UNFORMATTED" ]]; then
    log_error "The following files need gofmt formatting:"
    echo "$UNFORMATTED" | sed 's/^/  /'
    OVERALL_EXIT=1
  else
    log_success "All files pass gofmt."
  fi
else
  # shellcheck disable=SC2086
  echo "$GO_FILES" | xargs gofmt -w
  log_success "gofmt applied."
fi

# ---------------------------------------------------------------------------
# goimports
# ---------------------------------------------------------------------------
log_section "goimports"

if [[ "$CI_MODE" == "true" ]]; then
  # shellcheck disable=SC2086
  UNFORMATTED=$(echo "$GO_FILES" | xargs goimports -l 2>/dev/null || true)
  if [[ -n "$UNFORMATTED" ]]; then
    log_error "The following files need goimports formatting:"
    echo "$UNFORMATTED" | sed 's/^/  /'
    OVERALL_EXIT=1
  else
    log_success "All files pass goimports."
  fi
else
  # shellcheck disable=SC2086
  echo "$GO_FILES" | xargs goimports -w
  log_success "goimports applied."
fi

# ---------------------------------------------------------------------------
# Result
# ---------------------------------------------------------------------------
echo ""
if [[ "$OVERALL_EXIT" -ne 0 ]]; then
  log_error "Formatting check FAILED — run './scripts/format.sh' locally to fix."
  exit 1
fi

if [[ "$CI_MODE" == "true" ]]; then
  log_success "All files are correctly formatted."
fi
