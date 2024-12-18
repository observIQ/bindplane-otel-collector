#!/usr/bin/env bash
# Copyright  observIQ, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

PREREQS="printf sed uname sudo tar gzip"
INDENT_WIDTH='  '
indent=""

collector_dir=/opt/bindplane-otel-collector

# Colors
num_colors=$(tput colors 2>/dev/null)
if test -n "$num_colors" && test "$num_colors" -ge 8; then
  reset="$(tput sgr0)"
  fg_cyan="$(tput setaf 6)"
  fg_green="$(tput setaf 2)"
  fg_red="$(tput setaf 1)"
  fg_yellow="$(tput setaf 3)"
fi

if [ -z "$reset" ]; then
  sed_ignore=''
else
  sed_ignore="/^[$reset]+$/!"
fi

printf() {
  if command -v sed >/dev/null; then
    command printf -- "$@" | sed -E "$sed_ignore s/^/$indent/g" # Ignore sole reset characters if defined
  else
    # Ignore $* suggestion as this breaks the output
    # shellcheck disable=SC2145
    command printf -- "$indent$@"
  fi
}

increase_indent() { indent="$INDENT_WIDTH$indent"; }
decrease_indent() { indent="${indent#*"$INDENT_WIDTH"}"; }

# Color functions reset only when given an argument
# Ignore "parameters are never passed"
# shellcheck disable=SC2120
reset() { command printf "$reset$*$(if [ -n "$1" ]; then command printf "$reset"; fi)"; }
fg_cyan() { command printf "$fg_cyan$*$(if [ -n "$1" ]; then command printf "$reset"; fi)"; }
fg_green() { command printf "$fg_green$*$(if [ -n "$1" ]; then command printf "$reset"; fi)"; }
fg_red() { command printf "$fg_red$*$(if [ -n "$1" ]; then command printf "$reset"; fi)"; }
fg_yellow() { command printf "$fg_yellow$*$(if [ -n "$1" ]; then command printf "$reset"; fi)"; }

# Intentionally using variables in format string
# shellcheck disable=SC2059
info() { printf "$*\\n"; }

# Intentionally using variables in format string
# shellcheck disable=SC2059
error() {
  increase_indent
  printf "$fg_red$*$reset\\n"
  decrease_indent
}

# Intentionally using variables in format string
# shellcheck disable=SC2059
success() { printf "$fg_green$*$reset\\n"; }

observiq_banner() {
  fg_cyan "           888                                        8888888 .d88888b.\\n"
  fg_cyan "           888                                          888  d88P\" \"Y88b\\n"
  fg_cyan "           888                                          888  888     888\\n"
  fg_cyan "   .d88b.  88888b.  .d8888b   .d88b.  888d888 888  888  888  888     888\\n"
  fg_cyan "  d88\"\"88b 888 \"88b 88K      d8P  Y8b 888P\"   888  888  888  888     888\\n"
  fg_cyan "  888  888 888  888 \"Y8888b. 88888888 888     Y88  88P  888  888 Y8b 888\\n"
  fg_cyan "  Y88..88P 888 d88P      X88 Y8b.     888      Y8bd8P   888  Y88b.Y8b88P\\n"
  fg_cyan "   \"Y88P\"  88888P\"   88888P'  \"Y8888  888       Y88P  8888888 \"Y888888\"\\n"
  fg_cyan "                                                                   Y8b  \\n"

  reset
}

separator() { printf "===================================================\\n"; }

banner() {
  printf "\\n"
  separator
  printf "| %s\\n" "$*"
  separator
}

usage() {
  increase_indent
  USAGE=$(
    cat <<EOF
Usage:
  Collects support bundle for BDOT
EOF
  )
  info "$USAGE"
  decrease_indent
  return 0
}

force_exit() {
  # Exit regardless of subshell level with no "Terminated" message
  kill -PIPE $$
  # Call exit to handle special circumstances (like running script during docker container build)
  exit 1
}

error_exit() {
  line_num=$(if [ -n "$1" ]; then command printf ":$1"; fi)
  error "ERROR ($SCRIPT_NAME$line_num): ${2:-Unknown Error}" >&2
  if [ -n "$0" ]; then
    increase_indent
    error "$*"
    decrease_indent
  fi
  force_exit
}

succeeded() {
  increase_indent
  success "Succeeded!"
  decrease_indent
}

failed() {
  error "Failed!"
}

root_check() {
  system_user_name=$(id -un)
  if [[ "${system_user_name}" != 'root' || $EUID -ne 0 ]]; then
    failed
    error_exit "$LINENO" "Script needs to be run as root or with sudo"
  fi
}

os_check() {
  info "Checking that the operating system is supported..."
  os_type=$(uname -s)
  case "$os_type" in
  Linux)
    succeeded
    ;;
  *)
    failed
    error_exit "$LINENO" "The operating system $(fg_yellow "$os_type") is not supported by this script."
    ;;
  esac
}

