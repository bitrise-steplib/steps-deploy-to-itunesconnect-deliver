#!/bin/bash

#
# Formatted output file (Markdown format) helper
#
# You can find more bash utility / helper scripts at [https://github.com/bitrise-io/steps-utils-bash-toolkit](https://github.com/bitrise-io/steps-utils-bash-toolkit)
#

formatted_output_file_path="$BITRISE_STEP_FORMATTED_OUTPUT_FILE_PATH"

#
# Writes a single line into the Formatted Output
#  param 1: the text to write
#  param 2: if not zero then the given text will be printed to stdout too
function echo_string_to_formatted_output {
	print_msg=$1
	is_dont_print_to_std_out=$2

	echo "${print_msg}" >> ${formatted_output_file_path}
	if [ -z ${is_dont_print_to_std_out} ]; then
		echo "${print_msg}"
	fi
}

#
# Writes a markdown section (empty-line, text, empty-line) Formatted Output
#  param 1: the text to write
#  param 2: if not zero then the given text will be printed to stdout too
function write_section_to_formatted_output {
	print_msg=$1
	is_dont_print_to_std_out=$2

	echo '' >> ${formatted_output_file_path}
	echo "${print_msg}" >> ${formatted_output_file_path}
	echo '' >> ${formatted_output_file_path}

	if [ -z ${is_dont_print_to_std_out} ]; then
		echo ''
		echo "${print_msg}"
		echo ''
	fi
}

#
# Writes a markdown section start (text, empty-line) Formatted Output
#  param 1: the text to write
#  param 2: if not zero then the given text will be printed to stdout too
function write_section_start_to_formatted_output {
	print_msg=$1
	is_dont_print_to_std_out=$2

	echo "${print_msg}" >> ${formatted_output_file_path}
	echo '' >> ${formatted_output_file_path}

	if [ -z ${is_dont_print_to_std_out} ]; then
		echo "${print_msg}"
		echo ''
	fi
}
