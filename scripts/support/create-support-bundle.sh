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

PREREQS="printf sed awk uname sudo tar gzip curl"
INDENT_WIDTH='  '
indent=""

collector_dir=/opt/observiq-otel-collector

# Colors
num_colors=$(tput colors 2>/dev/null || true)
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
    command printf -- "$@" | sed -E "$sed_ignore s/^/$indent/g"  # Ignore sole reset characters if defined
  else
    # Ignore $* suggestion as this breaks the output
    # shellcheck disable=SC2145
    command printf -- "$indent$@"
  fi
}

increase_indent() { indent="$INDENT_WIDTH$indent" ; }
decrease_indent() { indent="${indent#*"$INDENT_WIDTH"}" ; }

# Color functions reset only when given an argument
# Ignore "parameters are never passed"
# shellcheck disable=SC2120
reset() { command printf "$reset$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }
fg_cyan() { command printf "$fg_cyan$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }
fg_green() { command printf "$fg_green$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }
fg_red() { command printf "$fg_red$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }
fg_yellow() { command printf "$fg_yellow$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }

# Intentionally using variables in format string
# shellcheck disable=SC2059
info() { printf "$*\\n" ; }

# Intentionally using variables in format string
# shellcheck disable=SC2059
error() {
  increase_indent
  printf "$fg_red$*$reset\\n"
  decrease_indent
}

# Intentionally using variables in format string
# shellcheck disable=SC2059
success() { printf "$fg_green$*$reset\\n" ; }

bindplane_banner()
{
  fg_cyan " oooooooooo.   o8o                    .o8              oooo\\n"
  fg_cyan " '888'   '88b  '\"'                   \"888              '888\\n"
  fg_cyan "  888     888 oooo  ooo. .oo.    .oooo888  oooooooo.    888   .oooo.   ooo. .oo.    .ooooo.\\n"
  fg_cyan "  888oooo888' '888  '888P\"Y88b  d88' '888  '888' 'Y88.  888  'P  )88b  '888P\"Y88b  d88' '88b\\n"
  fg_cyan "  888    '88b  888   888   888  888   888   888    888  888   .oP\"888   888   888  888ooo888\\n"
  fg_cyan "  888    .88P  888   888   888  888   888   888   .88'  888  d8(  888   888   888  888    .o\\n"
  fg_cyan " o888bood8P'  o888o o888o o888o 'Y8bod88P\"  888bod8P'  o888o 'Y888\"\"8o o888o o888o '88bod8P'\\n"
  fg_cyan "                                            888\\n"
  fg_cyan "                                           o888o\\n"

  reset
}

separator() { printf "===================================================\\n" ; }

banner() {
  printf "\\n"
  separator
  printf "| %s\\n" "$*" ;
  separator
}

usage() {
  increase_indent
  USAGE=$(cat <<EOF
Usage:
  Collects support bundle for Bindplane Agent
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
    x86_64|amd64)
      arch=amd64
      ;;
    aarch64|arm64|aarch64_be|armv8b|armv8l)
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

# Redact secrets from a staged bundle file in place (originals untouched).
# awk pass runs first (needs the raw block indicator): (1) redact a multi-line
# YAML block scalar under a sensitive key by dropping its more-indented body;
# (2) collapse PEM private-key blocks (same-line or multi-line) to one marker,
# keeping text before BEGIN and after END so the file is not truncated to EOF.
# sed passes then handle single-line keys, secret-shaped values anywhere (URL
# creds, Bearer, AWS key, JWT), and a mid-line "key: value" in log text.
redact_in_place() {
  f="$1"
  [ -f "$f" ] || return 0
  awk '
    {
      line = $0
      while (1) {
        if (match(tolower(line), /^[ \t]*[a-z0-9_.-]*(password|passwd|secret|token|key|creds|honeycomb|api[_-]?key|access[_-]?key|private[_-]?key|encryption[_-]?key|credential|passphrase|authorization|bearer|connection[_-]?string|account[_-]?key)[a-z0-9_.-]*[ \t]*:[ \t]*[|>][-+]?[0-9]?[ \t]*$/)) {
          match(line, /^[ \t]*/); ind = RLENGTH
          k = line; sub(/:[ \t]*[|>][-+]?[0-9]?[ \t]*$/, ": \"[REDACTED]\"", k); print k
          got = 0
          while ((getline b) > 0) {
            if (b ~ /^[ \t]*$/) continue
            match(b, /^[ \t]*/); if (RLENGTH > ind) continue
            got = 1; break
          }
          if (!got) next
          line = b; continue
        }
        break
      }
      while (match(line, /-----BEGIN[A-Z ]*PRIVATE KEY-----/)) {
        b = RSTART
        rest = substr(line, b)
        if (match(rest, /-----END[A-Z ]*PRIVATE KEY-----/)) {
          e = b + RSTART + RLENGTH - 1
          line = substr(line, 1, b - 1) "[REDACTED-PRIVATE-KEY]" substr(line, e + 1)
        } else {
          prefix = substr(line, 1, b - 1)
          done = 0
          while ((getline nl) > 0) {
            if (nl ~ /-----END[A-Z ]*PRIVATE KEY-----/) {
              sub(/^.*-----END[A-Z ]*PRIVATE KEY-----/, "", nl)
              line = prefix "[REDACTED-PRIVATE-KEY]" nl; done = 1; break
            }
          }
          if (!done) { line = prefix "[REDACTED-PRIVATE-KEY]"; break }
        }
      }
      print line
    }
  ' "$f" > "$f.redact.$$" && mv "$f.redact.$$" "$f"
  sed -E -i \
    -e 's/^([[:space:]]*[-]?[[:space:]]*[A-Za-z0-9_.-]*(password|passwd|secret|token|key|creds|honeycomb|api[_-]?key|access[_-]?key|private[_-]?key|encryption[_-]?key|credential|passphrase|authorization|bearer|connection[_-]?string|account[_-]?key)[A-Za-z0-9_.-]*[[:space:]]*:[[:space:]]*).*$/\1"[REDACTED]"/I' \
    -e 's#([A-Za-z][A-Za-z0-9+.-]*://[^:/@[:space:]]+):[^@/[:space:]]+@#\1:[REDACTED]@#g' \
    -e 's/([Bb]earer[[:space:]]+)[A-Za-z0-9._~+/=-]+/\1[REDACTED]/g' \
    -e 's/AKIA[0-9A-Z]{16}/[REDACTED-AWS-ACCESS-KEY]/g' \
    -e 's/eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+/[REDACTED-JWT]/g' \
    -e 's/([A-Za-z0-9_.-]*(password|passwd|secret|token|api[_-]?key|access[_-]?key|private[_-]?key|passphrase|authorization|bearer)[A-Za-z0-9_.-]*[[:space:]]*[:=][[:space:]]*).*$/\1"[REDACTED]"/I' \
    "$f"
}

function bundle_files() {
    banner "Collecting files for support bundle"
    increase_indent
    # Directory for logs
    log_dir="$collector_dir/log"
    
    # Check if directory exists
    if [ ! -d "$log_dir" ]; then
        info "Directory ($fg_red $log_dir)$(reset) does not exist."
        # shellcheck disable=SC2162
        read -p "Please enter an existing directory for logs: " log_dir
        if [ ! -d "$log_dir" ]; then
            echo "Directory $log_dir does not exist."
            return 1
        fi
    fi

    # shellcheck disable=SC2162
    read -p "Do you want to include only the most recent logs (y or n)? " response
    increase_indent
    tar_filename="support_bundle_$(date +%Y%m%d_%H%M%S).tar"
    # Stage selected logs, redact them, then build the tarball from the
    # redacted copies. The originals under $log_dir are never modified.
    log_stage="sb_logs_$$"
    mkdir -p "$log_stage"
    if [ "$response" = "n" ]; then
        # Get all the log files
        info "Collecting all log files in $(fg_cyan "$log_dir")$(reset)"
        for log_file in "$log_dir"/*; do
            [ -f "$log_file" ] && cp "$log_file" "$log_stage/"
        done
    else
        # Get the most recent log file
        # shellcheck disable=SC2012
        recent_log=$(ls -Art "$log_dir" | tail -n 1)
        if [ -n "$recent_log" ]; then
            cp "$log_dir/$recent_log" "$log_stage/"
            info "Added file $(fg_cyan "$recent_log")$(reset) to the tar file."

        else
            # shellcheck disable=SC2086
            info "No logs found in $(fg_red $log_dir)"
            rm -rf "$log_stage"
            return 1
        fi
        # Get the /log/observiq_collector.err file
        err_file="$log_dir/observiq_collector.err"
        if [ -f "$err_file" ]; then
            cp "$err_file" "$log_stage/"
            info "Added file $(fg_cyan "$err_file")$(reset) to the tar file."
        fi
        err_backup_file="$log_dir/observiq_collector.err.1"
        if [ -f "$err_backup_file" ]; then
            cp "$err_backup_file" "$log_stage/"
            info "Added file $(fg_cyan "$err_backup_file")$(reset) to the tar file."
        fi
    fi
    # Redact every staged log before it enters the bundle.
    for log_file in "$log_stage"/*; do
        [ -f "$log_file" ] && redact_in_place "$log_file"
    done
    tar -cf "$tar_filename" -C "$log_stage" .
    rm -rf "$log_stage"

    # Check if the files exist, if yes append them to the tar file
    for file in issue os-release redhat-release debian_version
    do
        if [ -f "/etc/$file" ]; then
            # These might be symlinks, so cat them to real files
            cat "/etc/$file" > "$file"
            tar --append --file="$tar_filename" $file
            rm $file
            info "Added file $(fg_cyan "/etc/$file")$(reset) to the tar file."
        else
            info "File $(fg_red "/etc/$file")$(reset) does not exist."
        fi
    done

    # Stage config, manager, and journalctl output into a private temp dir so
    # redaction and cleanup never touch same-named files in the working dir.
    file_stage="sb_files_$$"
    mkdir -p "$file_stage"

    collector_config="$collector_dir/config.yaml"
    collector_manager="$collector_dir/manager.yaml"
    if [ -f "$collector_config" ] || [ -f "$collector_manager" ]; then
        # shellcheck disable=SC2162
        read -p "Do you want to include the collector config (y or n)? " response
        if [ "$response" != "n" ]; then
            # Stage a copy, redact it, then bundle the copy so the on-disk
            # originals are never modified.
            if [ -f "$collector_config" ]; then
                info "Adding collector config (redacted) $(fg_cyan "$collector_config")$(reset)"
                cp "$collector_config" "$file_stage/config.yaml"
                redact_in_place "$file_stage/config.yaml"
                tar --append --file="$tar_filename" -C "$file_stage" config.yaml
            fi
            if [ -f "$collector_manager" ]; then
                info "Adding manager config (redacted) $(fg_cyan "$collector_manager")$(reset)"
                cp "$collector_manager" "$file_stage/manager.yaml"
                redact_in_place "$file_stage/manager.yaml"
                tar --append --file="$tar_filename" -C "$file_stage" manager.yaml
            fi
        fi
    fi

    # Grab the logs from journalctl -- in some cases, the collector.log file
    # may be empty, but there may be logs in journalctl
    info "Collecting logs from journalctl..."
    journalctl -u observiq-otel-collector.service -n 50 > "$file_stage/journalctl.log"
    redact_in_place "$file_stage/journalctl.log"
    tar --append --file="$tar_filename" -C "$file_stage" journalctl.log
    rm -rf "$file_stage"

    collect_profiles "$tar_filename"

    # Compress the tar file
    info "Compressing the tar file..."
    gzip "$tar_filename"

    info "Files have been added to the file $(realpath "$tar_filename.gz") successfully."
    decrease_indent
}

collect_profiles() {
  # shellcheck disable=SC2162
  read -p "Collect go pprof profiles? (y/n) " PPROF
  if [[ "$PPROF" == y*  ]]; then
    tar_filename="$1"
    increase_indent
    # POSIX prompt
    info "For endpoint, please use the format http://localhost:1777 or https://localhost:1777\n"
    info "where 1777 is the port you configured in your profile extension (1777 is the default)"
    # shellcheck disable=SC2162
    read -p "Endpoint: " ENDPOINT
    printf "\n"
    info "Collecting golang pprof profiles in parallel..."
    # Fire every profile request concurrently. profile (CPU) and trace use the
    # same 30s window so go tool trace can attribute CPU stacks to trace spans.
    # -f makes curl exit non-zero on an HTTP error so the per-PID status below
    # reflects a real failure instead of a saved error page.
    curl -fksS "$ENDPOINT/debug/pprof/goroutine" --output goroutines.pprof & pid_goroutine=$!
    curl -fksS "$ENDPOINT/debug/pprof/heap" --output heap.pprof & pid_heap=$!
    curl -fksS "$ENDPOINT/debug/pprof/threadcreate" --output threadcreate.pprof & pid_threadcreate=$!
    curl -fksS "$ENDPOINT/debug/pprof/block" --output block.pprof & pid_block=$!
    curl -fksS "$ENDPOINT/debug/pprof/mutex" --output mutex.pprof & pid_mutex=$!
    curl -fksS "$ENDPOINT/debug/pprof/profile?seconds=30" --output cpu.pprof & pid_cpu=$!
    curl -fksS "$ENDPOINT/debug/pprof/trace?seconds=30" --output trace.out & pid_trace=$!

    # Wait on each PID individually so a single failure is attributed by name.
    failed=""
    wait $pid_goroutine || failed="$failed goroutine"
    wait $pid_heap || failed="$failed heap"
    wait $pid_threadcreate || failed="$failed threadcreate"
    wait $pid_block || failed="$failed block"
    wait $pid_mutex || failed="$failed mutex"
    wait $pid_cpu || failed="$failed profile"
    wait $pid_trace || failed="$failed trace"
    if [ -n "$failed" ]; then
      info "Warning: failed to collect the following profiles:$failed"
    fi

    # Bundle only the profiles that were actually written.
    collected=""
    for f in goroutines.pprof heap.pprof threadcreate.pprof block.pprof mutex.pprof cpu.pprof trace.out; do
      [ -f "$f" ] && collected="$collected $f"
    done
    if [ -n "$collected" ]; then
      # shellcheck disable=SC2086
      tar -rf "$tar_filename" $collected
      # shellcheck disable=SC2086
      rm -f $collected
    fi

    info "Profile files have been added to the file $(realpath "$tar_filename") successfully."
    decrease_indent
  fi
}

main() {
  if [ $# -ge 1 ]; then
    while [ -n "$1" ]; do
      case "$1" in                
        -h|--help)
          usage
          force_exit
          ;;
      --)
        shift; break ;;
      *)
        error "Invalid argument: $1"
        usage
        force_exit
        ;;
      esac
    done
  fi

  bindplane_banner
  check_prereqs
  bundle_files
}

# Only run main when executed directly, so tests can source the redaction
# helpers without triggering collection.
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
  main "$@"
fi