os_arch_check() {
  info "Checking for valid operating system architecture..."
  arch=$(uname -m)
  case "$arch" in
  x86_64 | amd64)
    arch=amd64
    ;;
  aarch64 | arm64 | aarch64_be | armv8b | armv8l)
    arch=arm64
    succeeded
    ;;
  *)
    failed
    error_exit "$LINENO" "The operating system architecture $(fg_yellow "$arch") is not supported by this script."
    ;;
  esac
}

# This will check if the current environment has
# all required shell dependencies to run the installation.
dependencies_check() {
  info "Checking for script dependencies..."
  FAILED_PREREQS=''
  for prerequisite in $PREREQS; do
    if command -v "$prerequisite" >/dev/null; then
      continue
    else
      if [ -z "$FAILED_PREREQS" ]; then
        FAILED_PREREQS="${fg_red}$prerequisite${reset}"
      else
        FAILED_PREREQS="$FAILED_PREREQS, ${fg_red}$prerequisite${reset}"
      fi
    fi
  done

  if [ -n "$FAILED_PREREQS" ]; then
    failed
    error_exit "$LINENO" "The following dependencies are required by this script: [$FAILED_PREREQS]"
  fi
  succeeded
}

check_prereqs() {
  banner "Checking Prerequisites"
  increase_indent
  root_check
  os_check
  os_arch_check
  dependencies_check
  success "Prerequisite check complete!"
  decrease_indent
}

function bundle_files() {
  banner "Collecting files for support bundle"
  increase_indent
  # Directory for logs
  log_dir="$collector_dir/log"

  # Check if directory exists
  if [ ! -d "$log_dir" ]; then
    info "Directory ($fg_red "$log_dir")$(reset) does not exist."
    read -p "Please enter an existing directory for logs: " log_dir
    if [ ! -d "$log_dir" ]; then
      echo "Directory $log_dir does not exist."
      return 1
    fi
  fi

  read -p "Do you want to include only the most recent logs (y or n)? " response
  increase_indent
  tar_filename="support_bundle_$(date +%Y%m%d_%H%M%S).tar"
  if [ "$response" = "n" ]; then
    # Get all the log files
    info "Collecting all log files in $(fg_cyan "$log_dir")$(reset)"
    tar -cf $tar_filename -C $log_dir $(ls -Art $log_dir)
  else
    # Get the most recent log file
    recent_log=$(ls -Art $log_dir | tail -n 1)
    if [ -n "$recent_log" ]; then
      tar -cf $tar_filename -C $log_dir $recent_log
      info "Added file $(fg_cyan "$recent_log")$(reset) to the tar file."

    else
      info "No logs found in $(fg_red $log_dir)"
      return 1
    fi
    # Get the /log/observiq_collector.err file
    err_file="$log_dir/observiq_collector.err"
    if [ -f "$err_file" ]; then
      tar --append --file=$tar_filename -C $log_dir observiq_collector.err
      info "Added file $(fg_cyan "$err_file")$(reset) to the tar file."
    fi
  fi

  # Check if the files exist, if yes append them to the tar file
  for file in issue os-release; do
    if [ -f "/etc/$file" ]; then
      # These might be symlinks, so cat them to real files
      cat "/etc/$file" >"$file"
      tar --append --file=$tar_filename $file
      rm $file
      info "Added file $(fg_cyan "/etc/$file")$(reset) to the tar file."
    else
      info "File $(fg_red "/etc/$file")$(reset) does not exist."
    fi
  done

  collector_config="$collector_dir/config.yaml"
  if [ -f "$collector_config" ]; then
    read -p "Do you want to include the collector config (y or n)? " response
    if [ "$response" != "n" ]; then
      info "Adding collector config $(fg_cyan "$collector_config")$(reset)"
      tar --append --file=$tar_filename -C $collector_dir config.yaml
    fi
  fi

  # Grab the logs from journalctl -- in some cases, the collector.log file
  # may be empty, but there may be logs in journalctl
  info "Collecting logs from journalctl..."
  journalctl -u bindplane-otel-collector.service -n 50 >journalctl.log
  tar --append --file=$tar_filename journalctl.log
  rm journalctl.log

  # Compress the tar file
  info "Compressing the tar file..."
  gzip $tar_filename

  info "Files have been added to the file $(realpath $tar_filename.gz) successfully."
}

main() {
  if [ $# -ge 1 ]; then
    while [ -n "$1" ]; do
      case "$1" in
      -h | --help)
        usage
        force_exit
        ;;
      --)
        shift
        break
        ;;
      *)
        error "Invalid argument: $1"
        usage
        force_exit
        ;;
      esac
    done
  fi

  observiq_banner
  check_prereqs
  bundle_files
}

main "$@"
