#!/usr/bin/env bash
(return 0 2>/dev/null) || {
  echo "Script is only intended to be source'd, not executed directly."
  exit 1
}

set -e -u -o pipefail
[[ -n "${DEBUG:-}" ]] && set -x

# Source all Bash helper scripts.
lib_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${lib_dir}/log.sh"
source "${lib_dir}/git.sh"

# Ensure a script sourcing setup.sh will always run starting with the repo root as working directory.
cd "${lib_dir}/../.."
REPO_ROOT="$(pwd)"
export REPO_ROOT

unset lib_dir
