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

# Allow configurable runtime user/group (used for permissions and manager.yaml)
: "${BDOT_USER:=bdot}"
: "${BDOT_GROUP:=bdot}"

# Agent Constants
PACKAGE_NAME="bindplane-otel-collector"
DOWNLOAD_BASE="https://bdot.bindplane.com"

# Determine if we need service or systemctl for prereqs
if command -v systemctl >/dev/null 2>&1; then
  SVC_PRE=systemctl
elif command -v service >/dev/null 2>&1; then
  SVC_PRE=service
elif command -v mkssys > /dev/null 2>&1; then
  SVC_PRE=mkssys
fi

# Script Constants
COLLECTOR_USER="${BDOT_USER}"
COLLECTOR_GROUP="${BDOT_GROUP}"
COLLECTOR_USER_LEGACY="bindplane-otel-collector"
TMP_DIR=${TMPDIR:-"/tmp"} # Allow this to be overriden by cannonical TMPDIR env var
INSTALL_DIR="/opt/bindplane-otel-collector"
SUPERVISOR_YML_PATH="$INSTALL_DIR/supervisor.yaml"
PREREQS="curl printf $SVC_PRE sed uname cut tr tar sudo"
SCRIPT_NAME="$0"
INDENT_WIDTH='  '
indent=""
non_interactive=false
error_mode=false
skip_gpg_check=false

# Default Supervisor Config Hash
DEFAULT_SUPERVISOR_CFG_HASH="ac4e6001f1b19d371bba6a2797ba0a55d7ca73151ba6908040598ca275c0efca"

# Require bash for non-AIX to create a standardized environment
# Also set up the supervisor timeout values (AIX needs higher values due to slow I/O)
if [ "$(uname -s)" = "AIX" ]; then
  config_apply_timeout="150"
  bootstrap_timeout="120"
else
  PREREQS="bash $PREREQS"
  config_apply_timeout="30"
  bootstrap_timeout="5"
fi

os="unknown"
os_arch="unknown"
# package_out_file_path is the full path to the downloaded package (e.g. "/tmp/bindplane-otel-collector_linux_amd64.deb")
package_out_file_path="unknown"

# gpg_tar_out_file_path is the full path to the downloaded GPG tar.gz file (e.g. "/tmp/bdot-gpg-keys.tar.gz")
gpg_tar_out_file_path="unknown"

offline_installation=false

# RPM_GPG_KEYS_TO_REMOVE is a list of GPG keys to remove from the RPM package. This is used for revoked keys. Deb packages are handled differently.
# The entries in this should be formatted similarly to gpg-pubkey-<version>-<release>
# The entries can be found by importing the revoked public key and then exploring the RPM database for the key.
# rpm --import <revoked_public_key>.asc
# rpm -q gpg-pubkey, it's one of these
# rpm -q gpg-pubkey --info, go find the BDOT public key and use the version and release numbers from there.
RPM_GPG_KEYS_TO_REMOVE=[]

# Colors
num_colors=$(tput colors 2>/dev/null)
if [ "$non_interactive" = "false" ] && [ "$(uname -s)" != AIX ] && test -n "$num_colors" && test "$num_colors" -ge 8; then
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

if [ -z "$reset" ]; then
  sed_ignore=''
else
  sed_ignore="/^[$reset]+$/!"
fi

