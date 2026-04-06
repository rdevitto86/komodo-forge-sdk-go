#!/usr/bin/env bash
# _lib.sh — shared utilities sourced by all scripts in this directory.
# Do not execute directly.
set -euo pipefail

# ---------------------------------------------------------------------------
# Colors (suppressed in CI)
# ---------------------------------------------------------------------------
if [[ "${CI:-false}" == "true" ]]; then
  RED="" GREEN="" YELLOW="" BLUE="" BOLD="" RESET=""
else
  RED='\033[0;31m'
  GREEN='\033[0;32m'
  YELLOW='\033[1;33m'
  BLUE='\033[0;34m'
  BOLD='\033[1m'
  RESET='\033[0m'
fi

# ---------------------------------------------------------------------------
# Logging
# ---------------------------------------------------------------------------
log_info()    { echo -e "${BLUE}[INFO]${RESET}    $*"; }
log_success() { echo -e "${GREEN}[OK]${RESET}      $*"; }
log_warn()    { echo -e "${YELLOW}[WARN]${RESET}    $*"; }
log_error()   { echo -e "${RED}[ERROR]${RESET}   $*" >&2; }
log_section() { echo -e "\n${BOLD}==> $*${RESET}"; }

# ---------------------------------------------------------------------------
# require_tool <name> <install-command>
# Checks whether <name> is on PATH; if not, runs <install-command>.
# Exits non-zero if the tool is still missing after install.
# ---------------------------------------------------------------------------
require_tool() {
  local name="$1"
  local install_cmd="$2"

  if command -v "$name" &>/dev/null; then
    return 0
  fi

  log_warn "'$name' not found on PATH — installing..."

  if [[ "${CI:-false}" == "true" ]]; then
    # Non-interactive install in CI
    eval "$install_cmd" 2>&1
  else
    eval "$install_cmd"
  fi

  if ! command -v "$name" &>/dev/null; then
    log_error "Failed to install '$name'. Please install it manually and re-run."
    exit 1
  fi

  log_success "'$name' installed successfully."
}

# ---------------------------------------------------------------------------
# check_go_version <min_major> <min_minor>
# Example: check_go_version 1 21  →  requires Go 1.21+
# ---------------------------------------------------------------------------
check_go_version() {
  local min_major="$1"
  local min_minor="$2"

  if ! command -v go &>/dev/null; then
    log_error "Go is not installed or not on PATH."
    exit 1
  fi

  local version_string
  version_string=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | head -1 | sed 's/go//')

  local cur_major cur_minor
  cur_major=$(echo "$version_string" | cut -d. -f1)
  cur_minor=$(echo "$version_string" | cut -d. -f2)

  if [[ "$cur_major" -lt "$min_major" ]] || \
     { [[ "$cur_major" -eq "$min_major" ]] && [[ "$cur_minor" -lt "$min_minor" ]]; }; then
    log_error "Go ${min_major}.${min_minor}+ is required; found ${version_string}."
    exit 1
  fi
}

# ---------------------------------------------------------------------------
# repo_root
# Prints the absolute path of the git repository root.
# ---------------------------------------------------------------------------
repo_root() {
  git rev-parse --show-toplevel 2>/dev/null || pwd
}
