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

# Collector Constants
SERVICE_NAME="com.observiq.collector"
DOWNLOAD_BASE="https://github.com/observIQ/bindplane-otel-collector/releases/download"

# Script Constants
PREREQS="printf sed uname tr find grep"
TMP_DIR="${TMPDIR:-"/tmp/"}observiq-otel-collector" # Allow this to be overriden by cannonical TMPDIR env var
INSTALL_DIR="/opt/observiq-otel-collector"
SUPERVISOR_YML_PATH="$INSTALL_DIR/supervisor.yaml"
SCRIPT_NAME="$0"
INDENT_WIDTH='  '
indent=""
non_interactive=false
error_mode=false

# Colors
if [ "$non_interactive" = "false" ]; then
  num_colors=$(tput colors 2>/dev/null)
  if test -n "$num_colors" && test "$num_colors" -ge 8; then
    bold="$(tput bold)"
    underline="$(tput smul)"
    # standout can be bold or reversed colors dependent on terminal
    standout="$(tput smso)"
    reset="$(tput sgr0)"
    bg_black="$(tput setab 0)"
    bg_blue="$(tput setab 4)"
    bg_cyan="$(tput setab 6)"
    bg_green="$(tput setab 2)"
    bg_magenta="$(tput setab 5)"
    bg_red="$(tput setab 1)"
    bg_white="$(tput setab 7)"
    bg_yellow="$(tput setab 3)"
    fg_black="$(tput setaf 0)"
    fg_blue="$(tput setaf 4)"
    fg_cyan="$(tput setaf 6)"
    fg_green="$(tput setaf 2)"
    fg_magenta="$(tput setaf 5)"
    fg_red="$(tput setaf 1)"
    fg_white="$(tput setaf 7)"
    fg_yellow="$(tput setaf 3)"
  fi
fi

if [ -z "$reset" ]; then
  sed_ignore=''
else
  sed_ignore="/^[$reset]+$/!"
fi

# Helper Functions
printf() {
  if [ "$non_interactive" = "false" ] || [ "$error_mode" = "true" ]; then
    if command -v sed >/dev/null; then
      command printf -- "$@" | sed -E "$sed_ignore s/^/$indent/g" # Ignore sole reset characters if defined
    else
      # Ignore $* suggestion as this breaks the output
      # shellcheck disable=SC2145
      command printf -- "$indent$@"
    fi
  fi
}

increase_indent() { indent="$INDENT_WIDTH$indent"; }
decrease_indent() { indent="${indent#*"$INDENT_WIDTH"}"; }

# Color functions reset only when given an argument
bold() { command printf "$bold$*$(if [ -n "$1" ]; then command printf "$reset"; fi)"; }
underline() { command printf "$underline$*$(if [ -n "$1" ]; then command printf "$reset"; fi)"; }
standout() { command printf "$standout$*$(if [ -n "$1" ]; then command printf "$reset"; fi)"; }
# Ignore "parameters are never passed"
# shellcheck disable=SC2120
reset() { command printf "$reset$*$(if [ -n "$1" ]; then command printf "$reset"; fi)"; }
bg_black() { command printf "$bg_black$*$(if [ -n "$1" ]; then command printf "$reset"; fi)"; }
bg_blue() { command printf "$bg_blue$*$(if [ -n "$1" ]; then command printf "$reset"; fi)"; }
bg_cyan() { command printf "$bg_cyan$*$(if [ -n "$1" ]; then command printf "$reset"; fi)"; }
bg_green() { command printf "$bg_green$*$(if [ -n "$1" ]; then command printf "$reset"; fi)"; }
bg_magenta() { command printf "$bg_magenta$*$(if [ -n "$1" ]; then command printf "$reset"; fi)"; }
bg_red() { command printf "$bg_red$*$(if [ -n "$1" ]; then command printf "$reset"; fi)"; }
bg_white() { command printf "$bg_white$*$(if [ -n "$1" ]; then command printf "$reset"; fi)"; }
bg_yellow() { command printf "$bg_yellow$*$(if [ -n "$1" ]; then command printf "$reset"; fi)"; }
fg_black() { command printf "$fg_black$*$(if [ -n "$1" ]; then command printf "$reset"; fi)"; }
fg_blue() { command printf "$fg_blue$*$(if [ -n "$1" ]; then command printf "$reset"; fi)"; }
fg_cyan() { command printf "$fg_cyan$*$(if [ -n "$1" ]; then command printf "$reset"; fi)"; }
fg_green() { command printf "$fg_green$*$(if [ -n "$1" ]; then command printf "$reset"; fi)"; }
fg_magenta() { command printf "$fg_magenta$*$(if [ -n "$1" ]; then command printf "$reset"; fi)"; }
fg_red() { command printf "$fg_red$*$(if [ -n "$1" ]; then command printf "$reset"; fi)"; }
fg_white() { command printf "$fg_white$*$(if [ -n "$1" ]; then command printf "$reset"; fi)"; }
fg_yellow() { command printf "$fg_yellow$*$(if [ -n "$1" ]; then command printf "$reset"; fi)"; }

