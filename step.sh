#!/bin/bash

THIS_SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
# load bash utils
source "${THIS_SCRIPT_DIR}/bash_utils/utils.sh"
source "${THIS_SCRIPT_DIR}/bash_utils/formatted_output.sh"


# ------------------------------
# --- Error Cleanup

function finalcleanup {
  echo "-> finalcleanup"
  local fail_msg="$1"

  write_section_to_formatted_output "# Error"
  if [ ! -z "${fail_msg}" ] ; then
    write_section_to_formatted_output "**Error Description**:"
    write_section_to_formatted_output "${fail_msg}"
  fi
  write_section_to_formatted_output "*See the logs for more information*"

  write_section_to_formatted_output "**If this is the very first build**
of the app you try to deploy to iTunes Connect then you might want to upload the first
build manually to make sure it fulfills the initial iTunes Connect submission
verification process."

  if [[ "${STEP_DELIVER_DEPLOY_IS_SUBMIT_FOR_BETA}" == "yes" ]] ; then
	write_section_to_formatted_output "**Beta deply note:** you
should try to disable the \`Submit for TestFlight Beta Testing\` option and try
the deploy again."
  fi
}

function CLEANUP_ON_ERROR_FN {
  local err_msg="$1"
  finalcleanup "${err_msg}"
}
set_error_cleanup_function CLEANUP_ON_ERROR_FN


# ---------------------
# --- Required Inputs

if [ -z "${STEP_DELIVER_DEPLOY_IPA_PATH}" ] ; then
	echo " [!] \`STEP_DELIVER_DEPLOY_IPA_PATH\` not provided!"
	exit 1
fi

if [ -z "${STEP_DELIVER_DEPLOY_ITUNESCON_PASSWORD}" ] ; then
	echo " [!] \`STEP_DELIVER_DEPLOY_ITUNESCON_PASSWORD\` not provided!"
	exit 1
fi

if [ -z "${STEP_DELIVER_DEPLOY_ITUNESCON_USER}" ] ; then
	echo " [!] \`STEP_DELIVER_DEPLOY_ITUNESCON_USER\` not provided!"
	exit 1
fi

if [ -z "${STEP_DELIVER_DEPLOY_ITUNESCON_APP_ID}" ] ; then
	echo " [!] \`STEP_DELIVER_DEPLOY_ITUNESCON_APP_ID\` not provided!"
	exit 1
fi

CONFIG_testflight_beta_deploy_type_flag='--skip-deploy'
if [[ "${STEP_DELIVER_DEPLOY_IS_SUBMIT_FOR_BETA}" == "yes" ]] ; then
	CONFIG_testflight_beta_deploy_type_flag='--beta'
fi

echo " (i) TestFlight beta deploy type flag: ${CONFIG_testflight_beta_deploy_type_flag}"


# ---------------------
# --- Main

write_section_to_formatted_output "# Setup"
bash "${THIS_SCRIPT_DIR}/_setup.sh"
fail_if_cmd_error "Failed to setup the required tools!"

write_section_to_formatted_output "# Deploy"

write_section_to_formatted_output "**Note:** if your password
contains special characters
and you experience problems, please
consider changing your password
to something with only
alphanumeric characters."


export DELIVER_USER="${STEP_DELIVER_DEPLOY_ITUNESCON_USER}"
export DELIVER_PASSWORD="${STEP_DELIVER_DEPLOY_ITUNESCON_PASSWORD}"
deliver testflight ${CONFIG_testflight_beta_deploy_type_flag} -a "${STEP_DELIVER_DEPLOY_ITUNESCON_APP_ID}" "${STEP_DELIVER_DEPLOY_IPA_PATH}"
fail_if_cmd_error "Deploy failed!"

write_section_to_formatted_output "# Success"
echo_string_to_formatted_output "* The app (.ipa) was successfully uploaded to [iTunes Connect](https://itunesconnect.apple.com), you should see it in the *Prerelease* section on the app's iTunes Connect page!"
if [[ "${STEP_DELIVER_DEPLOY_IS_SUBMIT_FOR_BETA}" != "yes" ]] ; then
	echo_string_to_formatted_output "* **Don't forget to enable** the **TestFlight Beta Testing** switch on iTunes Connect (on the *Prerelease* tab of the app) if this is a new version of the app!"
fi

exit 0