# Helper Functions
printf() {
  if [ "$non_interactive" = "false" ] || [ "$error_mode" = "true" ]; then
    if [ "$(uname -s)" != "AIX" ] && command -v sed >/dev/null; then
      command printf -- "$@" | sed -n "$sed_ignore s/^/$indent/g"  # Ignore sole reset characters if defined
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
      Defines the version of the BDOT.
      If not provided, this will default to the latest version.
      Alternatively the COLLECTOR_VERSION environment variable can be
      set to configure the agent version.
      Example: '-v 1.2.12' will download 1.2.12.

  $(fg_yellow '-r, --uninstall')
      Stops the agent services and uninstalls the agent.

  $(fg_yellow '-l, --url')
      Defines the URL that the components will be downloaded from.
      If not provided, this will default to BDOT\'s GitHub releases.
      Example: '-l http://my.domain.org/bindplane-otel-collector' will download from there.

  $(fg_yellow '-gl, --gpg-tar-url')
      Defines the URL that the GPG tar file will be downloaded from.
      If not provided, this will default to Bindplane Agent\'s GitHub releases.
      Example: '-gl http://my.domain.org/bdot-gpg-keys.tar.gz' will download from there.

  $(fg_yellow '-b, --base-url')
      Defines the base of the download URL used in conjunction with the version to download the package and GPG tar file.
      '{base_url}/v{version}/{PACKAGE_NAME}_v{version}_{os}_{os_arch}.{package_type}'
      and
      '{base_url}/v{version}/gpg-keys.tar.gz'
      If not provided, this will default to '$DOWNLOAD_BASE'.
      Example: '-b http://my.domain.org/bindplane-otel-collector/binaries' will be used as the base of the download URL.

  $(fg_yellow '-f, --file')
      Install Agent from a local file instead of downloading from a URL.
      Example: '-f /path/to/bindplane-otel-collector_v1.2.12_linux_amd64.deb' will install from the local file.
      Required if '--gpg-tar-file' is specified.

  $(fg_yellow '-gf, --gpg-tar-file')
      Verify the Agent from a local GPG tar file instead of downloading from a URL.
      Example: '-gf /path/to/bdot-gpg-keys.tar.gz' will verify from the local file.
      Required if '--file' is specified.

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
    Check access to the Bindplane server URL.

    This parameter will have the script check access to Bindplane based on the provided '--endpoint'

  $(fg_yellow '--no-gpg-check')
      Skips GPG signature verification of the package. When using this flag, the
      package signature will not be verified. This should only be used in trusted
      or offline environments where the package authenticity has been verified
      through other means.
      
      This option is incompatible with '--gpg-tar-file' and will cause the script
      to exit with an error if both are specified.

  $(fg_yellow '-q, --quiet')
    Use quiet (non-interactive) mode to run the script in headless environments.
    
    Note: If a GPG signature verification failure occurs during installation and
    '--no-gpg-check' was not specified, the script will exit immediately without
    prompting the user to continue. For interactive handling of verification
    failures, do not use the '--quiet' flag.

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

# This will validate that the version is at least v1.82.0
validate_version()
{
  if [ -z "$version" ]; then
    return 0  # No version specified, let the script handle it
  fi

  info "Validating version compatibility..."

  # Remove 'v' prefix if present
  version_clean=$(echo "$version" | sed 's/^v//')

  # Extract major and minor version numbers
  major=$(echo "$version_clean" | cut -d'.' -f1)
  minor=$(echo "$version_clean" | cut -d'.' -f2)

  # Check if major version is 2 and minor version is >= 0
  if [ "$major" = "2" ] && [ "$minor" -ge 0 ] 2>/dev/null; then
    succeeded
    return 0
  else
    failed
    error_exit "$LINENO" "Version $version is not supported. This script supports collector v2.0.0 or newer. Please use the script versioned with your desired collector version."
  fi
}

# This will set all installation variables
# at the beginning of the script.
setup_installation() {
  banner "Configuring Installation Variables"
  increase_indent

  # Installation variables
  set_os_arch
  set_package_type

  # if offline_installation is false then download the package
  if [ "$offline_installation" = "false" ]; then
    set_download_urls
    set_proxy
    set_file_names
  else
    package_out_file_path="$package_path"
    gpg_tar_out_file_path="$gpg_tar_path"
  fi

  set_opamp_endpoint
  set_opamp_labels
  set_opamp_secret_key

  ask_clean_install

  success "Configuration complete!"
  decrease_indent
}

set_file_names() {
  if [ -z "$version" ]; then
    package_file_name="${PACKAGE_NAME}_${os}_${os_arch}.${package_type}"
  else
    package_file_name="${PACKAGE_NAME}_v${version}_${os}_${os_arch}.${package_type}"
  fi
  package_out_file_path="$TMP_DIR/$package_file_name"

  gpg_tar_out_file_path="$TMP_DIR/bdot-gpg-keys.tar.gz"
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


set_os_arch()
{
  case "$os_arch" in 
    # arm64 strings. These are from https://stackoverflow.com/questions/45125516/possible-values-for-uname-m
    aarch64|arm64|aarch64_be|armv8b|armv8l)
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
    powerpc)
      if [ "$(uname -s)" = "AIX" ] && command -v bootinfo > /dev/null && [ "$(bootinfo -y)" = "64" ]; then
        os_arch="ppc64"
      elif command -v bootinfo > /dev/null; then
        error_exit "$LINENO" "Command 'bootinfo' not found, OS likely isn't AIX, but we expect it to be"
      else
        error_exit "$LINENO" "uname returned arch of 'powerpc', but OS is either not AIX or not 64 bit. 'uname -s': $(uname -s), 'bootinfo -y': $(bootinfo -y)"
      fi
      ;;
    # armv6/32bit. These are what raspberry pi can return, which is the main reason we support 32-bit arm
    arm|armv6l|armv7l)
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
  if [ -n "$package_path" ] && [ "$(uname -s)" != "AIX" ]; then
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
    if command -v dpkg > /dev/null 2>&1; then
        package_type="deb"
    elif command -v mkssys > /dev/null 2>&1; then
        package_type="mkssys"
    elif command -v rpm > /dev/null 2>&1; then
        package_type="rpm"
    else
        error_exit "$LINENO" "Could not find mkssys, dpkg or rpm on the system"
    fi
  fi

}

# This will set the urls to use when downloading the agent and its plugins.
# These urls are constructed based on the --version flag or COLLECTOR_VERSION env variable.
# If not specified, the version defaults to whatever the latest release on github is.
set_download_urls()
{
  if [ -z "$url" ]; then
    if [ -z "$base_url" ]; then
      base_url=$DOWNLOAD_BASE
    fi

    collector_download_url="$base_url/v$version/${PACKAGE_NAME}_v${version}_${os}_${os_arch}.${package_type}"
  else
    collector_download_url="$url"
  fi

  if [ -z "$gpg_tar_url" ]; then
    if [ -z "$base_url" ] ; then
      base_url=$DOWNLOAD_BASE
    fi

    gpg_tar_download_url="$base_url/v$version/gpg-keys.tar.gz"
  else
    gpg_tar_download_url="$gpg_tar_url"
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
    # Check for default config file hash
    cfg_file_hash=$(sha256sum "$SUPERVISOR_YML_PATH" | awk '{print $1}')
    if [ "$cfg_file_hash" = "$DEFAULT_SUPERVISOR_CFG_HASH" ]; then
      # config matches default config, mark clean_install as true
      clean_install="true"
    else
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
    fi
  else
    warn "Previous supervisor config not found, doing clean install"
    clean_install="true"
  fi
}

