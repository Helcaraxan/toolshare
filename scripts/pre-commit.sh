#!/usr/bin/env bash
# shellcheck source=SCRIPTDIR/lib/setup.sh
source "$(dirname "${BASH_SOURCE[0]}")/lib/setup.sh"

command -v pre-commit >/dev/null 2>&1 || log_fatal "Ensure pre-commit is installed."

log_info "Using pre-commit version: $(pre-commit --version)"

pc_flags=(
  "--color=always"
)
if [[ -n "${DRIFT_CHECK:-}" ]]; then
  pc_flags+=("--all-files")
  log_warning "Running on full file content for drift check. This may take some extra time."
else
  pc_flags+=(
    "--from-ref=$(git_base_ref)"
    "--to-ref=HEAD"
  )
fi

log_info "Running pre-commit run"
pre-commit run "${pc_flags[@]}"
