#!/bin/sh
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

# POSIX sh only: AIX /bin/sh is ksh88 and Debian's is dash, so no bash features.
# AIX sed has no -E, -i or I flag, so anything beyond basic sed is done in awk.
OS_TYPE=$(uname -s)
PREREQS="printf sed awk uname tar gzip"
INDENT_WIDTH='  '
indent=""

V1_SERVICE=observiq-otel-collector
V2_SERVICE=bindplane-otel-collector
V1_DEFAULT_DIR=/opt/observiq-otel-collector
V2_DEFAULT_DIR=/opt/bindplane-otel-collector

# Resolved by detect_install.
collector_dir=""
collector_service=""
collector_version=""

# True when a service for the given collector name is installed: a systemd unit,
# or an SRC subsystem on AIX. Dir-independent, so a non-standard install dir
# does not defeat detection.
service_installed() {
  if [ "$OS_TYPE" = AIX ]; then
    lssrc -s "$1" >/dev/null 2>&1
  else
    systemctl list-unit-files --no-legend "$1.service" 2>/dev/null | grep -q "^$1.service"
  fi
}

# Service name for a collector name: the systemd unit, or the SRC subsystem on AIX.
service_name() {
  if [ "$OS_TYPE" = AIX ]; then echo "$1"; else echo "$1.service"; fi
}

# PID of the running service, or empty when it is stopped.
service_pid() {
  if [ "$OS_TYPE" = AIX ]; then
    # A stopped subsystem has no PID column, so only read it when active.
    lssrc -s "$1" 2>/dev/null | awk 'NR == 2 && $NF == "active" { print $(NF - 1) }'
  else
    systemctl show "$1" -p MainPID --value 2>/dev/null
  fi
}

# Path of the service's executable, or empty when it is unknown.
service_exec_path() {
  if [ "$OS_TYPE" = AIX ]; then
    # Field 4 of the subsystem definition is the program path.
    lssrc -S -s "$1" 2>/dev/null | awk -F: 'NR == 2 { print $4 }'
  else
    systemctl show "$1" -p ExecStart 2>/dev/null | sed -n 's/.*path=\([^ ;]*\).*/\1/p' | head -n 1
  fi
}

# Show a prompt on stderr, like bash read -p. POSIX read has no -p.
prompt() { command printf '%s' "$1" >&2 ; }

# Choose the version to collect from the installed flags ($1,$2 = v1,v2 booleans).
# Prompts when both or neither are installed, since the stopped one may be the
# one under investigation. Echoes "v1" or "v2".
select_version() {
  if { [ "$1" = true ] && [ "$2" = true ]; } || { [ "$1" != true ] && [ "$2" != true ]; }; then
    prompt "Which collector version to collect? (1 = v1, 2 = v2) "; read -r choice
    if [ "$choice" = 2 ]; then echo v2; else echo v1; fi
  elif [ "$2" = true ]; then
    echo v2
  else
    echo v1
  fi
}

# Install dir for a service (precedence): COLLECTOR_DIR env, else the dir of the
# service binary (works stopped or running, any location), else the
# standard default, else prompt. Echoes the dir.
resolve_dir() {
  svc="$1"; default="$2"
  if [ -n "${COLLECTOR_DIR:-}" ]; then echo "$COLLECTOR_DIR"; return 0; fi
  execpath=$(service_exec_path "$svc")
  if [ -n "$execpath" ]; then dirname "$execpath"; return 0; fi
  if [ -d "$default" ]; then echo "$default"; return 0; fi
  prompt "Collector directory not found. Enter the collector install directory: "; read -r entered
  echo "$entered"
}

# Detect the installed collector version, then resolve its service and install dir.
detect_install() {
  v1i=false; v2i=false
  if service_installed "$V1_SERVICE"; then v1i=true; fi
  if service_installed "$V2_SERVICE"; then v2i=true; fi
  collector_version=$(select_version "$v1i" "$v2i")
  if [ "$collector_version" = v2 ]; then
    collector_service=$(service_name "$V2_SERVICE")
    collector_dir=$(resolve_dir "$collector_service" "$V2_DEFAULT_DIR")
  else
    collector_service=$(service_name "$V1_SERVICE")
    collector_dir=$(resolve_dir "$collector_service" "$V1_DEFAULT_DIR")
  fi
}

