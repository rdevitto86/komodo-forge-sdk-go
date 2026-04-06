#!/usr/bin/env bash
# audit.sh — run go mod verify, go vet, and govulncheck.
#
# Usage:
#   ./scripts/audit.sh [--help]
#
# Environment variables:
#   SKIP_VULNCHECK   Set to "true" to skip govulncheck (e.g. offline environments)
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
  echo "Runs three checks in sequence:"
  echo "  1. go mod verify     — verifies module download integrity"
  echo "  2. go vet ./...      — reports likely mistakes in packages"
  echo "  3. govulncheck ./... — reports known vulnerabilities in dependencies"
  echo ""
  echo "Environment variables:"
  echo "  SKIP_VULNCHECK   Set to 'true' to skip govulncheck (default: false)"
  echo ""
  echo "Exit codes:"
  echo "  0  All checks passed"
  echo "  1  One or more checks failed"
  exit 0
fi

check_go_version 1 21

ROOT="$(repo_root)"
cd "$ROOT"

SKIP_VULNCHECK="${SKIP_VULNCHECK:-false}"
OVERALL_EXIT=0

# ---------------------------------------------------------------------------
# 1. go mod verify
# ---------------------------------------------------------------------------
log_section "go mod verify"

set +e
go mod verify 2>&1
MOD_EXIT=$?
set -e

if [[ "$MOD_EXIT" -ne 0 ]]; then
  log_error "go mod verify failed — module checksums do not match."
  OVERALL_EXIT=1
else
  log_success "Module checksums verified."
fi

# ---------------------------------------------------------------------------
# 2. go vet
# ---------------------------------------------------------------------------
log_section "go vet"

set +e
go vet ./... 2>&1
VET_EXIT=$?
set -e

if [[ "$VET_EXIT" -ne 0 ]]; then
  log_error "go vet reported issues — see output above."
  OVERALL_EXIT=1
else
  log_success "go vet passed."
fi

# ---------------------------------------------------------------------------
# 3. govulncheck
# ---------------------------------------------------------------------------
if [[ "$SKIP_VULNCHECK" == "true" ]]; then
  log_warn "Skipping govulncheck (SKIP_VULNCHECK=true)."
else
  log_section "govulncheck"

  require_tool govulncheck "go install golang.org/x/vuln/cmd/govulncheck@latest"

  set +e
  govulncheck ./... 2>&1
  VULN_EXIT=$?
  set -e

  if [[ "$VULN_EXIT" -ne 0 ]]; then
    log_error "govulncheck found vulnerabilities — see output above."
    OVERALL_EXIT=1
  else
    log_success "No known vulnerabilities found."
  fi
fi

# ---------------------------------------------------------------------------
# Final result
# ---------------------------------------------------------------------------
echo ""
if [[ "$OVERALL_EXIT" -ne 0 ]]; then
  log_error "Audit FAILED — one or more checks did not pass."
  exit 1
fi

log_success "All audit checks passed."
