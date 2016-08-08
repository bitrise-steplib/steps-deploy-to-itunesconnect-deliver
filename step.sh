#!/bin/bash

THIS_SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
# load bash utils
source "${THIS_SCRIPT_DIR}/bash_utils/utils.sh"
source "${THIS_SCRIPT_DIR}/bash_utils/formatted_output.sh"


echo "Config:"
echo " * ipa_path: ${ipa_path}"
echo " * pkg_path: ${pkg_path}"
echo " * itunescon_user: ${itunescon_user}"
echo " * password: ***"
echo " * app_id: ${app_id}"
echo " * submit_for_beta: ${submit_for_beta}"
echo " * skip_metadata: ${skip_metadata}"
echo " * skip_screenshots: ${skip_screenshots}"
echo " * team_id: ${team_id}"
echo " * team_name: ${team_name}"
echo " * update_deliver: ${update_deliver}"

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

  if [[ "${submit_for_beta}" == "yes" ]] ; then
	write_section_to_formatted_output "**Beta deploy note:** you
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

if [ -z "${ipa_path}" ] && [ -z "${pkg_path}" ] ; then
	echo " [!] ipa/pkg path not provided!"
	exit 1
fi

if [ -z "${password}" ] ; then
	echo " [!] \`password\` not provided!"
	exit 1
fi

if [ -z "${itunescon_user}" ] ; then
	echo " [!] \`itunescon_user\` not provided!"
	exit 1
fi

if [ -z "${app_id}" ] ; then
	echo " [!] \`app_id\` not provided!"
	exit 1
fi


CONFIG_package_type=''
CONFIG_package_path=''
if [ -n "${ipa_path}" ] ; then
  CONFIG_package_type='--ipa'
  CONFIG_package_path="${ipa_path}"
elif [ -n "${pkg_path}" ] ; then
  CONFIG_package_type='--pkg'
  CONFIG_package_path="${pkg_path}"
fi

CONFIG_testflight_beta_deploy_type_flag=''
if [[ "${submit_for_beta}" == "yes" ]] ; then
	CONFIG_testflight_beta_deploy_type_flag='--submit_for_review'
fi

CONFIG_skip_metadata_type_flag=''
if [[ "${skip_metadata}" == "yes" ]] ; then
	CONFIG_skip_metadata_type_flag='--skip_metadata'
fi

CONFIG_skip_screenshots_type_flag=''
if [[ "${skip_screenshots}" == "yes" ]] ; then
	CONFIG_skip_screenshots_type_flag='--skip_screenshots'
fi

# ---------------------
# --- Main

write_section_to_formatted_output "# Setup"
bash "${THIS_SCRIPT_DIR}/_setup.sh" ${update_deliver}
fail_if_cmd_error "Failed to setup the required tools!"

write_section_to_formatted_output "# Deploy"

write_section_to_formatted_output "**Note:** if your password
contains special characters
and you experience problems, please
consider changing your password
to something with only
alphanumeric characters."

write_section_to_formatted_output "**Be advised** that this
step uses a well maintained, open source tool which
uses *undocumented and unsupported APIs* (because the current
iTunes Connect platform does not have a documented and supported API)
to perform the deployment.
This means that when the API changes
**this step might fail until the tool is updated**."


export DELIVER_USER="${itunescon_user}"
export DELIVER_PASSWORD="${password}"
export DELIVER_APP_ID="${app_id}"

if [ -n "${team_id}" ] ; then
  deliver --team_id "${team_id}" ${CONFIG_package_type} "${CONFIG_package_path}" ${CONFIG_skip_screenshots_type_flag} ${CONFIG_skip_metadata_type_flag} --force ${CONFIG_testflight_beta_deploy_type_flag}
elif [ -n "${team_name}" ] ; then
  deliver --team_name "${team_name}" ${CONFIG_package_type} "${CONFIG_package_path}" ${CONFIG_skip_screenshots_type_flag} ${CONFIG_skip_metadata_type_flag} --force ${CONFIG_testflight_beta_deploy_type_flag}
else
  deliver ${CONFIG_package_type} "${CONFIG_package_path}" ${CONFIG_skip_screenshots_type_flag} ${CONFIG_skip_metadata_type_flag} --force ${CONFIG_testflight_beta_deploy_type_flag}
fi
fail_if_cmd_error "Deploy failed!"

write_section_to_formatted_output "# Success"
echo_string_to_formatted_output "* The app (.ipa) was successfully uploaded to [iTunes Connect](https://itunesconnect.apple.com), you should see it in the *Prerelease* section on the app's iTunes Connect page!"
if [[ "${submit_for_beta}" != "yes" ]] ; then
	echo_string_to_formatted_output "* **Don't forget to enable** the **TestFlight Beta Testing** switch on iTunes Connect (on the *Prerelease* tab of the app) if this is a new version of the app!"
fi

exit 0
