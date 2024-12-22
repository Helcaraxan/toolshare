#!/usr/bin/env bash
(return 0 2>/dev/null) || {
  echo "Script is only intended to be source'd, not executed directly."
  exit 1
}
if [[ -z "${lib_dir:-}" ]] || ((${#BASH_SOURCE[@]} < 2)) || ! [[ "${BASH_SOURCE[1]}" -ef "${lib_dir}/setup.sh" ]]; then
  echo "This script should only be source'd automatically by source'ing the adjacent 'setup.sh' file."
  exit 1
fi

# Returns the SHA of the commit with respect to which any current changes should be computed.
#  - In the context of a GitHub Pull Request this will be the head of the target branch.
#  - On the default branch this will be the previous commit (assuming trunk-based development).
#  - On a non-default branch this will be the merge-base with the default branch.
function git_base_ref() {
  local current_branch default_branch
  if [[ "${GITHUB_ACTIONS:-}" == "true" ]] && [[ -n "${GITHUB_BASE_REF:-}" ]]; then
    git rev-parse "${GITHUB_BASE_REF}"
    return
  fi

  git fetch --all >/dev/null
  default_branch="$(git symbolic-ref "refs/remotes/${GIT_REMOTE_NAME:-origin}/HEAD")"
  default_branch="${default_branch#"refs/remotes/${GIT_REMOTE_NAME:-origin}/"}"
  current_branch="${GITHUB_REF_NAME:-"$(git rev-parse --abbrev-ref HEAD)"}"

  if [[ "${current_branch}" == "${default_branch}" ]]; then
    git rev-parse "${default_branch}~1"
  else
    git rev-parse "${default_branch}"
  fi
}