# Intentionally using variables in format string
# shellcheck disable=SC2059
info() { printf "$*\\n"; }
# Intentionally using variables in format string
# shellcheck disable=SC2059
warn() {
  increase_indent
  printf "$fg_yellow$*$reset\\n"
  decrease_indent
}
# Intentionally using variables in format string
# shellcheck disable=SC2059
error() {
  increase_indent
  error_mode=true
  printf "$fg_red$*$reset\\n"
  error_mode=false
  decrease_indent
}
# Intentionally using variables in format string
# shellcheck disable=SC2059
success() { printf "$fg_green$*$reset\\n"; }
# Ignore 'arguments are never passed'
# shellcheck disable=SC2120
prompt() {
  if [ "$1" = 'n' ]; then
    command printf "y/$(fg_red '[n]'): "
  else
    command printf "$(fg_green '[y]')/n: "
  fi
}

bindplane_banner() {
  if [ "$non_interactive" = "false" ]; then
    fg_cyan " oooooooooo.   o8o                    .o8  ooooooooo.   oooo\\n"
    fg_cyan " '888'   '88b  '\"'                   \"888  '888   'Y88. '888\\n"
    fg_cyan "  888     888 oooo  ooo. .oo.    .oooo888   888   .d88'  888   .oooo.   ooo. .oo.    .ooooo.\\n"
    fg_cyan "  888oooo888' '888  '888P\"Y88b  d88' '888   888ooo88P'   888  'P  )88b  '888P\"Y88b  d88' '88b\\n"
    fg_cyan "  888    '88b  888   888   888  888   888   888          888   .oP\"888   888   888  888ooo888\\n"
    fg_cyan "  888    .88P  888   888   888  888   888   888          888  d8(  888   888   888  888    .o\\n"
    fg_cyan " o888bood8P'  o888o o888o o888o 'Y8bod88P\" o888o        o888o 'Y888\"\"8o o888o o888o '88bod8P'\\n"

    reset
  fi
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
  $(fg_yellow '-v, --version')
      Defines the version of the BindPlane Agent.
      If not provided, this will default to the latest version.
      Alternatively the COLLECTOR_VERSION environment variable can be
      set to configure the agent version.
      Example: '-v 1.2.12' will download 1.2.12.

  $(fg_yellow '-r, --uninstall')
      Stops the agent services and uninstalls the agent.

  $(fg_yellow '-l, --url')
      Defines the URL that the components will be downloaded from.
      If not provided, this will default to BindPlane Agent\'s GitHub releases.
      Example: '-l http://my.domain.org/observiq-otel-collector' will download from there.

  $(fg_yellow '-b, --base-url')
      Defines the base of the download URL as '{base_url}/v{version}/observiq-otel-collector-v{version}-darwin-{os_arch}.tar.gz'.
      If not provided, this will default to '$DOWNLOAD_BASE'.
      Example: '-b http://my.domain.org/observiq-otel-collector/binaries' will be used as the base of the download URL.
    
  $(fg_yellow '-f, --file')
      Install Agent from a local file instead of downloading from a URL.

  $(fg_yellow '-e, --endpoint')
      Defines the endpoint of an OpAMP compatible management server for this agent install.
      This parameter may also be provided through the ENDPOINT environment variable.
      
      Specifying this will install the agent in a managed mode, as opposed to the
      normal headless mode.
  
  $(fg_yellow '-k, --labels')
      Defines a list of comma seperated labels to be used for this agent when communicating 
      with an OpAMP compatible server.
      
      This parameter may also be provided through the LABELS environment variable.
      The '--endpoint' flag must be specified if this flag is specified.
    
  $(fg_yellow '-s, --secret-key')
    Defines the secret key to be used when communicating with an OpAMP compatible server.
    
    This parameter may also be provided through the SECRET_KEY environment variable.
    The '--endpoint' flag must be specified if this flag is specified.

  $(fg_yellow '-c, --check-bp-url')
    Check access to the BindPlane server URL.

    This parameter will have the script check access to BindPlane based on the provided '--endpoint'

  $(fg_yellow '-q, --quiet')
    Use quiet (non-interactive) mode to run the script in headless environments

  $(fg_yellow '-i, --clean-install')
    Do a clean install of the agent regardless of if a supervisor.yaml config file is already present.

  $(fg_yellow '-u --dirty-install')
    Do a dirty install by not generating a supervisor.yaml. Useful when one already exists and treats the install as an update.

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
  shift 2
  if [ -n "$0" ]; then
    increase_indent
    error "$*"
    decrease_indent
  fi
  force_exit
}

