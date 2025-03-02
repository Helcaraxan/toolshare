#!/usr/bin/env bash
# shellcheck source=SCRIPTDIR/lib/setup.sh
source "$(dirname "${BASH_SOURCE[0]}")/lib/setup.sh"

command -v golangci-lint >/dev/null 2>&1 || log_fatal "Ensure golangci-lint is installed."

log_info "$(golangci-lint --version)"

golangci-lint config verify

golangci-lint run --new-from-rev="$(git_base_ref)" --fix
