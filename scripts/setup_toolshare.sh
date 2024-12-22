#!/usr/bin/env bash

# This script deliberately does not source lib/setup.sh as it is also used by
# .envrc and we do not want to pollute the shell environment by default.
set -e -u -o pipefail

function setup_toolshare() {
  local -l gobin preceding_tag toolshare_bin

  gobin="$(go env GOPATH)/bin"
  subscriptions_path="${XDG_CONFIG_HOME:-"${HOME}/.config"}/toolshare/subscriptions"

  if [[ ${GITHUB_ACTIONS:-} == "true" ]]; then
    echo "${gobin}" >>"${GITHUB_PATH}"
    echo "${subscriptions_path}" >>"${GITHUB_PATH}"

    echo "Installing toolshare from HEAD for CI purposes."
    go build -o "${gobin}/toolshare" .
  else
    export PATH="${subscriptions_path}:${gobin}:${PATH}"

    preceding_tag="$(git tag --list --sort=-v:refname --merged | head --lines=1)"
    toolshare_bin="$(command -v toolshare || true)"

    if (
      [[ -z ${toolshare_bin} ]] ||
        [[ ${toolshare_bin} != "$(go env GOPATH)/bin/toolshare" ]] ||
        ! go version -m "${toolshare_bin}" | grep --basic-regexp --quiet $'mod\tgithub.com/Helcaraxan/toolshare\t'"${preceding_tag}"
    ); then
      echo "Installing toolshare@${preceding_tag} to use the last release preceding the current HEAD. This may take some time."
      go install "github.com/Helcaraxan/toolshare@${preceding_tag}"
    fi
  fi

  toolshare sync --mode=shim
}

setup_toolshare