# Test connection to Bindplane if it was specified
connection_check() {
  if [ -n "$check_bp_url" ]; then
    if [ -n "$opamp_endpoint" ]; then
      HTTP_ENDPOINT="$(echo "${opamp_endpoint}" | sed -z 's#^ws#http#' | sed -z 's#/v1/opamp$##')"
      info "Testing connection to Bindplane: $fg_magenta$HTTP_ENDPOINT$reset..."

      if curl --max-time 20 -s "${HTTP_ENDPOINT}" >/dev/null; then
        succeeded
      else
        failed
        warn "Connection to Bindplane has failed."
        increase_indent
        printf "%sDo you wish to continue installation?%s  " "$fg_yellow" "$reset"
        prompt "n"
        decrease_indent
        read -r input
        printf "\\n"
        if [ "$input" = "y" ] || [ "$input" = "Y" ]; then
          info "Continuing installation."
        else
          error_exit "$LINENO" "Aborting due to user input after connectivity failure between this system and the Bindplane server."
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
  user_check
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
interactive_check()
{
  # Incompatible with --no-gpg-check and --gpg-tar-file
  if [ "$skip_gpg_check" = "true" ] && [ -n "$gpg_tar_path" ]; then
    failed
    error_exit "$LINENO" "--no-gpg-check is incompatible with '--gpg-tar-file'. These options cannot be used together."
  fi

  # Incompatible with proxies unless both username and password are passed
  if [ "$non_interactive" = "true" ] && [ -n "$proxy_password" ]; then
    failed
    error_exit "$LINENO" "The proxy password must be set via the command line argument -P, if called non-interactively."
  fi

  # Incompatible with checking the BP url since it can be interactive on failed connection
  if [ "$non_interactive" = "true" ] && [ "$check_bp_url" = "true" ]; then
    failed
    error_exit "$LINENO" "Checking the Bindplane server URL is not compatible with quiet (non-interactive) mode."
  fi
}

offline_check()
{
  # --file without --gpg-tar-file is allowed when --no-gpg-check is set
  if [ -n "$package_path" ] && [ -z "$gpg_tar_path" ] && [ "$skip_gpg_check" != "true" ]; then
    error_exit "$LINENO" "Both --file and --gpg-tar-file must be specified together, or use --no-gpg-check to skip signature verification."
  fi

  if [ -z "$package_path" ] && [ -n "$gpg_tar_path" ]; then
    error_exit "$LINENO" "--gpg-tar-file requires --file to be specified."
  fi

  if [ -n "$package_path" ]; then
    offline_installation=true
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
    AIX)
      succeeded
      ;;
    *)
      failed
      error_exit "$LINENO" "The operating system $(fg_yellow "$os_type") is not supported by this script."
      ;;
  esac

  # Create lowercase os variable
  os=$(echo "$os_type" | tr '[:upper:]' '[:lower:]')
}

# This will check if the system architecture is supported.
os_arch_check() {
  info "Checking for valid operating system architecture..."
  if [ "$(uname -s)" = "AIX" ]; then
    os_arch=$(uname -p)
  else
    os_arch=$(uname -m)
  fi
  case "$os_arch" in 
    x86_64|aarch64|powerpc|ppc64|ppc64le|arm64|aarch64_be|armv8b|armv8l|arm|armv6l|armv7l)
      succeeded
      ;;
    *)
      failed
      error_exit "$LINENO" "The operating system architecture $(fg_yellow "$os_arch") is not supported by this script."
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

# This will check if the required collector user exists when BDOT_SKIP_RUNTIME_USER_CREATION is set to true.
user_check()
{
  if [ "$BDOT_SKIP_RUNTIME_USER_CREATION" != "true" ]; then
    succeeded
    return 0
  fi

  info "BDOT_SKIP_RUNTIME_USER_CREATION is set to true, checking for existing collector users..."

  user_exists=false

  if id "$COLLECTOR_USER" >/dev/null 2>&1; then
    user_exists=true
    info "Found collector user: $COLLECTOR_USER"
  fi

  if id "$COLLECTOR_USER_LEGACY" >/dev/null 2>&1; then
    user_exists=true
    info "Found legacy collector user: $COLLECTOR_USER_LEGACY"
  fi

  if [ "$user_exists" = "false" ]; then
    failed
    error_exit "$LINENO" "BDOT_SKIP_RUNTIME_USER_CREATION is set to true, but neither collector user ($COLLECTOR_USER) nor legacy collector user ($COLLECTOR_USER_LEGACY) exists on the system."
  fi

  succeeded
}

