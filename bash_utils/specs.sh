#!/bin/bash

function print_and_do_command {
  echo "$ $@"
  $@
}

#
# This one expects a string as it's input, and will eval it
# 
# Useful for piped commands like this: print_and_do_command_string "printf '%s' \"$filecont\" > \"$testfile_path\""
#  where calling print_and_do_command function would write the command itself into the file as well because
#  of the precedence order of the '>' operator
#
function print_and_do_command_string {
  echo "$ $1"
  eval "$1"
}

#
# Inspects the test result (it's first parameter) and increments the success or the error counter
#  ! Requires the test_results_success_count and test_results_error_count variables to be defined
#
# Example: 
#  (your test commands)
#  test_result=$?
#  inspect_test_result $test_result
#
function inspect_test_result {
  if [ $1 -eq 0 ]; then
    test_results_success_count=$[test_results_success_count + 1]
  else
    test_results_error_count=$[test_results_error_count + 1]
  fi
}

#
# First param is the expect message, other are the command which will be executed.
#
# Example:
#  expect_success "Folder $testfold_path should exist" \
#    is_dir_exist "$testfold_path"
#
function expect_success {
  expect_msg="$1"

  echo " -> $expect_msg"
  "${@:2}"
  cmd_res=$?

  if [ $cmd_res -eq 0 ]; then
    echo " [OK] Expected zero return code, got: 0"
  else
    echo " [ERROR] Expected zero return code, got: $cmd_res"
    exit 1
  fi
}

#
# First param is the expect message, other are the command which will be executed.
#
# Example:
#  expect_error "Folder $testfold_path should NOT exist" \
#    is_dir_exist "$testfold_path"
#
function expect_error {
  expect_msg="$1"

  echo " -> $expect_msg"
  "${@:2}"
  cmd_res=$?

  if [ ! $cmd_res -eq 0 ]; then
    echo " [OK] Expected non-zero return code, got: $cmd_res"
  else
    echo " [ERROR] Expected non-zero return code, got: 0"
    exit 1
  fi
}

function is_dir_exist {
  if [ -d "$1" ]; then
    return 0
  else
    return 1
  fi
}

function is_file_exist {
  if [ -f "$1" ]; then
    return 0
  else
    return 1
  fi
}

function compare_file_content {
  filepth="$1"
  expected_filecontent="$2"

  actual_file_content=$(cat "$filepth")
  if [ "$actual_file_content" == "$expected_filecontent" ]; then
    return 0
  else
    return 1
  fi
}


# ---------------------
# --- Example tests ---

# [DESCRIBE] Create test file then remove it
#  [IT SHOULD] create the test file, then it should remove it
(
  testfile_path='test file.txt'
  filecont='test file content'
  print_and_do_command_string "printf '%s' \"$filecont\" > \"$testfile_path\""

  # File should exist, and content should be the same as we provided
  expect_success "File $testfile_path should exist" \
    is_file_exist "$testfile_path"

  expect_success "File content should be the one we provided" \
    compare_file_content "$testfile_path" "$filecont"

  # Remove the file
  expect_success "We should be able to remove the file" \
    rm "$testfile_path"

  # Expect the file to not to exist anymore
  expect_error "File $testfile_path should NOT exist" \
    is_file_exist "$testfile_path"
)
test_result=$?
inspect_test_result $test_result


# [DESCRIBE] A failed test example
#  [IT SHOULD] fail
(
  this_file_should_not_exist='some-file-path-here.ext'
  # This should fail - expect the file to exist (but it should not)
  expect_success "File $this_file_should_not_exist should exist" \
    is_file_exist "$this_file_should_not_exist"
)
test_result=$?
inspect_test_result $test_result


# --------------------
# --- Test Results ---

echo
echo "--- Results ---"
echo " * Errors: $test_results_error_count"
echo " * Success: $test_results_success_count"
echo "---------------"