print_prereq_line() {
  if [ -n "$2" ]; then
    command printf "\\n${indent}  - "
    command printf "[$1]: $2"
  fi
}

check_failure() {
  if [ "$indent" != '' ]; then increase_indent; fi
  command printf "${indent}${fg_red}ERROR: %s check failed!${reset}" "$1"

  print_prereq_line "Issue" "$2"
  print_prereq_line "Resolution" "$3"
  print_prereq_line "Help Link" "$4"
  print_prereq_line "Rerun" "$5"

  command printf "\\n"
  if [ "$indent" != '' ]; then decrease_indent; fi
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

# This will check all prerequisites before running an installation.
check_prereqs() {
  banner "Checking Prerequisites"
  increase_indent
  root_check
  os_check
  dependencies_check
  success "Prerequisite check complete!"
  decrease_indent
}

# Test non-interactive mode compatibility
interactive_check() {
  # Incompatible with checking the BP url since it can be interactive on failed connection
  if [ "$non_interactive" = "true" ] && [ "$check_bp_url" = "true" ]; then
    failed
    error_exit "$LINENO" "Checking the BindPlane server URL is not compatible with quiet (non-interactive) mode."
  fi
}

# Test connection to BindPlane if it was specified
connection_check() {
  if [ "$check_bp_url" = "true" ]; then
    if [ -n "$opamp_endpoint" ]; then
      HTTP_ENDPOINT="$(echo "${opamp_endpoint}" | sed 's#^ws#http#' | sed 's#/v1/opamp$##')"
      info "Testing connection to BindPlane: $fg_magenta$HTTP_ENDPOINT$reset..."

      if curl --max-time 20 -s "${HTTP_ENDPOINT}" >/dev/null; then
        succeeded
      else
        failed
        warn "Connection to BindPlane has failed."
        increase_indent
        printf "%sDo you wish to continue installation?%s  " "$fg_yellow" "$reset"
        prompt "n"
        decrease_indent
        read -r input
        printf "\\n"
        if [ "$input" = "y" ] || [ "$input" = "Y" ]; then
          info "Continuing installation."
        else
          error_exit "$LINENO" "Aborting due to user input after connectivity failure between this system and the BindPlane server."
        fi
      fi
    fi
  fi
}

# This will check if the operating system is supported.
os_check() {
  info "Checking that the operating system is supported..."
  os_type=$(uname -s)
  case "$os_type" in
  Darwin)
    succeeded
    ;;
  *)
    failed
    error_exit "$LINENO" "The operating system $(fg_yellow "$os_type") is not supported by this script."
    ;;
  esac
}