# This will check to ensure either dpkg or rpm is installed on the system
package_type_check()
{
  info "Checking for package manager..."
  if command -v dpkg > /dev/null 2>&1; then
      succeeded
  # Check ALL of the AIX commands needed
  elif command -v mkssys > /dev/null 2>&1 \
    && command -v mkgroup > /dev/null 2>&1 \
    && command -v useradd > /dev/null 2>&1 \
    && command -v startsrc > /dev/null 2>&1 \
    && command -v stopsrc > /dev/null 2>&1 \
    && command -v lssrc > /dev/null 2>&1 \
    && command -v mkitab > /dev/null 2>&1 \
    && command -v rmitab > /dev/null 2>&1 \
    && command -v lsitab > /dev/null 2>&1; then
      succeeded
  elif command -v rpm > /dev/null 2>&1; then
      succeeded
  else
      failed
      if [ "$(uname -s)" = "AIX" ]; then
        error_exit "$LINENO" "Could not find AIX installation tools on the system"
      else
        error_exit "$LINENO" "Could not find dpkg or rpm on the system"
      fi
  fi
}

# latest_version gets the tag of the latest release, without the v prefix.
latest_version()
{
  curl -s https://bdot.bindplane.com/latest
}

# This will install the package by downloading the archived agent,
# extracting the binaries, and then removing the archive.
install_package() {
  banner "Installing BDOT"
  increase_indent

  # if the user didn't specify a local file then download the package
  if [ "$offline_installation" = "false" ]; then
    proxy_args=""
    if [ -n "$proxy" ]; then
      proxy_args="-x $proxy"
      if [ -n "$proxy_user" ]; then
        proxy_args="$proxy_args -U $proxy_user:$proxy_password"
      fi
    fi

    if [ -n "$proxy" ]; then
      info "Downloading package from $collector_download_url using proxy..."
    else 
      info "Downloading package from $collector_download_url..."
    fi

    eval curl -L "$proxy_args" "$collector_download_url" -o "$package_out_file_path" --progress-bar --fail || error_exit "$LINENO" "Failed to download package"

    if [ -n "$proxy" ]; then
      info "Downloading GPG key tar file from $gpg_tar_download_url using proxy..."
    else 
      info "Downloading GPG key tar file from $gpg_tar_download_url..."
    fi

    eval curl -L "$proxy_args" "$gpg_tar_download_url" -o "$gpg_tar_out_file_path" --progress-bar --fail || error_exit "$LINENO" "Failed to download GPG tar file"
    succeeded
  fi

  info "Installing package..."
  # if target install directory doesn't exist and we're using dpkg ensure a clean state
  # by checking for the package and running purge if it exists.
  if [ ! -d "/opt/bindplane-otel-collector" ] && [ "$package_type" = "deb" ]; then
    info "Running dpkg --purge to ensure a clean state"
    dpkg -s "bindplane-otel-collector" > /dev/null 2>&1 && dpkg --purge "bindplane-otel-collector" > /dev/null 2>&1
  fi

  # Verify the package signature, with optional user override on failure
  # Capture GPG verification output to display failure details
  # Temporarily disable set -e to allow capture of failing command output
  set +e
  gpg_verify_output=$(verify_package 2>&1)
  gpg_verify_exit_code=$?
  set -e

  if [ -n "$gpg_verify_output" ]; then
    printf "%s\n" "$gpg_verify_output"
  fi
  
  if [ $gpg_verify_exit_code -ne 0 ]; then
    if [ "$non_interactive" = "true" ]; then
      # In quiet mode, fail immediately on GPG verification failure
      if [ -n "$gpg_verify_output" ]; then
        increase_indent
        printf "%s\n" "$gpg_verify_output"
        decrease_indent
      fi
      error_exit "$LINENO" "Failed to verify package signature. Use '--no-gpg-check' to skip verification."
    else
      # In interactive mode, show verification output, prompt the user, and explain failure
      if [ -n "$gpg_verify_output" ]; then
        increase_indent
        printf "%s\n" "$gpg_verify_output"
        decrease_indent
      fi
      
      increase_indent
      printf "\\n${indent}The package signature could not be verified. This may indicate:\n"
      printf "${indent}  - The GPG keys are not properly installed or accessible\n"
      printf "${indent}  - The package has been tampered with\n"
      printf "${indent}  - The signing key has expired or been revoked\n"
      printf "${indent}  - Network issues prevented GPG key retrieval\n"
      printf "\\n${indent}$(fg_yellow 'Continuing without signature verification is NOT RECOMMENDED unless you have independently verified the package authenticity.')\\n\\n"
      decrease_indent
      
      command printf "${indent}Do you wish to continue installation without GPG verification? "
      prompt "n"
      read -r gpg_override_input
      printf "\\n"
      
      if [ "$gpg_override_input" != "y" ] && [ "$gpg_override_input" != "Y" ]; then
        if [ -n "$gpg_verify_output" ]; then
          increase_indent
          error "Verification failed due to:"
          printf "%s\n" "$gpg_verify_output"
          decrease_indent
        fi
        error_exit "$LINENO" "Installation aborted due to GPG verification failure."
      fi
      
      warn "Continuing installation without GPG verification. Ensure package authenticity has been verified through other means."
    fi
  fi
  install_package_file || error_exit "$LINENO" "Failed to extract package"
  succeeded

  create_supervisor_config "$SUPERVISOR_YML_PATH"

  if [ "$SVC_PRE" = "systemctl" ]; then
    if [ "$(systemctl is-enabled bindplane-otel-collector)" = "enabled" ]; then
      # The unit is already enabled; It may be running, too, if this was an upgrade.
      # We'll want to restart, which will start it if it wasn't running already,
      # and restart in the case that this was an upgrade on a running agent.
      info "Restarting service..."
      systemctl restart bindplane-otel-collector >/dev/null 2>&1 || error_exit "$LINENO" "Failed to restart service"
      succeeded
    else
      info "Enabling service..."
      systemctl enable --now bindplane-otel-collector >/dev/null 2>&1 || error_exit "$LINENO" "Failed to enable service"
      succeeded
    fi
  elif [ "$SVC_PRE" = "service" ]; then
    case "$(service bindplane-otel-collector status)" in
    *running*)
      # The service is running.
      # We'll want to restart.
      info "Restarting service..."
      service bindplane-otel-collector restart >/dev/null 2>&1 || error_exit "$LINENO" "Failed to restart service"
      succeeded
      ;;
    *)
      info "Enabling and starting service..."
      chkconfig bindplane-otel-collector on >/dev/null 2>&1 || error_exit "$LINENO" "Failed to enable service"
      service bindplane-otel-collector start >/dev/null 2>&1 || error_exit "$LINENO" "Failed to start service"
      succeeded
      ;;
    esac
  elif [ "$SVC_PRE" = "mkssys" ]; then
    case "$(lssrc -g bpcollector | grep collector)" in
      *active*)
        # The service is running.
        # We'll want to restart.
        info "Restarting service..."
        # AIX does not support service "restart", so stop and start instead
        stopsrc -g bpcollector
        # Start the service with the proper environment variables
        startsrc -g bpcollector -e "$(cat /etc/bindplane-otel-collector.aix.env)"
        succeeded
        ;;
      *inoperative*)
        info "Starting service..."
        # Start the service with the proper environment variables
        startsrc -g bpcollector-e "$(cat /etc/bindplane-otel-collector.aix.env)"
        succeeded
        ;;
      *)
        info "Creating, enabling and starting service..."
        # Add the service, removing it if it already exists in order
        # to make sure we have the most recent version
        if lssrc -g bpcollector > /dev/null 2>&1; then
          rmssys -g bpcollector
        else
          mkssys -s bindplane-otel-collector  -G bpcollector -p /opt/bindplane-otel-collector/opampsupervisor -u "$(id -u root)" -S -n15 -f9 -a '--config /opt/bindplane-otel-collector/supervisor.yaml'
        fi

        # Install the service to start on boot
        # Removing it if it exists, in order to have the most recent version
        if lsitab bpcollector > /dev/null 2>&1; then
          rmitab bpcollector
        else
          # shellcheck disable=SC2016
          mkitab 'bpcollector:23456789:once:startsrc -g bpcollector -e "$(cat /etc/bindplane-otel-collector.aix.env)"'
        fi

        # Start the service with the proper environment variables
        startsrc -g bpcollector -e "$(cat /etc/bindplane-otel-collector.aix.env)"

        succeeded
        ;;
    esac
  else
    # This is an error state that should never be reached
    error_exit "$LINENO" "Found an invalid SVC_PRE value in install_package()"
  fi

  success "BDOT installation complete!"
  decrease_indent
}

