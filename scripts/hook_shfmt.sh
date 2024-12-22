#!/usr/bin/env bash
# shellcheck source=SCRIPTDIR/lib/setup.sh
source "$(dirname "${BASH_SOURCE[0]}")/lib/setup.sh"

log_info "Using shfmt version: $(shfmt --version)"

shfmt --write --diff "$@" >/dev/null