# Colors. None on AIX, like the install script.
reset=""
if [ "$OS_TYPE" != AIX ]; then
  num_colors=$(tput colors 2>/dev/null || true)
  if test -n "$num_colors" && test "$num_colors" -ge 8; then
    reset="$(tput sgr0)"
    fg_cyan="$(tput setaf 6)"
    fg_green="$(tput setaf 2)"
    fg_red="$(tput setaf 1)"
    fg_yellow="$(tput setaf 3)"
  fi
fi

if [ -z "$reset" ]; then
  sed_ignore=''
else
  sed_ignore="/^[$reset]+$/!"
fi

printf() {
  if [ "$OS_TYPE" != AIX ] && command -v sed >/dev/null; then
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
  if [ "${system_user_name}" != 'root' ] || [ "$(id -u)" -ne 0 ]; then
    failed
    error_exit "$LINENO" "Script needs to be run as root or with sudo"
  fi
}

os_check() {
  info "Checking that the operating system is supported..."
  os_type=$(uname -s)
  case "$os_type" in
    Linux|AIX)
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
  if [ "$OS_TYPE" = AIX ]; then
    # Only 64-bit AIX is supported, and its uname -m is a machine ID
    if [ "$(uname -p)" = "powerpc" ] && [ "$(bootinfo -y)" = "64" ]; then
      succeeded
    else
      failed
      error_exit "$LINENO" "Only 64-bit AIX on powerpc is supported. 'uname -p': $(uname -p), 'bootinfo -y': $(bootinfo -y)"
    fi
    return
  fi
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

