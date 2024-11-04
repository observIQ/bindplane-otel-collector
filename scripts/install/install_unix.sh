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

# Agent Constants
PACKAGE_NAME="observiq-otel-collector"
DOWNLOAD_BASE="https://github.com/observIQ/bindplane-otel-collector/releases/download"

# Determine if we need service or systemctl for prereqs
if command -v systemctl >/dev/null 2>&1; then
  SVC_PRE=systemctl
elif command -v service >/dev/null 2>&1; then
  SVC_PRE=service
fi

# Script Constants
TMP_DIR=${TMPDIR:-"/tmp"} # Allow this to be overriden by cannonical TMPDIR env var
INSTALL_DIR="/opt/observiq-otel-collector"
SUPERVISOR_YML_PATH="$INSTALL_DIR/supervisor.yaml"
PREREQS="curl printf $SVC_PRE sed uname cut"
SCRIPT_NAME="$0"
INDENT_WIDTH='  '
indent=""
non_interactive=false
error_mode=false

# out_file_path is the full path to the downloaded package (e.g. "/tmp/observiq-otel-collector_linux_amd64.deb")
out_file_path="unknown"

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
      command printf -- "$@" | sed -r "$sed_ignore s/^/$indent/g" # Ignore sole reset characters if defined
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
      Defines the base of the download URL as '{base_url}/v{version}/{PACKAGE_NAME}_v{version}_linux_{os_arch}.{package_type}'.
      If not provided, this will default to '$DOWNLOAD_BASE'.
      Example: '-b http://my.domain.org/observiq-otel-collector/binaries' will be used as the base of the download URL.

  $(fg_yellow '-f, --file')
      Install Agent from a local file instead of downloading from a URL.

  $(fg_yellow '-x, --proxy')
      Defines the proxy server to be used for communication by the install script.
      Example: $(fg_blue -x) $(fg_magenta http\(s\)://server-ip:port/).

  $(fg_yellow '-U, --proxy-user')
      Defines the proxy user to be used for communication by the install script.

  $(fg_yellow '-P, --proxy-password')
      Defines the proxy password to be used for communication by the install script.
    
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

# This will set all installation variables
# at the beginning of the script.
setup_installation() {
  banner "Configuring Installation Variables"
  increase_indent

  # Installation variables
  set_os_arch
  set_package_type

  # if package_path is not set then download the package
  if [ -z "$package_path" ]; then
    set_download_urls
    set_proxy
    set_file_name
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

set_file_name() {
  if [ -z "$version" ]; then
    package_file_name="${PACKAGE_NAME}_linux_${arch}.${package_type}"
  else
    package_file_name="${PACKAGE_NAME}_v${version}_linux_${arch}.${package_type}"
  fi
  out_file_path="$TMP_DIR/$package_file_name"
}

set_proxy() {
  if [ -n "$proxy" ]; then
    info "Using proxy from arguments: $proxy"
    if [ -n "$proxy_user" ]; then
      while [ -z "$proxy_password" ]; do
        increase_indent
        command printf "${indent}$(fg_blue "$proxy_user@$proxy")'s password: "
        stty -echo
        read -r proxy_password
        stty echo
        info
        if [ -z "$proxy_password" ]; then
          warn "The password must be provided!"
        fi
        decrease_indent
      done
      protocol="$(echo "$proxy" | cut -d'/' -f1)"
      host="$(echo "$proxy" | cut -d'/' -f3)"
      full_proxy="$protocol//$proxy_user:$proxy_password@$host"
    fi
  fi

  if [ -z "$full_proxy" ]; then
    full_proxy="$proxy"
  fi
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
  # experimental PowerPC arch support for collector
  ppc64)
    os_arch="ppc64"
    ;;
  ppc64le)
    os_arch="ppc64le"
    ;;
  # armv6/32bit. These are what raspberry pi can return, which is the main reason we support 32-bit arm
  arm | armv6l | armv7l)
    os_arch="arm"
    ;;
  *)
    error_exit "$LINENO" "Unsupported os arch: $os_arch"
    ;;
  esac
}