# This checks to see if the user who is running the script has root permissions.
root_check() {
  system_user_name=$(id -un)
  if [ "${system_user_name}" != 'root' ]; then
    failed
    error_exit "$LINENO" "Script needs to be run as root or with sudo"
  fi
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

# This will set all installation variables
# at the beginning of the script.
setup_installation() {
  banner "Configuring Installation Variables"
  increase_indent

  set_os_arch

  if [ -z "$package_path" ]; then
    set_download_urls
    out_file_path="$TMP_DIR/observiq-otel-collector.tar.gz"
  else
    out_file_path="$package_path"
  fi

  set_opamp_endpoint
  set_opamp_labels
  set_opamp_secret_key

  ask_clean_install

  success "Configuration complete!"
  decrease_indent
}

set_os_arch() {
  os_arch=$(uname -m)
  case "$os_arch" in
  # arm64 strings. These are from https://stackoverflow.com/questions/45125516/possible-values-for-uname-m
  aarch64 | arm64 | aarch64_be | armv8b | armv8l)
    os_arch="arm64"
    ;;
  x86_64)
    os_arch="amd64"
    ;;
  *)
    # We only support arm64/amd64 architectures for macOS
    error_exit "$LINENO" "Unsupported os arch: $os_arch"
    ;;
  esac
}

# This will set the urls to use when downloading the agent and its plugins.
# These urls are constructed based on the --version flag or COLLECTOR_VERSION env variable.
# If not specified, the version defaults to whatever the latest release on github is.
set_download_urls() {
  if [ -z "$url" ]; then
    if [ -z "$version" ]; then
      # shellcheck disable=SC2153
      version=$COLLECTOR_VERSION
    fi

    if [ -z "$version" ]; then
      version=$(latest_version)
    fi

    if [ -z "$version" ]; then
      error_exit "$LINENO" "Could not determine version to install"
    fi

    if [ -z "$base_url" ]; then
      base_url=$DOWNLOAD_BASE
    fi

    collector_download_url="$base_url/v$version/observiq-otel-collector-v${version}-darwin-${os_arch}.tar.gz"
  else
    collector_download_url="$url"
  fi
}

set_opamp_endpoint() {
  if [ -z "$opamp_endpoint" ]; then
    opamp_endpoint="$ENDPOINT"
  fi

  OPAMP_ENDPOINT="$opamp_endpoint"
}

set_opamp_labels() {
  if [ -z "$opamp_labels" ]; then
    opamp_labels=$LABELS
  fi

  OPAMP_LABELS="$opamp_labels"

  if [ -n "$OPAMP_LABELS" ] && [ -z "$OPAMP_ENDPOINT" ]; then
    error_exit "$LINENO" "An endpoint must be specified when providing labels"
  fi
}

set_opamp_secret_key() {
  if [ -z "$opamp_secret_key" ]; then
    opamp_secret_key=$SECRET_KEY
  fi

  OPAMP_SECRET_KEY="$opamp_secret_key"

  if [ -n "$OPAMP_SECRET_KEY" ] && [ -z "$OPAMP_ENDPOINT" ]; then
    error_exit "$LINENO" "An endpoint must be specified when providing a secret key"
  fi
}

# If an existing supervisor.yaml is present, ask whether we should do a clean install.
# Want to avoid inadvertanly overwriting endpoint or secret_key values.
ask_clean_install() {
  if [ "$clean_install" = "true" ] || [ "$clean_install" = "false" ]; then
    # install type already set, so just return
    return
  fi

  if [ -f "$SUPERVISOR_YML_PATH" ]; then
    command printf "${indent}An installation already exists. Would you like to do a clean install? $(prompt n)"
    read -r clean_install_response
    clean_install_response=$(echo "$clean_install_response" | tr '[:upper:]' '[:lower:]')
    case $clean_install_response in
    y | yes)
      increase_indent
      success "Doing clean install!"
      decrease_indent
      clean_install="true"
      ;;
    *)
      warn "Doing upgrade instead of clean install"
      clean_install="false"
      ;;
    esac
  else
    warn "Previous supervisor config not found, doing clean install"
    clean_install="true"
  fi
}

