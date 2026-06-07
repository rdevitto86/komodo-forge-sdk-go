#!/usr/bin/env bash
set -euo pipefail

# Format all staged Go files and re-stage them.
go_files=$(git diff --name-only --cached --diff-filter=ACM | grep '\.go$' || true)
if [ -n "$go_files" ]; then
    gofmt -w $go_files
    git add $go_files
fi