# Set the package type before install
set_package_type() {
  # if package_path is set get the file extension otherwise look at what's available on the system
  if [ -n "$package_path" ]; then
    case "$package_path" in
    *.deb)
      package_type="deb"
      ;;
    *.rpm)
      package_type="rpm"
      ;;
    *)
      error_exit "$LINENO" "Unsupported package type: $package_path"
      ;;
    esac
  else
    if command -v dpkg >/dev/null 2>&1; then
      package_type="deb"
    elif command -v rpm >/dev/null 2>&1; then
      package_type="rpm"
    else
      error_exit "$LINENO" "Could not find dpkg or rpm on the system"
    fi
  fi

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

    collector_download_url="$base_url/v$version/${PACKAGE_NAME}_v${version}_linux_${os_arch}.${package_type}"
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

# Test connection to BindPlane if it was specified
connection_check() {
  if [ -n "$check_bp_url" ]; then
    if [ -n "$opamp_endpoint" ]; then
      HTTP_ENDPOINT="$(echo "${opamp_endpoint}" | sed -z 's#^ws#http#' | sed -z 's#/v1/opamp$##')"
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

# This will check all prerequisites before running an installation.
check_prereqs() {
  banner "Checking Prerequisites"
  increase_indent
  root_check
  os_check
  os_arch_check
  package_type_check
  dependencies_check
  success "Prerequisite check complete!"
  decrease_indent
}

# This checks to see if the user who is running the script has root permissions.
root_check() {
  system_user_name=$(id -un)
  if [ "${system_user_name}" != 'root' ]; then
    failed
    error_exit "$LINENO" "Script needs to be run as root or with sudo"
  fi
}

# Test non-interactive mode compatibility
interactive_check() {
  # Incompatible with proxies unless both username and password are passed
  if [ "$non_interactive" = "true" ] && [ -n "$proxy_password" ]; then
    failed
    error_exit "$LINENO" "The proxy password must be set via the command line argument -P, if called non-interactively."
  fi

  # Incompatible with checking the BP url since it can be interactive on failed connection
  if [ "$non_interactive" = "true" ] && [ "$check_bp_url" = "true" ]; then
    failed
    error_exit "$LINENO" "Checking the BindPlane server URL is not compatible with quiet (non-interactive) mode."
  fi
}

# This will check if the operating system is supported.
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

# This will check if the system architecture is supported.
os_arch_check() {
  info "Checking for valid operating system architecture..."
  arch=$(uname -m)
  case "$arch" in
  x86_64 | aarch64 | ppc64 | ppc64le | arm64 | aarch64_be | armv8b | armv8l | arm | armv6l | armv7l)
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

# This will check to ensure either dpkg or rpm is installed on the system
package_type_check() {
  info "Checking for package manager..."
  if command -v dpkg >/dev/null 2>&1; then
    succeeded
  elif command -v rpm >/dev/null 2>&1; then
    succeeded
  else
    failed
    error_exit "$LINENO" "Could not find dpkg or rpm on the system"
  fi
}

# latest_version gets the tag of the latest release, without the v prefix.
latest_version() {
  curl -sSL -H"Accept: application/vnd.github.v3+json" https://api.github.com/repos/observIQ/observiq-otel-collector/releases/latest |
    grep "\"tag_name\"" |
    sed -r 's/ *"tag_name": "v([0-9]+\.[0-9]+\.[0-9+])",/\1/'
}

# This will install the package by downloading the archived agent,
# extracting the binaries, and then removing the archive.
install_package() {
  banner "Installing BindPlane Agent"
  increase_indent

  # if the user didn't specify a local file then download the package
  if [ -z "$package_path" ]; then
    proxy_args=""
    if [ -n "$proxy" ]; then
      proxy_args="-x $proxy"
      if [ -n "$proxy_user" ]; then
        proxy_args="$proxy_args -U $proxy_user:$proxy_password"
      fi
    fi

    if [ -n "$proxy" ]; then
      info "Downloading package using proxy..."
    fi

    info "Downloading package..."
    eval curl -L "$proxy_args" "$collector_download_url" -o "$out_file_path" --progress-bar --fail || error_exit "$LINENO" "Failed to download package"
    succeeded
  fi

  info "Installing package..."
  # if target install directory doesn't exist and we're using dpkg ensure a clean state
  # by checking for the package and running purge if it exists.
  if [ ! -d "$INSTALL_DIR" ] && [ "$package_type" = "deb" ]; then
    dpkg -s "observiq-otel-collector" >/dev/null 2>&1 && dpkg --purge "observiq-otel-collector" >/dev/null 2>&1
  fi

  unpack_package || error_exit "$LINENO" "Failed to extract package"
  succeeded

  # If an endpoint was specified, we need to write the manager.yaml
  if [ -n "$OPAMP_ENDPOINT" ]; then
    info "Creating supervisor config..."
    create_supervisor_config "$SUPERVISOR_YML_PATH"
    succeeded
  fi

  if [ "$SVC_PRE" = "systemctl" ]; then
    if [ "$(systemctl is-enabled observiq-otel-collector)" = "enabled" ]; then
      # The unit is already enabled; It may be running, too, if this was an upgrade.
      # We'll want to restart, which will start it if it wasn't running already,
      # and restart in the case that this was an upgrade on a running agent.
      info "Restarting service..."
      systemctl restart observiq-otel-collector >/dev/null 2>&1 || error_exit "$LINENO" "Failed to restart service"
      succeeded
    else
      info "Enabling service..."
      systemctl enable --now observiq-otel-collector >/dev/null 2>&1 || error_exit "$LINENO" "Failed to enable service"
      succeeded
    fi
  else
    case "$(service observiq-otel-collector status)" in
    *running*)
      # The service is running.
      # We'll want to restart.
      info "Restarting service..."
      service observiq-otel-collector restart >/dev/null 2>&1 || error_exit "$LINENO" "Failed to restart service"
      succeeded
      ;;
    *)
      info "Enabling and starting service..."
      chkconfig observiq-otel-collector on >/dev/null 2>&1 || error_exit "$LINENO" "Failed to enable service"
      service observiq-otel-collector start >/dev/null 2>&1 || error_exit "$LINENO" "Failed to start service"
      succeeded
      ;;
    esac
  fi

  success "BindPlane Agent installation complete!"
  decrease_indent
}