# latest_version gets the tag of the latest release, without the v prefix.
latest_version() {
  curl -sSL -H"Accept: application/vnd.github.v3+json" https://api.github.com/repos/observIQ/bindplane-agent/releases/latest |
    grep "\"tag_name\"" |
    sed -E 's/ *"tag_name": "v([0-9]+\.[0-9]+\.[0-9+])",/\1/'
}

# This will install the package by downloading & unpacking the tarball into the install directory
install_package() {
  banner "Installing BindPlane Agent"
  increase_indent

  # Remove temporary directory, if it exists
  rm -rf "$TMP_DIR"
  mkdir -p "$TMP_DIR/artifacts"

  # Download into tmp dir
  if [ -z "$package_path" ]; then
    info "Downloading tarball into temporary directory..."
    curl -L "$collector_download_url" -o "$out_file_path" --progress-bar --fail || error_exit "$LINENO" "Failed to download package"
    succeeded
  fi

  # unpack
  info "Unpacking tarball..."
  tar -xzf "$out_file_path" -C "$TMP_DIR/artifacts" || error_exit "$LINENO" "Failed to unpack archive $out_file_path"
  succeeded

  mkdir -p "$INSTALL_DIR" || error_exit "$LINENO" "Failed to create directory $INSTALL_DIR"

  info "Creating install directory structure..."
  increase_indent
  # Find all directorys in the unpackaged tar
  DIRS=$(
    cd "$TMP_DIR/artifacts"
    find "." -type d
  )
  for d in $DIRS; do
    mkdir -p "$INSTALL_DIR/$d" || error_exit "$LINENO" "Failed to create directory $INSTALL_DIR/$d"
  done

  # Create the storage dir; This dir is necessary for filelog based plugins
  mkdir -p "$INSTALL_DIR/storage" || error_exit "$LINENO" "Failed to create directory $INSTALL_DIR/storage"

  decrease_indent
  succeeded

  info "Copying artifacts to install directory..."
  increase_indent

  # This find command gets a list of all artifacts paths except opentelemetry-java-contrib-jmx-metrics.jar
  FILES=$(
    cd "$TMP_DIR/artifacts"
    find "." -type f -not \( -name opentelemetry-java-contrib-jmx-metrics.jar \)
  )
  # Move files to install dir
  for f in $FILES; do
    rm -rf "$INSTALL_DIR/${f:?}"
    cp "$TMP_DIR/artifacts/${f:?}" "$INSTALL_DIR/${f:?}" || error_exit "$LINENO" "Failed to copy artifact $f to install dir"
  done
  decrease_indent
  succeeded

  # If an endpoint was specified, we need to write the supervisor.yaml
  if [ -n "$OPAMP_ENDPOINT" ]; then
    create_supervisor_config "$SUPERVISOR_YML_PATH"
  fi

  # Install jmx jar
  info "Moving opentelemetry-java-contrib-jmx-metrics.jar to /opt..."
  mv "$TMP_DIR/artifacts/opentelemetry-java-contrib-jmx-metrics.jar" "/opt/opentelemetry-java-contrib-jmx-metrics.jar" || error_exit "$LINENO" "Failed to copy opentelemetry-java-contrib-jmx-metrics.jar to /opt"
  succeeded

  if [ -f "/Library/LaunchDaemons/$SERVICE_NAME.plist" ]; then
    # Existing service file, we should stop & unload first.
    info "Uninstalling existing service file..."
    launchctl unload -w "/Library/LaunchDaemons/$SERVICE_NAME.plist" >/dev/null 2>&1 || error_exit "$LINENO" "Failed to unload service file /Library/LaunchDaemons/$SERVICE_NAME.plist"
    succeeded
  fi

  # Install service file
  info "Installing service file..."
  sed "s|\\[INSTALLDIR\\]|${INSTALL_DIR}/|g" "$INSTALL_DIR/install/$SERVICE_NAME.plist" | tee "/Library/LaunchDaemons/$SERVICE_NAME.plist" >/dev/null 2>&1 || error_exit "$LINENO" "Failed to install service file"
  launchctl load -w "/Library/LaunchDaemons/$SERVICE_NAME.plist" >/dev/null 2>&1 || error_exit "$LINENO" "Failed to load service file /Library/LaunchDaemons/$SERVICE_NAME.plist"
  succeeded

  info "Starting service..."
  launchctl start "$SERVICE_NAME" || error_exit "$LINENO" "Failed to start service file $SERVICE_NAME"
  succeeded

  info "Removing temporary files..."
  rm -rf "$TMP_DIR" || error_exit "$LINENO" "Failed to remove temp dir: $TMP_DIR"
  succeeded

  success "BindPlane Agent installation complete!"
  decrease_indent
}