verify_package() {
  # If GPG check is skipped, return success immediately
  if [ "$skip_gpg_check" = "true" ]; then
    warn "GPG signature verification is being bypassed with the '--no-gpg-check' flag."
    warn "This disables a critical security check and should only be used if your organization policies permit it."
    return 0
  fi

  [ -d "$TMP_DIR/gpg" ] && rm -rf "$TMP_DIR/gpg"
  mkdir -p "$TMP_DIR/gpg"

  if ! tar -xzf "$gpg_tar_out_file_path" -C "$TMP_DIR/gpg" > /dev/null 2>&1; then
    error "Failed to extract GPG key tar file"
    return 1
  fi

  case "$package_type" in
    deb)
      if ! verify_package_deb; then
        return 1
      fi
      ;;
    rpm)
      if ! verify_package_rpm; then
        return 1
      fi
      ;;
    *)
      error "Unrecognized package type"
      return 1
      ;;
  esac

  return 0
}

verify_package_deb() {
  if ! command -v gpg > /dev/null 2>&1; then
    info "gpg is not installed, skipping signature verification"
    return 0
  fi

  if ! command -v ar > /dev/null 2>&1; then
    info "ar is not installed, skipping signature verification"
    return 0
  fi

  if ! GNUPGHOME="$TMP_DIR/gpg" gpg --import "$TMP_DIR/gpg/bdot-public-gpg-key.asc" > /dev/null 2>&1; then
    error "Failed to import public key"
    return 1
  fi
  # if there are any revocation keys, import them
  if [ -n "$(ls -A "$TMP_DIR/gpg/deb-revocations/" 2>/dev/null)" ]; then
    for key in "$TMP_DIR/gpg/deb-revocations/"*; do
      if ! GNUPGHOME="$TMP_DIR/gpg" gpg --import "$key" > /dev/null 2>&1; then
        error "Failed to import revocation key"
        return 1
      fi
    done
  fi

  if ! ar x "$package_out_file_path" "_gpgorigin" > /dev/null 2>&1; then
    error "Failed to extract package signature"
    return 1
  fi

  if ! mv "_gpgorigin" "$TMP_DIR/gpg/_gpgorigin"; then
    error "Failed to move package signature to temporary directory"
    return 1
  fi

  set +e
  # Run pipeline, capture both output and exit code
  OUTPUT=$(ar p "$package_out_file_path" debian-binary control.tar.gz data.tar.gz | \
          GNUPGHOME="$TMP_DIR/gpg" gpg --verify "$TMP_DIR/gpg/_gpgorigin" - 2>&1)
  EXIT_CODE=$?
  set -e

  # Fail if gpg failed
  if [ $EXIT_CODE -ne 0 ]; then
    error "Package signature is invalid"
    return 1
  fi

  # Fail if key is revoked
  if echo "$OUTPUT" | grep -q "key has been revoked"; then
    error "Package signature is from a revoked key"
    return 1
  fi

  if echo "$OUTPUT" | grep -q "key has expired"; then
    error "Package signature is from an expired key"
    return 1
  fi

  success "Package signature is valid, not revoked, and subkey is not expired"
  return 0
}