unpack_package() {
  case "$package_type" in
  deb)
    dpkg --force-confold -i "$out_file_path" >/dev/null || error_exit "$LINENO" "Failed to unpack package"
    ;;
  rpm)
    rpm -U "$out_file_path" >/dev/null || error_exit "$LINENO" "Failed to unpack package"
    ;;
  *)
    error "Unrecognized package type"
    return 1
    ;;
  esac
  return 0
}

# create_supervisor_config creates the supervisor.yml at the specified path, containing opamp information.
create_supervisor_config() {
  supervisor_yml_path="$1"

  # Return if we're not doing a clean install
  if [ "$clean_install" = "false" ]; then
    return
  fi

  # Note here: We create the file and change permissions of the file here BEFORE writing info to it.
  # We do this because the file contains the secret key.
  # We do not want the file readable by anyone other than root/obseriq-otel-collector.
  command printf '' >>"$supervisor_yml_path"
  chown observiq-otel-collector:observiq-otel-collector "$supervisor_yml_path"
  chmod 0600 "$supervisor_yml_path"

  command printf 'server:\n' >"$supervisor_yml_path"
  command printf '  endpoint: "%s"\n' "$OPAMP_ENDPOINT" >>"$supervisor_yml_path"
  command printf '  headers:\n' >>"$supervisor_yml_path"
  [ -n "$OPAMP_SECRET_KEY" ] && command printf '    Authorization: "Secret-Key %s"\n' "$OPAMP_SECRET_KEY" >>"$supervisor_yml_path"
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
}

