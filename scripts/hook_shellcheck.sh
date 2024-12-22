#!/usr/bin/env bash
# shellcheck source=SCRIPTDIR/lib/setup.sh
source "$(dirname "${BASH_SOURCE[0]}")/lib/setup.sh"

log_info "Using shellcheck version: $(shellcheck --version | grep version: | tail -c+10)"

shellcheck "$@"
