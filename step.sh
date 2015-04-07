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



# ---------------------
# --- Main

set -e
set -v

write_section_to_formatted_output "# Setup"
set +e
bash "${THIS_SCRIPT_DIR}/_setup.sh"
fail_if_cmd_error "Failed to setup the required tools!"
set -e

write_section_to_formatted_output "# Deploy"
set +e
export DELIVER_USER="${STEP_DELIVER_DEPLOY_ITUNESCON_USER}"
export DELIVER_PASSWORD="${STEP_DELIVER_DEPLOY_ITUNESCON_PASSWORD}"
deliver testflight --skip-deploy -a "${STEP_DELIVER_DEPLOY_ITUNESCON_APP_ID}" "${STEP_DELIVER_DEPLOY_IPA_PATH}"
fail_if_cmd_error "Deploy failed!"
set -e

write_section_to_formatted_output "# Success"
echo_string_to_formatted_output "* The app (.ipa) was successfully uploaded to [iTunes Connect](https://itunesconnect.apple.com), you should see it in the *Prerelease* section on the app's iTunes Connect page!"
echo_string_to_formatted_output "* Don't forget to enable the **TestFlight Beta Testing** switch on iTunes Connect (on the *Prerelease* tab of the app) if this is a new version of the app!"

exit 0
