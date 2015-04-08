#!/bin/bash

THIS_SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
# load bash utils
source "${THIS_SCRIPT_DIR}/bash_utils/utils.sh"
source "${THIS_SCRIPT_DIR}/bash_utils/formatted_output.sh"

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
export DELIVER_USER="${STEP_DELIVER_DEPLOY_ITUNESCON_USER}"
export DELIVER_PASSWORD="${STEP_DELIVER_DEPLOY_ITUNESCON_PASSWORD}"
deliver testflight ${CONFIG_testflight_beta_deploy_type_flag} -a "${STEP_DELIVER_DEPLOY_ITUNESCON_APP_ID}" "${STEP_DELIVER_DEPLOY_IPA_PATH}"
fail_if_cmd_error "Deploy failed!"

write_section_to_formatted_output "# Success"
echo_string_to_formatted_output "* The app (.ipa) was successfully uploaded to [iTunes Connect](https://itunesconnect.apple.com), you should see it in the *Prerelease* section on the app's iTunes Connect page!"
echo_string_to_formatted_output "* Don't forget to enable the **TestFlight Beta Testing** switch on iTunes Connect (on the *Prerelease* tab of the app) if this is a new version of the app!"

exit 0