# Redact secrets from a staged copy in place; originals untouched. Best effort.
# Key list from resource params marked sensitive:true (value/endpoint/DSN
# excluded from whole-value redaction). awk runs first (needs raw block char).
redact_in_place() {
  f="$1"
  [ -f "$f" ] || return 0
  awk '
    {
      line = $0
      while (1) {
        if (match(tolower(line), /^[ \t]*[a-z0-9_.-]*(password|passwd|secret|token|key|creds|credential|honeycomb|authorization|bearer|passphrase|client_id)[a-z0-9_.-]*[ \t]*:[ \t]*[|>][-+]?[0-9]?[ \t]*$/)) {
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
  ' "$f" > "$f.redact.$$" || { rm -f "$f.redact.$$"; error_exit "$LINENO" "redaction failed for $f"; }
  mv "$f.redact.$$" "$f"
  # Require non-empty value so a bare "key:" opener stays valid YAML.
  # awk, not sed -E -i with the I flag, which AIX sed lacks. Case-insensitive
  # rules match on a lowercased copy, then cut the original at the same offsets.
  awk '
    function inline_keys(s,   out, low, m, n) {
      out = ""
      low = tolower(s)
      while (match(low, /[a-z0-9_.-]*(password|passwd|pwd|secret|token|api[_-]?key|access[_-]?key|private[_-]?key|passphrase|authorization|bearer)[a-z0-9_.-]*[[:space:]]*[:=][[:space:]]*[^[:space:];,"]+/)) {
        m = RSTART; n = RLENGTH
        match(substr(low, m, n), /[:=][[:space:]]*/)
        out = out substr(s, 1, m + RSTART + RLENGTH - 2) "[REDACTED]"
        s = substr(s, m + n); low = substr(low, m + n)
      }
      return out s
    }
    function url_creds(s,   out, m, n, p) {
      out = ""
      while (match(s, /[A-Za-z][A-Za-z0-9+.-]*:\/\/[^:\/@[:space:]]+:[^@\/[:space:]]+@/)) {
        m = RSTART; n = RLENGTH
        p = index(substr(s, m, n), "://") + 2
        p += index(substr(s, m + p, n - p), ":")
        out = out substr(s, 1, m + p - 1) "[REDACTED]@"
        s = substr(s, m + n)
      }
      return out s
    }
    function bearer(s,   out, m, n) {
      out = ""
      while (match(s, /[Bb]earer[[:space:]]+[A-Za-z0-9._~+\/=-]+/)) {
        m = RSTART; n = RLENGTH
        match(substr(s, m, n), /^[Bb]earer[[:space:]]+/)
        out = out substr(s, 1, m + RLENGTH - 1) "[REDACTED]"
        s = substr(s, m + n)
      }
      return out s
    }
    {
      line = $0
      if (match(tolower(line), /^[[:space:]]*[-]?[[:space:]]*[a-z0-9_.-]*(password|passwd|secret|token|key|creds|credential|honeycomb|authorization|bearer|passphrase|client_id)[a-z0-9_.-]*[[:space:]]*:[[:space:]]*[^[:space:]]/))
        line = substr(line, 1, RLENGTH - 1) "\"[REDACTED]\""
      line = url_creds(line)
      line = bearer(line)
      gsub(/AKIA[0-9A-Z][0-9A-Z][0-9A-Z][0-9A-Z][0-9A-Z][0-9A-Z][0-9A-Z][0-9A-Z][0-9A-Z][0-9A-Z][0-9A-Z][0-9A-Z][0-9A-Z][0-9A-Z][0-9A-Z][0-9A-Z]/, "[REDACTED-AWS-ACCESS-KEY]", line)
      gsub(/eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+/, "[REDACTED-JWT]", line)
      print inline_keys(line)
    }
  ' "$f" > "$f.redact.$$" || { rm -f "$f.redact.$$"; error_exit "$LINENO" "redaction failed for $f"; }
  mv "$f.redact.$$" "$f"
}

# Copy a file into the staging dir under a target name, redact it, append to the
# tar. No-op when the source is absent. Args: src, name-in-tar, tar, stage.
stage_redacted() {
    [ -f "$1" ] || return 0
    info "Adding $(fg_cyan "$1")$(reset) (redacted)"
    cp "$1" "$4/$2"
    redact_in_place "$4/$2"
    tar -rf "$3" -C "$4" "$2"
}

bundle_files() {
    banner "Collecting files for support bundle"
    increase_indent

    prompt "Do you want to include only the most recent logs (y or n)? "; read -r response
    increase_indent
    tar_filename="support_bundle_$(date +%Y%m%d_%H%M%S).tar"
    # Stage logs, redact copies, tar from the stage. Originals untouched.
    log_stage="sb_logs_$$"
    mkdir -p "$log_stage"

    if [ "$collector_version" = v2 ]; then
        # v2 logs: supervisor.log in the install dir plus the supervisor_storage logs.
        have_v2_logs=false
        for log_file in "$collector_dir/supervisor.log" "$collector_dir"/supervisor_storage/*.log; do
            [ -f "$log_file" ] && have_v2_logs=true
        done
        if [ "$have_v2_logs" = false ]; then
            info "No logs found for the v2 collector in $(fg_red "$collector_dir")$(reset)"
        elif [ "$response" = "n" ]; then
            info "Collecting all v2 collector logs"
            for log_file in "$collector_dir/supervisor.log" "$collector_dir"/supervisor_storage/*.log; do
                [ -f "$log_file" ] && cp "$log_file" "$log_stage/"
            done
        else
            # supervisor.log (the supervisor) and supervisor_storage/agent.log (the
            # collector) are distinct live logs. Collect supervisor.log plus the
            # newest supervisor_storage log so neither is dropped by a race.
            if [ -f "$collector_dir/supervisor.log" ]; then
                cp "$collector_dir/supervisor.log" "$log_stage/"
                info "Added file $(fg_cyan "$collector_dir/supervisor.log")$(reset) to the tar file."
            fi
            newest_storage=""
            for log_file in "$collector_dir"/supervisor_storage/*.log; do
                [ -f "$log_file" ] || continue
                # -nt is in POSIX.1-2024 and dash, ksh88 (AIX) and bash support it. Older shellcheck flags it.
                # shellcheck disable=SC3013
                { [ -z "$newest_storage" ] || [ "$log_file" -nt "$newest_storage" ]; } && newest_storage="$log_file"
            done
            if [ -n "$newest_storage" ]; then
                cp "$newest_storage" "$log_stage/"
                info "Added file $(fg_cyan "$newest_storage")$(reset) to the tar file."
            fi
        fi
    else
        # v1 logs live under $collector_dir/log.
        log_dir="$collector_dir/log"
        if [ ! -d "$log_dir" ]; then
            info "Directory ($fg_red $log_dir)$(reset) does not exist."
            prompt "Please enter an existing directory for logs: "; read -r log_dir
            if [ ! -d "$log_dir" ]; then
                echo "Directory $log_dir does not exist."
                rm -rf "$log_stage"
                return 1
            fi
        fi
        if [ "$response" = "n" ]; then
            # Get all the log files
            info "Collecting all log files in $(fg_cyan "$log_dir")$(reset)"
            for log_file in "$log_dir"/*; do
                [ -f "$log_file" ] && cp "$log_file" "$log_stage/"
            done
        else
            # Get the most recent log file. The error logs are collected below,
            # and the collector recreates them on every start, so skip them here.
            recent_log=""
            for log_file in "$log_dir"/*; do
                [ -f "$log_file" ] || continue
                case "${log_file##*/}" in observiq_collector.err*) continue ;; esac
                # -nt is in POSIX.1-2024 and dash, ksh88 (AIX) and bash support it. Older shellcheck flags it.
                # shellcheck disable=SC3013
                { [ -z "$recent_log" ] || [ "$log_file" -nt "$recent_log" ]; } && recent_log="$log_file"
            done
            if [ -n "$recent_log" ]; then
                cp "$recent_log" "$log_stage/"
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
            tar -rf "$tar_filename" $file
            rm $file
            info "Added file $(fg_cyan "/etc/$file")$(reset) to the tar file."
        else
            info "File $(fg_red "/etc/$file")$(reset) does not exist."
        fi
    done

    # Stage config/manager/service logs in a temp dir, not the working dir.
    file_stage="sb_files_$$"
    mkdir -p "$file_stage"

    # Config artifacts differ by version: v1 ships config.yaml + manager.yaml,
    # v2 ships supervisor.yaml + supervisor_storage/effective.yaml. Older v2
    # installs named the supervisor config supervisor_config.yaml.
    if [ "$collector_version" = v2 ]; then
        config_one="$collector_dir/supervisor.yaml"
        [ -f "$config_one" ] || config_one="$collector_dir/supervisor_config.yaml"
        config_two="$collector_dir/supervisor_storage/effective.yaml"
    else
        config_one="$collector_dir/config.yaml"
        config_two="$collector_dir/manager.yaml"
    fi
    if [ -f "$config_one" ] || [ -f "$config_two" ]; then
        prompt "Do you want to include the collector config and manager files (y or n)? "; read -r response
        if [ "$response" != "n" ]; then
            stage_redacted "$config_one" "$(basename "$config_one")" "$tar_filename" "$file_stage"
            stage_redacted "$config_two" "$(basename "$config_two")" "$tar_filename" "$file_stage"
        fi
    fi

    if [ "$OS_TYPE" = AIX ]; then
        # AIX has no journal. Record the SRC status and subsystem definition,
        # plus the error log's SRC entries, which record each subsystem exit.
        info "Collecting the SRC status and error log..."
        service_log=src_status.log
        {
            lssrc -s "$collector_service" 2>&1 || true
            lssrc -S -s "$collector_service" 2>&1 || true
            echo "=== errpt -N SRC ==="
            errpt -N SRC 2>&1 | head -n 50 || true
        } > "$file_stage/$service_log"
        oslevel -s > "$file_stage/oslevel.txt" 2>/dev/null || true
        tar -rf "$tar_filename" -C "$file_stage" oslevel.txt
    else
        # Grab the logs from journalctl -- in some cases, the collector.log file
        # may be empty, but there may be logs in journalctl
        info "Collecting logs from journalctl..."
        service_log=journalctl.log
        journalctl -u "$collector_service" -n 50 > "$file_stage/$service_log"
    fi
    redact_in_place "$file_stage/$service_log"
    tar -rf "$tar_filename" -C "$file_stage" "$service_log"
    rm -rf "$file_stage"

    collect_profiles "$tar_filename"

    collect_handles_limits "$tar_filename"

    collect_system_stats "$tar_filename"

    # Compress the tar file
    info "Compressing the tar file..."
    gzip "$tar_filename"

    info "Files have been added to the file $(pwd)/$tar_filename.gz successfully."
    decrease_indent
}

collect_profiles() {
  prompt "Collect go pprof profiles? (y/n) "; read -r PPROF
  case "$PPROF" in y*) ;; *) return 0 ;; esac
  if ! command -v curl >/dev/null; then
    info "curl is not installed, so profiles cannot be collected"
    return 0
  fi
  tar_filename="$1"
  increase_indent
  # POSIX prompt
  info "For endpoint, please use the format http://localhost:1777 or https://localhost:1777\n"
  info "where 1777 is the port you configured in your profile extension (1777 is the default)"
  prompt "Endpoint: "; read -r ENDPOINT
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

  info "Profile files have been added to the file $(pwd)/$tar_filename successfully."
  decrease_indent
}

# Collect the collector's open file descriptors and resource limits.
# PROC defaults to /proc (overridable for tests).
collect_handles_limits() {
  prompt "Collect the collector's open files and resource limits? (y or n) "; read -r HL
  case "$HL" in y*) ;; *) return 0 ;; esac
  tar_filename="$1"
  increase_indent
  service="${collector_service:-$(service_name "$V1_SERVICE")}"
  proc="${PROC:-/proc}"
  stage="sb_handles_$$"
  mkdir -p "$stage"

  pid=$(service_pid "$service")
  if [ "$OS_TYPE" = AIX ]; then
    # SRC runs the collector as root, so root's limits are the configured ones.
    # Works even when the collector is stopped.
    lsuser -a fsize cpu data stack core rss nofiles fsize_hard cpu_hard data_hard stack_hard core_hard rss_hard nofiles_hard root \
      > "$stage/user_limits.txt" 2>&1 || true
  else
    # Configured limits, all soft/hard. Works even when the collector is stopped.
    # grep '^Limit' is deliberate: it drops Environment= (which carries the OpAMP
    # secret key) from the output. Do not broaden this filter.
    systemctl show "$service" 2>/dev/null | grep '^Limit' > "$stage/systemd_limits.txt" || true
  fi

  if [ -n "$pid" ] && [ "$pid" != "0" ] && [ -d "$proc/$pid" ]; then
    info "Collecting open files and limits for pid $(fg_cyan "$pid")$(reset)"
    if [ "$OS_TYPE" = AIX ]; then
      # AIX /proc has no limits file. procfiles reports the open file rlimit.
      procfiles -n "$pid" > "$stage/procfiles.txt" 2>&1 || true
    else
      cat "$proc/$pid/limits" > "$stage/proc_limits.txt" 2>/dev/null || true
    fi
    # /proc/<pid>/fd is owner-only; capture stderr so a permission failure is
    # distinguishable from a genuinely empty listing rather than a silent 0-byte file.
    ls -l "$proc/$pid/fd" > "$stage/open_fds.txt" 2>"$stage/open_fds.err" || true
    [ -s "$stage/open_fds.err" ] || rm -f "$stage/open_fds.err"
    if command -v lsof >/dev/null; then
      lsof -p "$pid" > "$stage/lsof.txt" 2>"$stage/lsof.err" || true
      [ -s "$stage/lsof.err" ] || rm -f "$stage/lsof.err"
    fi
  else
    info "Collector process not running; collected configured limits only"
  fi

  # Redact staged files before bundling, per the bundle-wide redaction rule (#3655).
  for f in "$stage"/*; do
    [ -f "$f" ] && redact_in_place "$f"
  done
  tar -rf "$tar_filename" -C "$stage" .
  rm -rf "$stage"
  info "Handle and limit files have been added to the file $(pwd)/$tar_filename"
  decrease_indent
}

# Collect CPU, memory, and disk stats. Always collected (cheap, non-sensitive).
# PROC defaults to /proc (overridable for tests).
collect_system_stats() {
  tar_filename="$1"
  proc="${PROC:-/proc}"
  stage="sb_stats_$$"
  mkdir -p "$stage"
  info "Collecting CPU, memory, and disk stats..."
  # Each collector is best-effort; a missing source must not abort the bundle.
  if [ "$OS_TYPE" = AIX ]; then
    # AIX /proc has no loadavg or meminfo, and its df has no -h.
    {
      echo "=== lparstat -i ==="; lparstat -i 2>/dev/null || true
      echo "=== uptime ==="; uptime 2>/dev/null || true
      echo "=== vmstat ==="; vmstat 1 1 2>/dev/null || true
      echo "=== svmon -G ==="; svmon -G 2>/dev/null || true
      echo "=== disk usage (all filesystems) ==="; df -g 2>/dev/null || true
      echo "=== disk usage (collector dir) ==="; df -g "$collector_dir" 2>/dev/null || true
    } > "$stage/system_stats.txt"
  else
    {
      echo "=== cpu count ==="; nproc 2>/dev/null || true
      echo "=== loadavg ==="; cat "$proc/loadavg" 2>/dev/null || true
      echo "=== meminfo ==="; cat "$proc/meminfo" 2>/dev/null || true
      echo "=== disk usage (all filesystems) ==="; df -h 2>/dev/null || true
      echo "=== disk usage (collector dir) ==="; df -h "$collector_dir" 2>/dev/null || true
    } > "$stage/system_stats.txt"
  fi
  # Redact before bundling, per the bundle-wide redaction rule (#3655).
  redact_in_place "$stage/system_stats.txt"
  tar -rf "$tar_filename" -C "$stage" system_stats.txt
  rm -rf "$stage"
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
  detect_install
  bundle_files
}

# Tests source everything above this line, so main must stay the last line.
main "$@"
