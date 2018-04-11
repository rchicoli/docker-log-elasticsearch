#!/usr/bin/env bash

#==============================================================================
# Basht: is a bats alternative, testing framework for bash
#------------------------------------------------------------------------------
# Author: Rafael Chicoli
# License: Apache License 2.0, http://www.apache.org/licenses/
#==============================================================================

# vim: set ts=4 sw=4 tw=0 noet :

#-------------------------- TODO ----------------------------------------------

# CREATE A NEW PROJECT IN GITHUB
# CREATE A NEW PROJECT IN GITHUB
# CREATE A NEW PROJECT IN GITHUB
# CREATE A NEW PROJECT IN GITHUB
# CREATE A NEW PROJECT IN GITHUB

#-------------------------- Change Runtime configuration parameters -----------

# Abort on errors
# set -e
# Echo commands
# set -x

#-------------------------- Helper functions --------------------------------

# Console colors
red='\033[0;31m'
green='\033[0;32m'
yellow='\033[1;33m'
blue='\033[0;34m'
magenta='\033[0;35m'
no_color='\033[0m'

red ()    { timestamp="$(date +"[%Y-%m-%d %H:%M:%S]")"; echo -e "${red}$*${no_color}";    }
green ()  { timestamp="$(date +"[%Y-%m-%d %H:%M:%S]")"; echo -e "${green}$*${no_color}";  }
blue ()   { timestamp="$(date +"[%Y-%m-%d %H:%M:%S]")"; echo -e "${blue}$*${no_color}";   }
magenta ()   { timestamp="$(date +"[%Y-%m-%d %H:%M:%S]")"; echo -e "${magenta}$timestamp $*${no_color}";   }
yellow () { timestamp="$(date +"[%Y-%m-%d %H:%M:%S]")"; echo -e "${yellow}$*${no_color}"; }
no_color () { timestamp="$(date +"[%Y-%m-%d %H:%M:%S]")"; echo -e "$timestamp ${no_color}$*${no_color}"; }

#-------------------------- Environment ----------------------------

BASHT_DEBUG=${BASHT_DEBUG:-":"}

export BASHT_TEST_DIRNAME=$(dirname "$0")
export BASHT_TEST_NUMBER=1
BASHT_SUBTEST_NUMBER=1
BASHT_SUBMATCH_NUMBER=

BASHT_EXIT_CODE=0

#-------------------------- Function ----------------------------

function basht_run(){

	output=$("$@" 2>&1); status=$?

  if [[ "$status" -eq 0 ]]; then

    test "$BASHT_DEBUG" = ":" && green "$BASHT_TEST_NUMBER.$BASHT_SUBTEST_NUMBER - Command($status): ${green}Passed"
    $BASHT_DEBUG magenta "$BASHT_TEST_NUMBER.$((BASHT_SUBTEST_NUMBER++)) - Command($status): ${green}Passed"
    $BASHT_DEBUG green "$*"
    # $BASHT_DEBUG yellow "Passed: OK"
    $BASHT_DEBUG yellow "$output$"
    return $status

  else
    magenta "$BASHT_TEST_NUMBER.$((BASHT_SUBTEST_NUMBER++)) - Command($status): ${red}Failed"
    red "$*"
    yellow "$output"
    BASHT_EXIT_CODE=1

    # if fail fast, then exit $status
    # return $status
  fi

}

function basht_assert(){

	# if output=$(echo "$output" | bash -c "$1" 2>&1); status=$?; then
  local output
  local status
  output=$(bash -c "$1"); status="$?"

  if [[ "$2" == "equals" ]] || [[ "$2" == "==" ]]; then
    if [[ "$output" == "$3" ]]; then
      $BASHT_DEBUG magenta "$BASHT_TEST_NUMBER.$BASHT_SUBTEST_NUMBER.$((BASHT_SUBMATCH_NUMBER++)) - Sub-Command($status): ${green}Passed"
      $BASHT_DEBUG blue "${1}"
      return $status
    fi
  elif [[ "$2" == "regexp" ]] || [[ "$2" == "=~" ]]; then
    if [[ "$output" =~ $3 ]]; then
      $BASHT_DEBUG magenta "$BASHT_TEST_NUMBER.$BASHT_SUBTEST_NUMBER.$((BASHT_SUBMATCH_NUMBER++)) - Sub-Command($status): ${green}Passed"
      $BASHT_DEBUG blue "${1}"
      return $status
    fi
  fi

  magenta "$BASHT_TEST_NUMBER.$BASHT_SUBTEST_NUMBER.$((BASHT_SUBMATCH_NUMBER++)) - Sub-Command($status): ${red}Failed"
  red "$*"
  yellow "\"${output}\" does not match \"${3}\""
  BASHT_EXIT_CODE=1
  # if fail fast, then exit $status
  # return $status

}

