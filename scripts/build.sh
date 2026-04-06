#!/usr/bin/env bash
# build.sh — compile the Go binary with version metadata injected via ldflags.
#
# Usage:
#   ./scripts/build.sh [--help]
#
# Environment variables:
#   OUTPUT      Output binary path          (default: ./bin/app)
#   PKG_MAIN    Package containing main()   (default: .)
#   GOOS        Target OS for cross-compile (default: current OS)
#   GOARCH      Target arch                 (default: current arch)
#   VERSION     Version string              (default: git tag, or "dev")
#   LDFLAGS_PKG Import path for ldflags     (default: auto-detected from go.mod)
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
  echo "Compiles the Go binary with version, commit, and build-time ldflags."
  echo ""
  echo "Environment variables:"
  echo "  OUTPUT        Output binary path          (default: ./bin/app)"
  echo "  PKG_MAIN      Package containing main()   (default: .)"
  echo "  GOOS          Target OS for cross-compile (default: host OS)"
  echo "  GOARCH        Target arch                 (default: host arch)"
  echo "  VERSION       Version string              (default: git tag or 'dev')"
  echo "  LDFLAGS_PKG   Import path for ldflags     (default: from go.mod)"
  echo ""
  echo "Exit codes:"
  echo "  0  Build succeeded"
  echo "  1  Build failed"
  exit 0
fi

check_go_version 1 21

ROOT="$(repo_root)"
cd "$ROOT"

# ---------------------------------------------------------------------------
# Resolve config
# ---------------------------------------------------------------------------
OUTPUT="${OUTPUT:-./bin/app}"
PKG_MAIN="${PKG_MAIN:-.}"

VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}"
COMMIT="${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")}"
BUILD_TIME="${BUILD_TIME:-$(date -u +"%Y-%m-%dT%H:%M:%SZ")}"

# Detect the module import path for ldflags from go.mod
if [[ -z "${LDFLAGS_PKG:-}" ]]; then
  LDFLAGS_PKG=$(grep -E '^module ' go.mod | awk '{print $2}')
fi

# ---------------------------------------------------------------------------
# Build
# ---------------------------------------------------------------------------
log_section "Building"
log_info "Output:     ${OUTPUT}"
log_info "Package:    ${PKG_MAIN}"
log_info "Version:    ${VERSION}"
log_info "Commit:     ${COMMIT}"
log_info "Build time: ${BUILD_TIME}"
[[ -n "${GOOS:-}" ]]   && log_info "GOOS:       ${GOOS}"
[[ -n "${GOARCH:-}" ]] && log_info "GOARCH:     ${GOARCH}"

mkdir -p "$(dirname "$OUTPUT")"

LD_FLAGS="-X '${LDFLAGS_PKG}/version.Version=${VERSION}'"
LD_FLAGS="${LD_FLAGS} -X '${LDFLAGS_PKG}/version.Commit=${COMMIT}'"
LD_FLAGS="${LD_FLAGS} -X '${LDFLAGS_PKG}/version.BuildTime=${BUILD_TIME}'"
LD_FLAGS="${LD_FLAGS} -s -w"

go build \
  -ldflags "${LD_FLAGS}" \
  -o "${OUTPUT}" \
  "${PKG_MAIN}"

log_success "Binary written to ${OUTPUT}"