verify_package_rpm() {
  set +e
  # Capture stderr from rpm --import
  IMPORT_OUTPUT=$(rpm --import "$TMP_DIR/gpg/bdot-public-gpg-key.asc" 2>&1)
  IMPORT_EXIT_CODE=$?
  set -e

  # Fail if rpm --import itself fails
  if [ $IMPORT_EXIT_CODE -ne 0 ]; then
      error "Failed to import public key"
      return 1
  fi

  # Extract the signing key ID from checksig (reliable on EL7+)
  SIGNING_KEYID=$(rpm --checksig --verbose "$package_out_file_path" 2>&1 \
    | sed -nE 's/.*[Kk]ey ID ([0-9A-Fa-f]+):.*/\1/p' \
    | head -n1)

  if [ -z "$SIGNING_KEYID" ]; then
    error "Could not determine RPM signing key ID"
    return 1
  fi

  # Normalize key ID to lowercase (rpm stores gpg-pubkey in lowercase)
  SIGNING_KEYID=$(echo "$SIGNING_KEYID" | tr '[:upper:]' '[:lower:]')

  # Remove revoked keys (your existing logic)
  if [ ${#RPM_GPG_KEYS_TO_REMOVE[@]} -gt 0 ]; then
    for key in "${RPM_GPG_KEYS_TO_REMOVE[@]}"; do
      if rpm -q "$key" > /dev/null 2>&1; then
        if ! rpm -e "$key" > /dev/null 2>&1; then
          error "Failed to remove revocation key"
          return 1
        fi
      fi
    done
  fi

  if ! rpm -qa 'gpg-pubkey*' \
  | xargs -n1 rpm -qi \
  | gpg --quiet --with-colons --show-keys \
  | awk -F: '$1=="sub" {print tolower(substr($5, length($5)-7))}' \
  | grep -qx "$SIGNING_KEYID"; then
      error "RPM signed by subkey $SIGNING_KEYID which is not present in any installed GPG key"
      return 1
  fi

  # Verify the signature
  set +e
  CHECKSIG_OUTPUT=$(rpm --checksig --verbose "$package_out_file_path" 2>&1)
  CHECKSIG_EXIT_CODE=$?
  set -e

  # Reject hard failures first
  if echo "$CHECKSIG_OUTPUT" | grep -q "BAD"; then
    error "RPM signature is BAD"
    return 1
  fi

  if echo "$CHECKSIG_OUTPUT" | grep -qi "EXPIRED"; then
    error "RPM signature uses an expired key"
    return 1
  fi

  # On Oracle Linux 7, rpm --checksig may show NOKEY/MISSING KEYS even when valid
  if echo "$CHECKSIG_OUTPUT" | grep -qE "NOKEY|MISSING KEYS"; then
    info "Ignoring legacy MD5/PGP warning on Oracle Linux (key verified)"
  elif [ $CHECKSIG_EXIT_CODE -ne 0 ]; then
    error "Failed to verify package signature"
    return 1
  fi

  success "Package signature is valid, not revoked, and subkey is not expired"
  return 0
}

# Install on AIX manually from tar.gz file
install_aix()
{
  # Create the bindplane-otel-collector user and group. Group must be first.
  mkgroup "$COLLECTOR_USER" > /dev/null 2>&1
  useradd -d /opt/bindplane-otel-collector -g "$COLLECTOR_USER" -s "$(which bash)" "$COLLECTOR_USER" > /dev/null 2>&1

  # Create the install & storage directories
  mkdir -p /opt/bindplane-otel-collector/storage > /dev/null 2>&1

  # Extract 
  gunzip -c "$package_out_file_path" | tar -Uxvf - -C /opt/bindplane-otel-collector > /dev/null 2>&1

  # Move files to appropriate locations
  mv /opt/bindplane-otel-collector/install/bindplane-otel-collector.aix.env /etc/ > /dev/null 2>&1

  # Set ownership
  chown -R "$COLLECTOR_USER":"$COLLECTOR_USER" /opt/bindplane-otel-collector > /dev/null 2>&1
  chown "$COLLECTOR_USER":"$COLLECTOR_USER" /etc/bindplane-otel-collector.aix.env > /dev/null 2>&1
}

install_package_file()
{
  case "$package_type" in
    deb)
      dpkg --force-confold -i "$package_out_file_path" > /dev/null || error_exit "$LINENO" "Failed to unpack package"
      ;;
    mkssys)
      install_aix
      ;;
    rpm)
      rpm -U "$package_out_file_path" > /dev/null || error_exit "$LINENO" "Failed to unpack package"
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

  info "Creating supervisor config..."

  if [ -z "$OPAMP_ENDPOINT" ]; then
    OPAMP_ENDPOINT="ws://localhost:3001/v1/opamp"
    increase_indent
    info "No OpAMP endpoint specified, starting agent using 'ws://localhost:3001/v1/opamp' as endpoint."
    decrease_indent
  fi

  # Note here: We create the file and change permissions of the file here BEFORE writing info to it.
  # We do this because the file contains the secret key.
  # We do not want the file readable by anyone other than root/configured user.
  command printf '' >>"$supervisor_yml_path"
  chown ${BDOT_USER}:${BDOT_GROUP} "$supervisor_yml_path"
  chmod 0600 "$supervisor_yml_path"

  command printf 'server:\n' >"$supervisor_yml_path"
  command printf '  endpoint: "%s"\n' "$OPAMP_ENDPOINT" >>"$supervisor_yml_path"

  if [ -n "$OPAMP_SECRET_KEY" ]; then
    command printf '  headers:\n' >>"$supervisor_yml_path"
    command printf '    Authorization: "Secret-Key %s"\n' "$OPAMP_SECRET_KEY" >>"$supervisor_yml_path"
    command printf '    User-Agent: "bindplane-otel-collector/%s"\n' "$version" >>"$supervisor_yml_path"
  fi

  command printf '  tls:\n' >>"$supervisor_yml_path"
  command printf '    insecure: true\n' >>"$supervisor_yml_path"
  command printf '    insecure_skip_verify: true\n' >>"$supervisor_yml_path"
  command printf 'capabilities:\n' >>"$supervisor_yml_path"
  command printf '  accepts_remote_config: true\n' >>"$supervisor_yml_path"
  command printf '  reports_remote_config: true\n' >>"$supervisor_yml_path"
  command printf '  reports_available_components: true\n' >>"$supervisor_yml_path"
  command printf 'agent:\n' >>"$supervisor_yml_path"
  command printf '  executable: "%s"\n' "$INSTALL_DIR/bindplane-otel-collector" >>"$supervisor_yml_path"
  command printf '  config_apply_timeout: %ss\n' $config_apply_timeout >>"$supervisor_yml_path"
  command printf '  bootstrap_timeout: %ss\n' $bootstrap_timeout >>"$supervisor_yml_path"
  if [ "$(uname -s)" = "AIX" ]; then
    command printf '  orphan_detection_interval: 120s\n' >>"$supervisor_yml_path"
  fi
  command printf '  args: ["--feature-gates", "service.AllowNoPipelines"]\n' >>"$supervisor_yml_path"

  if [ -n "$OPAMP_LABELS" ]; then
    command printf '  description:\n' >>"$supervisor_yml_path"
    command printf '    non_identifying_attributes:\n' >>"$supervisor_yml_path"
    command printf '      service.labels: "%s"\n' "$OPAMP_LABELS" >>"$supervisor_yml_path"
  fi

  command printf 'storage:\n' >>"$supervisor_yml_path"
  command printf '  directory: "%s"\n' "$INSTALL_DIR/supervisor_storage" >>"$supervisor_yml_path"
  command printf 'telemetry:\n' >>"$supervisor_yml_path"
  command printf '  logs:\n' >>"$supervisor_yml_path"
  command printf '    level: 0\n' >>"$supervisor_yml_path"
  command printf '    output_paths: ["%s"]' "$INSTALL_DIR/supervisor.log" >>"$supervisor_yml_path"
  succeeded
}

