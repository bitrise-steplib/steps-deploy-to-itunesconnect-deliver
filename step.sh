#!/bin/bash

THIS_SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

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

echo "Start"

bash "${THIS_SCRIPT_DIR}/_setup.sh"

export DELIVER_USER="${STEP_DELIVER_DEPLOY_ITUNESCON_USER}"
export DELIVER_PASSWORD="${STEP_DELIVER_DEPLOY_ITUNESCON_PASSWORD}"

deliver testflight --force --skip-deploy -a "${STEP_DELIVER_DEPLOY_ITUNESCON_APP_ID}" "${STEP_DELIVER_DEPLOY_IPA_PATH}"

exit 0
