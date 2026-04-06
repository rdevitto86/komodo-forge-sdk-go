#!/usr/bin/env bash
# generate.sh — run go generate across all packages.
#
# If a tools.go file is present at the repo root, any `go install` directives
# declared in it are executed first so codegen tools are available on PATH.
#
# Usage:
#   ./scripts/generate.sh [--help]
#
# Environment variables:
#   GENERATE_PKGS   Package pattern passed to go generate  (default: ./...)
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
  echo "Runs go generate ./... to execute all //go:generate directives."
  echo ""
  echo "If a tools.go file exists at the repo root, tools declared in it are"
  echo "installed via 'go install' before generation runs."
  echo ""
  echo "Environment variables:"
  echo "  GENERATE_PKGS   Package pattern for go generate  (default: ./...)"
  echo ""
  echo "Exit codes:"
  echo "  0  Generation completed successfully"
  echo "  1  Generation failed"
  exit 0
fi

check_go_version 1 21

ROOT="$(repo_root)"
cd "$ROOT"

GENERATE_PKGS="${GENERATE_PKGS:-./...}"

# ---------------------------------------------------------------------------
# Install tools declared in tools.go (if present)
# ---------------------------------------------------------------------------
TOOLS_FILE="${ROOT}/tools.go"

if [[ -f "$TOOLS_FILE" ]]; then
  log_section "Installing codegen tools from tools.go"

  # Extract import paths from tools.go (lines with _ "..." imports)
  TOOL_IMPORTS=$(grep -E '^\s+_ "' "$TOOLS_FILE" | sed 's/.*_ "//;s/".*//' || true)

  if [[ -n "$TOOL_IMPORTS" ]]; then
    while IFS= read -r pkg; do
      log_info "Installing ${pkg}..."
      go install "${pkg}@latest"
    done <<< "$TOOL_IMPORTS"
    log_success "Tools installed."
  else
    log_warn "tools.go found but no blank-import tool declarations detected."
  fi
else
  log_info "No tools.go found — skipping tool installation."
fi

# ---------------------------------------------------------------------------
# Snapshot file mtimes before generation
# ---------------------------------------------------------------------------
BEFORE_SNAPSHOT=$(find . -name "*.go" ! -path "*/vendor/*" ! -path "*/.git/*" \
  -newer "$SCRIPT_DIR/generate.sh" 2>/dev/null | sort || true)

# ---------------------------------------------------------------------------
# Run go generate
# ---------------------------------------------------------------------------
log_section "Running go generate"
log_info "Packages: ${GENERATE_PKGS}"

set +e
go generate "${GENERATE_PKGS}" 2>&1
GEN_EXIT=$?
set -e

if [[ "$GEN_EXIT" -ne 0 ]]; then
  log_error "go generate failed — see output above."
  exit 1
fi

# ---------------------------------------------------------------------------
# Report modified files
# ---------------------------------------------------------------------------
AFTER_SNAPSHOT=$(find . -name "*.go" ! -path "*/vendor/*" ! -path "*/.git/*" \
  -newer "$SCRIPT_DIR/generate.sh" 2>/dev/null | sort || true)

MODIFIED=$(comm -13 \
  <(echo "$BEFORE_SNAPSHOT") \
  <(echo "$AFTER_SNAPSHOT") 2>/dev/null || true)

if [[ -n "$MODIFIED" ]]; then
  log_info "Files modified by go generate:"
  echo "$MODIFIED" | sed 's/^/  /'
else
  log_info "No files were modified."
fi

log_success "go generate completed."