create_supervisor_config() {
  supervisor_yml_path="$1"

  # Return if we're not doing a clean install
  if [ "$clean_install" = "false" ]; then
    return
  fi

  info "Creating supervisor config..."

  # Note here: We create the file and change permissions of the file here BEFORE writing info to it.
  # We do this because the file contains the secret key.
  # We do not want the file readable by anyone other than root.
  command printf '' >>"$supervisor_yml_path"
  chmod 0600 "$supervisor_yml_path"

  command printf 'server:\n' >"$supervisor_yml_path"
  command printf '  endpoint: "%s"\n' "$OPAMP_ENDPOINT" >>"$supervisor_yml_path"
  command printf '  headers:\n' >>"$supervisor_yml_path"
  [ -n "$OPAMP_SECRET_KEY" ] && command printf '    Authorization: "Secret-Key %s"\n' "$OPAMP_SECRET_KEY" >>"$supervisor_yml_path"
  # [ -n "$OPAMP_LABELS" ] && command printf '    X-Bindplane-Attribute: "%s"\n' "$OPAMP_LABELS" >> "$supervisor_yml_path"
  command printf '  tls:\n' >>"$supervisor_yml_path"
  command printf '    insecure: true\n' >>"$supervisor_yml_path"
  command printf '    insecure_skip_verify: true\n' >>"$supervisor_yml_path"
  command printf 'capabilities:\n' >>"$supervisor_yml_path"
  command printf '  accepts_remote_config: true\n' >>"$supervisor_yml_path"
  command printf '  reports_remote_config: true\n' >>"$supervisor_yml_path"
  command printf 'agent:\n' >>"$supervisor_yml_path"
  command printf '  executable: "%s"\n' "$INSTALL_DIR/observiq-otel-collector" >>"$supervisor_yml_path"
  command printf '  description:\n' >>"$supervisor_yml_path"
  command printf '    non_identifying_attributes:\n' >>"$supervisor_yml_path"
  [ -n "$OPAMP_LABELS" ] && command printf '      service.labels: "%s"\n' "$OPAMP_LABELS" >>"$supervisor_yml_path"
  command printf 'storage:\n' >>"$supervisor_yml_path"
  command printf '  directory: "%s"\n' "$INSTALL_DIR/supervisor_storage" >>"$supervisor_yml_path"
  command printf 'telemetry:\n' >>"$supervisor_yml_path"
  command printf '  logs:\n' >>"$supervisor_yml_path"
  command printf '    level: 0\n' >>"$supervisor_yml_path"
  command printf '    output_paths: ["%s"]' "$INSTALL_DIR/supervisor_storage/supervisor.log" >>"$supervisor_yml_path"
  succeeded
}

