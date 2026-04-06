#!/usr/bin/env bash
# lint.sh — run golangci-lint, installing it if not found on PATH.
#
# Usage:
#   ./scripts/lint.sh [--help]
#
# Environment variables:
#   GOLANGCI_LINT_VERSION   Version to install if missing  (default: latest)
#   LINT_FLAGS              Extra flags passed to golangci-lint run
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
  echo "Runs golangci-lint against ./... — installs the linter if not found."
  echo "Respects .golangci.yml or .golangci.yaml if present at the repo root."
  echo ""
  echo "Environment variables:"
  echo "  GOLANGCI_LINT_VERSION   Version to install if missing  (default: latest)"
  echo "  LINT_FLAGS              Extra flags for golangci-lint run"
  echo ""
  echo "Exit codes:"
  echo "  0  No lint findings"
  echo "  1  Lint findings found, or linter failed to run"
  exit 0
fi

check_go_version 1 21

ROOT="$(repo_root)"
cd "$ROOT"

GOLANGCI_LINT_VERSION="${GOLANGCI_LINT_VERSION:-latest}"
LINT_FLAGS="${LINT_FLAGS:-}"

# ---------------------------------------------------------------------------
# Install golangci-lint if missing
# ---------------------------------------------------------------------------
INSTALL_CMD="curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
  | sh -s -- -b \"\$(go env GOPATH)/bin\" ${GOLANGCI_LINT_VERSION}"

require_tool golangci-lint "$INSTALL_CMD"

# ---------------------------------------------------------------------------
# Detect config file
# ---------------------------------------------------------------------------
CONFIG_FLAG=""
for cfg in .golangci.yml .golangci.yaml .golangci.toml .golangci.json; do
  if [[ -f "${ROOT}/${cfg}" ]]; then
    CONFIG_FLAG="--config ${cfg}"
    log_info "Using lint config: ${cfg}"
    break
  fi
done

# ---------------------------------------------------------------------------
# Run
# ---------------------------------------------------------------------------
log_section "Running golangci-lint"

set +e
# shellcheck disable=SC2086
golangci-lint run ${CONFIG_FLAG} ${LINT_FLAGS} ./...
EXIT_CODE=$?
set -e

if [[ "$EXIT_CODE" -ne 0 ]]; then
  log_error "Lint findings detected — see output above."
  exit 1
fi

log_success "No lint findings."