# This will display the results of an installation
display_results() {
  banner 'Information'
  increase_indent
  info "Agent Home:          $(fg_cyan "$INSTALL_DIR")$(reset)"
  info "Agent Config:        $(fg_cyan "$INSTALL_DIR/supervisor_storage/effective.yaml")$(reset)"
  info "Agent Logs Command:  $(fg_cyan "sudo tail -F $INSTALL_DIR/supervisor_storage/agent.log")$(reset)"
  if [ "$SVC_PRE" = "systemctl" ]; then
    info "Supervisor Start Command:      $(fg_cyan "sudo systemctl start bindplane-otel-collector")$(reset)"
    info "Supervisor Stop Command:       $(fg_cyan "sudo systemctl stop bindplane-otel-collector")$(reset)"
    info "Supervisor Status Command:     $(fg_cyan "sudo systemctl status bindplane-otel-collector")$(reset)"
  elif [ "$SVC_PRE" = "service" ]; then
    info "Supervisor Start Command:      $(fg_cyan "sudo service bindplane-otel-collector start")$(reset)"
    info "Supervisor Stop Command:       $(fg_cyan "sudo service bindplane-otel-collector stop")$(reset)"
    info "Supervisor Status Command:     $(fg_cyan "sudo service bindplane-otel-collector status")$(reset)"
  elif [ "$SVC_PRE" = "mkssys" ]; then
    info "Supervisor Start Command:      $(fg_cyan "sudo startsrc -s bindplane-otel-collector -e \"\$(cat /etc/bindplane-otel-collector.aix.env)\"")$(reset)"
    info "Supervisor Stop Command:       $(fg_cyan "sudo stopsrc -s bindplane-otel-collector")$(reset)"
    info "Supervisor Status Command:     $(fg_cyan "sudo lssrc -s bindplane-otel-collector")$(reset)"
  fi
  info "Uninstall Command:  $(fg_cyan "sudo sh -c \"\$(curl -fsSlL ${DOWNLOAD_BASE}/v${version}/install_unix.sh)\" install_unix.sh -r")$(reset)"
  decrease_indent

  banner 'Support'
  increase_indent
  info "For more information on configuring the agent, see the docs:"
  increase_indent
  info "$(fg_cyan "https://github.com/observIQ/bindplane-otel-collector/tree/main#bindplane-otel-collector")$(reset)"
  decrease_indent
  info "If you have any other questions please contact us at $(fg_cyan support@observiq.com)$(reset)"
  increase_indent
  decrease_indent
  decrease_indent

    banner "$(fg_green Installation Complete!)"
    return 0
}

