#!/usr/bin/env bash
# shellcheck source=SCRIPTDIR/lib/setup.sh
source "$(dirname "${BASH_SOURCE[0]}")/lib/setup.sh"

command -v terraform >/dev/null 2>&1 || log_fatal "Ensure terraform is installed."

log_info "Using terraform version: $(terraform version)"

module_root="${1?"Specify the path to the Terraform root module on which to act."}"
action="${2?"Specify the Terraform command to run (apply | plan)."}"

cd "${module_root}"
module_name="$(basename "$(pwd)")"

terraform init

tf_flags=()
case "${action}" in
apply)
  tf_flags+=(
    "-auto-approve"
    "-input=false"
    "${REPO_ROOT}/${module_name}.tfplan"
  )
  ;;
plan)
  tf_flags+=(
    "-detailed-exitcode"
    "-input=false"
    "-out=${REPO_ROOT}/${module_name}.tfplan"
  )
  ;;
*)
  log_fatal "Unsupport Terraform command '${action}'."
  ;;
esac

exit_code=0
terraform "${action}" "${tf_flags[@]}" || exit_code=$?

if [[ ${action} == "plan" ]]; then
  case "${exit_code}" in
  0)
    log_info "Terraform planning did not detect any changes."
    ;;
  1)
    log_fatal "Terraform planning failed. See detailed output above for more information."
    ;;
  2)
    log_info "Terraform planning detected unapplied changes. Once this PR is merged these will be applied."
    [[ ${CI:-} == "true" ]] && echo "changes=true" >>"${GITHUB_OUTPUT}"
    exit_code=0
    ;;
  *)
    log_fatal "Unexpected exit-code for 'terraform plan -detailed-exitcode': ${exit_code}"
    ;;
  esac
fi

exit "${exit_code}"