# This will display the results of an installation
display_results() {
  banner 'Information'
  increase_indent
  info "Agent Home:                    $(fg_cyan "$INSTALL_DIR")$(reset)"
  info "Agent Config:                  $(fg_cyan "$INSTALL_DIR/supervisor_storage/effective.yaml")$(reset)"
  info "Agent Logs Command:            $(fg_cyan "sudo tail -F $INSTALL_DIR/supervisor_storage/agent.log")$(reset)"
  info "Supervisor Start Command:      $(fg_cyan "sudo launchctl load /Library/LaunchDaemons/$SERVICE_NAME.plist")$(reset)"
  info "Supervisor Stop Command:       $(fg_cyan "sudo launchctl unload /Library/LaunchDaemons/$SERVICE_NAME.plist")$(reset)"
  decrease_indent

  banner 'Support'
  increase_indent
  info "For more information on configuring the agent, see the docs:"
  increase_indent
  info "$(fg_cyan "https://github.com/observIQ/bindplane-otel-collector/tree/main#bindplane-agent")$(reset)"
  decrease_indent
  info "If you have any other questions please contact us at $(fg_cyan support@observiq.com)$(reset)"
  decrease_indent

  banner "$(fg_green Installation Complete!)"
  return 0
}

uninstall() {
  banner "Uninstalling BindPlane Agent"
  increase_indent

  if [ ! -f "$INSTALL_DIR/observiq-otel-collector" ]; then
    # If the agent binary is not present, we assume that the agent is not installed
    # In this case, do nothing.
    info "No install detected, skipping..."
    decrease_indent
    banner "$(fg_green Uninstallation Complete!)"
    return 0
  fi

  info "Uninstalling service file..."
  launchctl unload -w "/Library/LaunchDaemons/$SERVICE_NAME.plist" || error_exit "$LINENO" "Failed to unload service file /Library/LaunchDaemons/$SERVICE_NAME.plist"
  rm -f "/Library/LaunchDaemons/$SERVICE_NAME.plist" || error_exit "$LINENO" "Failed to remove service file /Library/LaunchDaemons/$SERVICE_NAME.plist"
  succeeded

  # Removes the whole install directory
  info "Removing installed artifacts..."
  rm -rf /opt/observiq-otel-collector || error_exit "$LINENO" "Failed to remove /opt/observiq-otel-collector"
  succeeded

  info "Removing any existing log files"
  rm -f "/var/log/observiq_collector.err" || error_exit "$LINENO" "Failed to remove /var/log/observiq_collector.err"
  succeeded

  info "Removing opentelemetry-java-contrib-jmx-metrics.jar from /opt..."
  rm -f "/opt/opentelemetry-java-contrib-jmx-metrics.jar" || error_exit "$LINENO" "Failed to remove /opt/opentelemetry-java-contrib-jmx-metrics.jar"
  succeeded

  decrease_indent
  banner "$(fg_green Uninstallation Complete!)"
}

main() {
  # We do these checks before we process arguments, because
  # some of these options bail early, and we'd like to be sure that those commands
  # (e.g. uninstall) can run

  bindplane_banner
  check_prereqs

  if [ $# -ge 1 ]; then
    while [ -n "$1" ]; do
      case "$1" in
      -q | --quiet)
        non_interactive="true"
        shift 1
        ;;
      -l | --url)
        url=$2
        shift 2
        ;;
      -v | --version)
        version=$2
        shift 2
        ;;
      -r | --uninstall)
        uninstall
        exit 0
        ;;
      -h | --help)
        usage
        exit 0
        ;;
      -e | --endpoint)
        opamp_endpoint=$2
        shift 2
        ;;
      -k | --labels)
        opamp_labels=$2
        shift 2
        ;;
      -s | --secret-key)
        opamp_secret_key=$2
        shift 2
        ;;
      -c | --check-bp-url)
        check_bp_url="true"
        shift 1
        ;;
      -b | --base-url)
        base_url=$2
        shift 2
        ;;
      -i | --clean-install)
        clean_install="true"
        shift 1
        ;;
      -u | --dirty-install)
        clean_install="false"
        shift 1
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

  interactive_check
  connection_check
  setup_installation
  install_package
  display_results
}

main "$@"