function check_status(){
  if [[ "$status" -eq "$1" ]]; then
    : green "Exit-Status"
  fi
}

# die exits the script
function die(){
  local _ret=$2
  test -n "$_ret" || _ret=1
  test "$_arg_print_help" = yes && print_help >&2
  echo "$1" >&2
  exit ${_ret}
}

#-------------------------- CLI --------------------------------

# defaults variables declaration

_arg_print_help="no"
# _arg_verbose="no"
_arg_version="0.0.1"
_arg_test_file=""
_arg_test_dir=""
# _arg_logfile="/dev/stdout"

# print_help outputs the help command
function print_help (){
  printf 'Basht: testing framework for bash\n'
  printf 'Usage: %s [arguments] [-v|--version] [--(no-)verbose] [-h|--help]\n\n' "$0"

  printf 'Arguments:\n'
  printf "  -b,--test-file:\t\t directory which contain the tests (default: '%s')\n\n" "${_arg_test_file}"

  # printf "  -l,--log-file:\t\t Log file (default: '%s')\n\n" "${_arg_logfile}"
  printf "  -v,--verbose:\t\t\t Increase the output level\n"
  printf "  -v,--version:\t\t\t Prints version\n"
  printf "  -h,--help:\t\t\t Prints help\n\n"
}

# parsing parameters
while test $# -gt 0
do
  _key="$1"
  case "$_key" in

    # required parameters
    -w|--test-dir)
      test $# -lt 2 && die "Missing value for argument '$_key'." 1
      _arg_test_dir="$2"
      shift
      ;;
    -w|--test-file)
      test $# -lt 2 && die "Missing value for argument '$_key'." 1
      _arg_test_file="$2"
      shift
      ;;

    # common options
    # -l|--log-file)
    #   test $# -lt 2 && die "Missing value for argument '$_key'." 1
    #   _arg_logfile="$2"
    #   shift
    #   ;;
    -v|--version)
      echo "$0" "$_arg_version"
      exit 0
      ;;
    --no-verbose|--verbose)
      BASHT_DEBUG=
      test "${1:0:5}" = "--no-" && BASHT_DEBUG=:
      ;;
    -h|--help)
      print_help
      exit 0
      ;;

    *)
      _positionals+=("$1")
      ;;
  esac
  shift
done

# main code
function main(){

  _positional_names=('_arg_filename' )
  for (( ii = 0; ii < ${#_positionals[@]}; ii++))
  do
    eval "${_positional_names[ii]}=\${_positionals[ii]}" || die "Error during argument parsing" 1
  done

  if [[ "${_arg_test_file}" == "" ]] && [[ "${_arg_test_dir}" == "" ]]; then
     _arg_print_help=yes die "missing --test-file or --test-dir"
  fi
  if [[ "${_arg_test_file}" != "" ]]; then
    _loop_func="$_arg_test_file"
  fi
  if [[ "${_arg_test_dir}" != "" ]]; then
    _loop_func="$_arg_test_dir"
  fi

  # _arg_print_help=yes die "do not know what to do"

  for func_file in $(ls -1 "$_loop_func"); do
    source "./${_arg_test_dir}/$func_file"
    export BASHT_TEST_FILENAME="$func_file"
    for func in $(sed -nr 's/^function (.*)\(\)\{/\1/p' < "./${_arg_test_dir}/$func_file" | grep -Ev "setUp|tearDown"); do

      setUp
      magenta "$func is running..."
      "$func"
      tearDown
      # if fail fast, then exit $BASHT_EXIT_CODE
      ((BASHT_TEST_NUMBER++))
      BASHT_SUBTEST_NUMBER=1

    done
  done

  return $BASHT_EXIT_CODE
}

main