# This will display the results of an installation
display_results() {
  banner 'Information'
  increase_indent
  info "Agent Home:         $(fg_cyan "$INSTALL_DIR")$(reset)"
  info "Agent Config:       $(fg_cyan "$INSTALL_DIR/supervisor_storage/effective.yaml")$(reset)"
  info "Agent Logs Command:       $(fg_cyan "sudo tail -F $INSTALL_DIR/supervisor_storage/agent.log")$(reset)"
  if [ "$SVC_PRE" = "systemctl" ]; then
    info "Supervisor Start Command:      $(fg_cyan "sudo systemctl start observiq-otel-collector")$(reset)"
    info "Supervisor Stop Command:       $(fg_cyan "sudo systemctl stop observiq-otel-collector")$(reset)"
    info "Status Command:     $(fg_cyan "sudo systemctl status observiq-otel-collector")$(reset)"
  else
    info "Supervisor Start Command:      $(fg_cyan "sudo service observiq-otel-collector start")$(reset)"
    info "Supervisor Stop Command:       $(fg_cyan "sudo service observiq-otel-collector stop")$(reset)"
    info "Status Command:     $(fg_cyan "sudo service observiq-otel-collector status")$(reset)"
  fi
  decrease_indent

  banner 'Support'
  increase_indent
  info "For more information on configuring the agent, see the docs:"
  increase_indent
  info "$(fg_cyan "https://github.com/observIQ/bindplane-otel-collector/tree/main#bindplane-agent")$(reset)"
  decrease_indent
  info "If you have any other questions please contact us at $(fg_cyan support@observiq.com)$(reset)"
  increase_indent
  decrease_indent
  decrease_indent

  banner "$(fg_green Installation Complete!)"
  return 0
}

uninstall_package() {
  case "$package_type" in
  deb)
    dpkg -r "observiq-otel-collector" >/dev/null 2>&1
    ;;
  rpm)
    rpm -e "observiq-otel-collector" >/dev/null 2>&1
    ;;
  *)
    error "Unrecognized package type"
    return 1
    ;;
  esac
  return 0
}

uninstall() {
  set_package_type
  banner "Uninstalling BindPlane Agent"
  increase_indent

  info "Checking permissions..."
  root_check
  succeeded

  if [ "$SVC_PRE" = "systemctl" ]; then
    info "Stopping service..."
    systemctl stop observiq-otel-collector >/dev/null || error_exit "$LINENO" "Failed to stop service"
    succeeded

    info "Disabling service..."
    systemctl disable observiq-otel-collector >/dev/null 2>&1 || error_exit "$LINENO" "Failed to disable service"
    succeeded
  else
    info "Stopping service..."
    service observiq-otel-collector stop
    succeeded

    info "Disabling service..."
    chkconfig observiq-otel-collector on
    # rm -f /etc/init.d/observiq-otel-collector
    succeeded
  fi

  info "Removing any existing manager.yaml..."
  rm -f "$MANAGEMENT_YML_PATH"
  succeeded

  info "Removing package..."
  uninstall_package || error_exit "$LINENO" "Failed to remove package"
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
      -v | --version)
        version=$2
        shift 2
        ;;
      -l | --url)
        url=$2
        shift 2
        ;;
      -f | --file)
        package_path=$2
        shift 2
        ;;
      -x | --proxy)
        proxy=$2
        shift 2
        ;;
      -U | --proxy-user)
        proxy_user=$2
        shift 2
        ;;
      -P | --proxy-password)
        proxy_password=$2
        shift 2
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
      -h | --help)
        usage
        exit 0
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
