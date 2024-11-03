#!/usr/bin/env bash
(return 0 2>/dev/null) || {
  echo "Script is only intended to be source'd, not executed directly."
  exit 1
}
if [[ -z "${lib_dir:-}" ]] || ((${#BASH_SOURCE[@]} < 2)) || ! [[ "${BASH_SOURCE[1]}" -ef "${lib_dir}/setup.sh" ]]; then
  echo "This script should only be source'd automatically by source'ing the adjacent 'setup.sh' file."
  exit 1
fi

function log_info() {
  echo -e "\033[0;34m$*\033[0m"
}

function log_success() {
  echo -e "\033[0;32m$*\033[0m"
}

function log_warning() {
  echo -e "\033[0;33m$*\033[0m"
}

function log_error() {
  echo -e "\033[0;31m$*\033[0m"
}

function log_fatal() {
  log_error "$@"
  exit 1
}

function output_on_fail() {
  local -r action="$1"
  log_info "Running ${action}."
  shift
  if ! out="$("$@" 2>&1)"; then
    log_error "Failed to run ${action}. See output below."
    log_fatal "${out}"
  fi
}