uninstall_aix()
{
  # Remove files
  rm -rf /opt/bindplane-otel-collector > /dev/null 2>&1
  rm -f /etc/bindplane-otel-collector.aix.env > /dev/null 2>&1
}

uninstall_package()
{
  case "$package_type" in
    deb)
      dpkg -r "bindplane-otel-collector" > /dev/null 2>&1
      ;;
    mkssys)
      uninstall_aix
      ;;
    rpm)
      rpm -e "bindplane-otel-collector" > /dev/null 2>&1
      ;;
    *)
      error "Unrecognized package type"
      return 1
      ;;
  esac
  return 0
}

uninstall()
{
  bindplane_banner

  set_package_type
  banner "Uninstalling BDOT"
  increase_indent

  info "Checking permissions..."
  root_check
  succeeded

  if [ "$SVC_PRE" = "systemctl" ]; then
    info "Stopping service..."
    systemctl stop bindplane-otel-collector > /dev/null || error_exit "$LINENO" "Failed to stop service"
    succeeded

    info "Disabling service..."
    systemctl disable bindplane-otel-collector > /dev/null 2>&1 || error_exit "$LINENO" "Failed to disable service"
    succeeded
  elif [ "$SVC_PRE" = "service" ]; then
    info "Stopping service..."
    service bindplane-otel-collector stop
    succeeded

    info "Disabling service..."
    chkconfig bindplane-otel-collector on
    # rm -f /etc/init.d/bindplane-otel-collector
    succeeded
  elif [ "$SVC_PRE" = "mkssys" ]; then
    # Using case here to bypass =~ missing in the POSIX standard
    case "$(lssrc -s bindplane-otel-collector | grep collector)" in
      *active*)
        # The service is running. Stop it before removing it.
        info "Stopping service..."
        stopsrc -s bindplane-otel-collector
        ;;
    esac

    # Remove the service
    info "Disabling service..."
    if lsitab bpcollector > /dev/null 2>&1; then
      # Removing start on boot for the service
      rmitab bpcollector
    fi
    if lssrc -s bindplane-otel-collector > /dev/null 2>&1; then
      # Removing actual service entry
      rmssys -s bindplane-otel-collector
    fi

    succeeded
  else
    # This is an error state that should never be reached
    error_exit "Found an invalid SVC_PRE value in uninstall()"
  fi

  succeeded

  info "Removing package..."
  uninstall_package || error_exit "$LINENO" "Failed to remove package"
  succeeded
  decrease_indent

  banner "$(fg_green Uninstallation Complete!)"
}

check_aix_name_length()
{
  aix_name_size="$(lsattr -El sys0 -a max_logname | cut -d" " -f 2)"
  if [ "$aix_name_size" -lt 24 ]; then
    error "$LINENO" "Current system will result in '3004-694 Error adding Name is too long.' when attempting to create group and user"
    error "Current max: $aix_name_size"
    error "Please raise your limit to 24 characters or greater by issuing these command:"
    error "    chdev -lsys0 -a max_logname=<NUM>"
    error "and then rebooting the system before attempting to run this script again"
    error_exit "Reference: https://www.ibm.com/support/pages/aix-security-change-maximum-length-user-name-group-name-or-password"
  fi
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
      -gl|--gpg-tar-url)
        gpg_tar_url=$2
        shift 2
        ;;
      -f | --file)
        package_path=$2
        shift 2
        ;;
      -gf|--gpg-tar-file)
        gpg_tar_path=$2
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
      --no-gpg-check)
        skip_gpg_check="true"
        shift 1
        ;;
      -r | --uninstall)
        uninstall
        exit 0
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

  validate_version
  interactive_check

  # AIX needs a special check
  if [ "$os" = "aix" ]; then
    check_aix_name_length
  fi

  connection_check
  offline_check
  setup_installation
  install_package
  display_results
}

main "$@"
