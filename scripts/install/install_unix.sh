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

# Cache OS type to avoid repeated subshell forks
OS_TYPE=$(uname -s)

# Reads optional package overrides. Users should deploy the override
# file before installing BDOT for the first time. The override should
# not be modified unless uninstalling and re-installing.
[ -f /etc/default/observiq-otel-collector ] && . /etc/default/observiq-otel-collector
[ -f /etc/sysconfig/observiq-otel-collector ] && . /etc/sysconfig/observiq-otel-collector

# The collectors installation directory
: "${BDOT_CONFIG_HOME:=/opt/observiq-otel-collector}"

# Allow configurable runtime user/group (used for permissions and manager.yaml)
: "${BDOT_USER:=bdot}"
: "${BDOT_GROUP:=bdot}"

# Agent Constants
PACKAGE_NAME="observiq-otel-collector"
DOWNLOAD_BASE="https://bdot.bindplane.com"

# Determine the service manager for prereqs. AIX always uses SRC, even when a
# systemctl or service command is also on PATH.
if [ "$OS_TYPE" = "AIX" ] && command -v mkssys > /dev/null 2>&1; then
  SVC_PRE=mkssys
elif command -v systemctl > /dev/null 2>&1; then
  SVC_PRE=systemctl
elif command -v service > /dev/null 2>&1; then
  SVC_PRE=service
fi

# Script Constants
COLLECTOR_USER="${BDOT_USER}"
COLLECTOR_GROUP="${BDOT_GROUP}"
COLLECTOR_USER_LEGACY="observiq-otel-collector"
TMP_DIR=${TMPDIR:-"/tmp"} # Allow this to be overriden by cannonical TMPDIR env var

# Remove the user install_aix created if the install fails, so a re-run creates it again.
# force_exit calls this too: AIX's ksh88 skips EXIT traps when a function kills or set -e ends the shell,
# and runs a function's own EXIT trap when the function returns, so the trap lives here.
created_aix_user=false
remove_created_aix_user() {
  if [ "$created_aix_user" = true ]; then rmuser -p "$COLLECTOR_USER" > /dev/null 2>&1; fi
}
trap remove_created_aix_user EXIT
MANAGEMENT_YML_PATH="${BDOT_CONFIG_HOME}/manager.yaml"
PREREQS="curl printf $SVC_PRE sed uname cut"
if [ "$OS_TYPE" = "AIX" ]; then
  # AIX has no curl without the Toolbox, so base Perl can download instead. It installs from a tar.gz.
  PREREQS="curl|perl printf $SVC_PRE sed uname cut tr tar gunzip awk"
  # AIX SRC (System Resource Controller) uses two identifiers, which must match the updater's:
  #   Group name: used with -g for startsrc, stopsrc and the inittab entry
  #   Subsystem name: used with -s for mkssys, rmssys and lssrc
  AIX_SRC_GROUP="oiqcollector"
  AIX_SRC_SUBSYSTEM="observiq-otel-collector"
  # Stands in for the systemd unit's Environment lines. One line of space-separated
  # KEY=value pairs, since startsrc -e splits only on spaces.
  AIX_ENV_FILE="/etc/observiq-otel-collector.env"
fi
SCRIPT_NAME="$0"
INDENT_WIDTH='  '
indent=""
non_interactive=false
error_mode=false
skip_gpg_check=false
# require_gpg_check fails the install, instead of skipping verification, when verification tools are missing
require_gpg_check=false

# package_out_file_path is the full path to the downloaded package (e.g. "/tmp/observiq-otel-collector_linux_amd64.deb")
package_out_file_path="unknown"

# gpg_tar_out_file_path is the full path to the downloaded GPG tar.gz file (e.g. "/tmp/bdot-gpg-keys.tar.gz")
gpg_tar_out_file_path="unknown"

offline_installation=false

# RPM_GPG_KEYS_TO_REMOVE lists revoked rpm key names to remove; the bundle's
# rpm-revocations.txt adds more. See signature/gpg/revocations.md.
RPM_GPG_KEYS_TO_REMOVE=""

# Colors
if [ "$non_interactive" = "false" ] && [ "$OS_TYPE" != AIX ]; then
  num_colors=$(tput colors 2>/dev/null || echo 0)
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
    if [ "$OS_TYPE" != "AIX" ] && command -v sed >/dev/null; then
      command printf -- "$@" | sed -r "$sed_ignore s/^/$indent/g"  # Ignore sole reset characters if defined
    else
      # Ignore $* suggestion as this breaks the output
      # shellcheck disable=SC2145
      command printf -- "$indent$@"
    fi
  fi
}

increase_indent() { indent="$INDENT_WIDTH$indent" ; }
decrease_indent() { indent="${indent#*"$INDENT_WIDTH"}" ; }

# Color functions reset only when given an argument
bold() { command printf "$bold$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }
underline() { command printf "$underline$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }
standout() { command printf "$standout$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }
# Ignore "parameters are never passed"
# shellcheck disable=SC2120
reset() { command printf "$reset$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }
bg_black() { command printf "$bg_black$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }
bg_blue() { command printf "$bg_blue$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }
bg_cyan() { command printf "$bg_cyan$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }
bg_green() { command printf "$bg_green$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }
bg_magenta() { command printf "$bg_magenta$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }
bg_red() { command printf "$bg_red$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }
bg_white() { command printf "$bg_white$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }
bg_yellow() { command printf "$bg_yellow$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }
fg_black() { command printf "$fg_black$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }
fg_blue() { command printf "$fg_blue$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }
fg_cyan() { command printf "$fg_cyan$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }
fg_green() { command printf "$fg_green$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }
fg_magenta() { command printf "$fg_magenta$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }
fg_red() { command printf "$fg_red$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }
fg_white() { command printf "$fg_white$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }
fg_yellow() { command printf "$fg_yellow$*$(if [ -n "$1" ]; then command printf "$reset"; fi)" ; }

# Intentionally using variables in format string
# shellcheck disable=SC2059
info() { printf "$*\\n" ; }
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
success() { printf "$fg_green$*$reset\\n" ; }
# Ignore 'arguments are never passed'
# shellcheck disable=SC2120
prompt() {
  if [ "$1" = 'n' ]; then
    command printf "y/$(fg_red '[n]'): "
  else
    command printf "$(fg_green '[y]')/n: "
  fi
}

bindplane_banner()
{
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

separator() { printf "===================================================\\n" ; }

banner()
{
  printf "\\n"
  separator
  printf "| %s\\n" "$*" ;
  separator
}

usage()
{
  increase_indent
  USAGE=$(cat <<EOF
Usage:
  $(fg_yellow '-v, --version')
      Defines the version of the Bindplane Agent.
      If not provided, this will default to the latest version.
      Alternatively the COLLECTOR_VERSION environment variable can be
      set to configure the agent version.
      Example: '-v 1.2.12' will download 1.2.12.

  $(fg_yellow '-r, --uninstall')
      Stops the agent services and uninstalls the agent.

  $(fg_yellow '-l, --url')
      Defines the URL that the components will be downloaded from.
      If not provided, this will default to Bindplane Agent\'s GitHub releases.
      Example: '-l http://my.domain.org/observiq-otel-collector' will download from there.

  $(fg_yellow '-gl, --gpg-tar-url')
      Defines the URL that the GPG tar file will be downloaded from.
      If not provided, this will default to Bindplane Agent\'s GitHub releases.
      Example: '-gl http://my.domain.org/bdot-gpg-keys.tar.gz' will download from there.

  $(fg_yellow '-b, --base-url')
      Defines the base of the download URL used in conjunction with the version to download the package and GPG tar file.
      '{base_url}/v{version}/{PACKAGE_NAME}_v{version}_linux_{os_arch}.{package_type}'
      and
      '{base_url}/v{version}/gpg-keys.tar.gz'
      If not provided, this will default to '$DOWNLOAD_BASE'.
      Example: '-b http://my.domain.org/observiq-otel-collector/binaries' will be used as the base of the download URL.

  $(fg_yellow '-f, --file')
      Install Agent from a local file instead of downloading from a URL.
      Example: '-f /path/to/observiq-otel-collector_v1.2.12_linux_amd64.deb' will install from the local file.
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

  $(fg_yellow '--require-gpg-check')
      Requires GPG signature verification of the package. Verification needs gpg,
      tar, gzip, awk, sed, grep, tr, and cut (and ar for deb packages). By default,
      when this system lacks them, the script warns and installs without
      verifying. With this flag, it stops the install instead. When the tools
      are present, verification runs the same either way.

      This option is incompatible with '--no-gpg-check' and will cause the script
      to exit with an error if both are specified.

  $(fg_yellow '-q, --quiet')
    Use quiet (non-interactive) mode to run the script in headless environments.
    
    Note: If a GPG signature verification failure occurs during installation and
    '--no-gpg-check' was not specified, the script will exit immediately without
    prompting the user to continue. For interactive handling of verification
    failures, do not use the '--quiet' flag.

EOF
  )
  info "$USAGE"
  decrease_indent
  return 0
}

force_exit()
{
  remove_created_aix_user
  # Exit regardless of subshell level with no "Terminated" message
  kill -PIPE $$
  # Call exit to handle special circumstances (like running script during docker container build)
  exit 1
}

error_exit()
{
  line_num=$(if [ -n "$1" ]; then command printf ":$1"; fi)
  error "ERROR ($SCRIPT_NAME$line_num): ${2:-Unknown Error}" >&2
  if [ -n "$0" ]; then
    increase_indent
    error "$*"
    decrease_indent
  fi
  force_exit
}

print_prereq_line()
{
  if [ -n "$2" ]; then
    command printf "\\n${indent}  - "
    command printf "[$1]: $2"
  fi
}

check_failure()
{
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

succeeded()
{
  increase_indent
  success "Succeeded!"
  decrease_indent
}

failed()
{
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

  # Check if major version is 1 and minor version is >= 82
  if [ "$major" = "1" ] && [ "$minor" -ge 82 ] 2>/dev/null; then
    succeeded
    return 0
  else
    failed
    error_exit "$LINENO" "Version $version is not supported. This script supports collector v1 version v1.82.0 or newer. Please use the script versioned with your desired collector version."
  fi
}

# This will set all installation variables
# at the beginning of the script.
setup_installation()
{
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

    success "Configuration complete!"
    decrease_indent
}

set_file_names() {
  # AIX ships the same tar.gz the updater uses
  if [ "$OS_TYPE" = "AIX" ]; then
    if [ -z "$version" ] ; then
      package_file_name="${PACKAGE_NAME}-aix-ppc64.tar.gz"
    else
      package_file_name="${PACKAGE_NAME}-v${version}-aix-ppc64.tar.gz"
    fi
  elif [ -z "$version" ] ; then
    package_file_name="${PACKAGE_NAME}_linux_${arch}.${package_type}"
  else
    package_file_name="${PACKAGE_NAME}_v${version}_linux_${arch}.${package_type}"
  fi
  package_out_file_path="$TMP_DIR/$package_file_name"

  gpg_tar_out_file_path="$TMP_DIR/bdot-gpg-keys.tar.gz"
}

set_proxy()
{
  if [ -n "$proxy" ]; then
    info "Using proxy from arguments: $proxy"
    if [ -n "$proxy_user" ]; then
      while [ -z "$proxy_password" ] ; do
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
  # AIX's uname -m is a machine ID, so ask for the processor type there
  if [ "$OS_TYPE" = "AIX" ]; then
    os_arch=$(uname -p)
  else
    os_arch=$(uname -m)
  fi
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
    # Only 64-bit AIX is supported
    powerpc)
      if [ "$OS_TYPE" = "AIX" ] && [ "$(bootinfo -y)" = "64" ]; then
        os_arch="ppc64"
      else
        error_exit "$LINENO" "Unsupported os arch: 32-bit powerpc"
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

# detect_distro_package_type prints the native package type ("deb" or "rpm") for
# this system. It uses a multi-layer fallback chain so that the presence of a
# cross-packaging tool (e.g. dpkg installed on Fedora) does not cause a wrong result.
#
# Fallback order:
#   1. /etc/os-release ID and ID_LIKE  (RHEL 7+, SLES 12+, all modern distros)
#   2. Distro-specific files           (RHEL 5, SLES 11, older CentOS/Fedora)
#   3. High-level package managers     (apt-get, dnf, yum, zypper)
#   4. Low-level packaging tools       (dpkg, rpm — least reliable)
#
# Prints nothing and returns 1 if detection fails.
detect_distro_package_type()
{
  # AIX installs from a tar.gz and runs under SRC
  if [ "$OS_TYPE" = "AIX" ]; then
    echo "mkssys"
    return 0
  fi

  # 1. /etc/os-release — most reliable on modern systems
  if [ -f /etc/os-release ]; then
    # Source in a subshell to avoid polluting the current environment
    _os_id=$(. /etc/os-release && echo "$ID")
    _os_id_like=$(. /etc/os-release && echo "${ID_LIKE:-}")

    # Combine ID and ID_LIKE for matching (ID_LIKE can contain multiple values)
    _os_ids="$_os_id $_os_id_like"
    case "$_os_ids" in
      *debian*|*ubuntu*|*raspbian*|*linuxmint*)
        echo "deb"
        return 0
        ;;
      *rhel*|*centos*|*fedora*|*rocky*|*almalinux*|*amzn*|*sles*|*suse*)
        echo "rpm"
        return 0
        ;;
    esac
  fi

  # 2. Distro-specific files (covers RHEL 5, SLES 11 SP4, and similar legacy systems)
  if [ -f /etc/debian_version ]; then
    echo "deb"
    return 0
  fi

  if [ -f /etc/redhat-release ] || [ -f /etc/centos-release ] || [ -f /etc/fedora-release ]; then
    echo "rpm"
    return 0
  fi

  # SuSE-release was used through SLES 11; removed in SLES 12+
  if [ -f /etc/SuSE-release ]; then
    echo "rpm"
    return 0
  fi

  # 3. High-level package managers (stronger signal than the low-level tools)
  if command -v apt-get > /dev/null 2>&1; then
    echo "deb"
    return 0
  fi

  if command -v dnf > /dev/null 2>&1 || command -v yum > /dev/null 2>&1 || command -v zypper > /dev/null 2>&1; then
    echo "rpm"
    return 0
  fi

  # 4. Last resort: low-level tools. These are the least reliable because
  # cross-packaging tools (e.g. dpkg on an RPM system) can cause false positives.
  if command -v dpkg > /dev/null 2>&1; then
    echo "deb"
    return 0
  fi

  if command -v rpm > /dev/null 2>&1; then
    echo "rpm"
    return 0
  fi

  return 1
}

# Set the package type before install
set_package_type()
{
  # if package_path is set get the file extension otherwise look at what's available on the system.
  # AIX always installs a tar.gz.
  if [ -n "$package_path" ] && [ "$OS_TYPE" != "AIX" ]; then
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
    package_type=$(detect_distro_package_type) || error_exit "$LINENO" "Could not detect a supported package manager (deb or rpm) on this system"
  fi

}

# This will set the urls to use when downloading the agent and its plugins.
# These urls are constructed based on the --version flag or COLLECTOR_VERSION env variable.
# If not specified, the version defaults to whatever the latest release on github is.
set_download_urls()
{
  if [ -z "$url" ] ; then
    if [ -z "$base_url" ] ; then
      base_url=$DOWNLOAD_BASE
    fi

    if [ "$OS_TYPE" = "AIX" ]; then
      collector_download_url="$base_url/v$version/${PACKAGE_NAME}-v${version}-aix-ppc64.tar.gz"
    else
      collector_download_url="$base_url/v$version/${PACKAGE_NAME}_v${version}_linux_${os_arch}.${package_type}"
    fi
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

set_opamp_endpoint()
{
  if [ -z "$opamp_endpoint" ] ; then
    opamp_endpoint="$ENDPOINT"
  fi

  OPAMP_ENDPOINT="$opamp_endpoint"
}

set_opamp_labels()
{
  if [ -z "$opamp_labels" ] ; then
    opamp_labels=$LABELS
  fi

  OPAMP_LABELS="$opamp_labels"

  if [ -n "$OPAMP_LABELS" ] && [ -z "$OPAMP_ENDPOINT" ]; then
    error_exit "$LINENO" "An endpoint must be specified when providing labels"
  fi
}

set_opamp_secret_key()
{
  if [ -z "$opamp_secret_key" ] ; then
    opamp_secret_key=$SECRET_KEY
  fi

  OPAMP_SECRET_KEY="$opamp_secret_key"

  if [ -n "$OPAMP_SECRET_KEY" ] && [ -z "$OPAMP_ENDPOINT" ]; then
    error_exit "$LINENO" "An endpoint must be specified when providing a secret key"
  fi
}

# Test connection to Bindplane if it was specified
connection_check()
{
  if [ -n "$check_bp_url" ] ; then
    if [ -n "$opamp_endpoint" ]; then
      HTTP_ENDPOINT="$(echo "${opamp_endpoint}" | sed -e 's#^ws#http#' -e 's#/v1/opamp$##')"
      info "Testing connection to Bindplane: $fg_magenta$HTTP_ENDPOINT$reset..."

      if http_check "${HTTP_ENDPOINT}"; then
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
check_prereqs()
{
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
root_check()
{
  system_user_name=$(id -un)
  if [ "${system_user_name}" != 'root' ]
  then
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

  if [ "$require_gpg_check" = "true" ] && [ "$skip_gpg_check" = "true" ]; then
    failed
    error_exit "$LINENO" "--require-gpg-check is incompatible with '--no-gpg-check'. These options cannot be used together."
  fi

  # Incompatible with proxies unless both username and password are passed
  if [ "$non_interactive" = "true" ] && [ -n "$proxy_password" ]
  then 
    failed
    error_exit "$LINENO" "The proxy password must be set via the command line argument -P, if called non-interactively."
  fi

  # Incompatible with checking the BP url since it can be interactive on failed connection
  if [ "$non_interactive" = "true" ] && [ "$check_bp_url" = "true" ]
  then 
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
os_check()
{
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

# This will check if the system architecture is supported.
os_arch_check()
{
  info "Checking for valid operating system architecture..."
  if [ "$OS_TYPE" = "AIX" ]; then
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
    x86_64|aarch64|ppc64|ppc64le|arm64|aarch64_be|armv8b|armv8l|arm|armv6l|armv7l)
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
dependencies_check()
{
  info "Checking for script dependencies..."
  FAILED_PREREQS=''
  for prerequisite in $PREREQS; do
    # Support alternate tools with pipe syntax (e.g. "shasum|sha256sum").
    # Checks each alternate in order via command -v. The first one found is
    # resolved, and a command variable is created with the full invocation
    # including any required flags (see resolve_alternate_cmd).
    case "$prerequisite" in
      *"|"*)
        _found=false
        # Temporarily change the field separator to pipe so we can iterate
        # over each alternate tool in the "tool1|tool2|toolN" entry
        _save_IFS="$IFS"; IFS="|"
        for _alt in $prerequisite; do
          IFS="$_save_IFS"
          if command -v "$_alt" >/dev/null; then
            _found=true
            resolve_alternate_cmd "$prerequisite" "$_alt"
            break
          fi
          IFS="|"
        done
        IFS="$_save_IFS"
        if [ "$_found" = false ]; then
          # Build a human-readable display string from the pipe-separated
          # alternatives. "shasum|sha256sum" becomes "shasum or sha256sum".
          # Uses only shell builtins — no tr/sed — because those tools may
          # themselves be the missing prereqs we're reporting on.
          _display=""
          _tmp_IFS="$IFS"; IFS="|"
          for _item in $prerequisite; do
            if [ -z "$_display" ]; then
              _display="$_item"
            else
              _display="$_display or $_item"
            fi
          done
          IFS="$_tmp_IFS"
          # Each missing prereq group is added to FAILED_PREREQS, separated
          # by commas. The final error message lists all missing prereqs:
          # e.g. "curl, shasum or sha256sum, gpg2 or gpg"
          if [ -z "$FAILED_PREREQS" ]; then
            FAILED_PREREQS="${fg_red}${_display}${reset}"
          else
            FAILED_PREREQS="$FAILED_PREREQS, ${fg_red}${_display}${reset}"
          fi
        fi
        ;;
      *)
        if command -v "$prerequisite" >/dev/null; then
          continue
        else
          if [ -z "$FAILED_PREREQS" ]; then
            FAILED_PREREQS="${fg_red}$prerequisite${reset}"
          else
            FAILED_PREREQS="$FAILED_PREREQS, ${fg_red}$prerequisite${reset}"
          fi
        fi
        ;;
    esac
  done

  if [ -n "$FAILED_PREREQS" ]; then
    failed
    error_exit "$LINENO" "The following dependencies are required by this script: [$FAILED_PREREQS]"
  fi
  succeeded

}

resolve_alternate_cmd()
{
  case "$1" in
    "curl|perl")
      HTTP_CLIENT="$2"
      ;;
  esac
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
  if detect_distro_package_type > /dev/null; then
      succeeded
  else
      failed
      error_exit "$LINENO" "Could not detect a supported package manager (deb or rpm) on this system"
  fi

  if [ "$OS_TYPE" = "AIX" ]; then
    info "Checking for required AIX commands..."
    if command -v mkssys > /dev/null 2>&1 \
      && command -v rmssys > /dev/null 2>&1 \
      && command -v mkgroup > /dev/null 2>&1 \
      && command -v mkuser > /dev/null 2>&1 \
      && command -v rmuser > /dev/null 2>&1 \
      && command -v startsrc > /dev/null 2>&1 \
      && command -v stopsrc > /dev/null 2>&1 \
      && command -v lssrc > /dev/null 2>&1 \
      && command -v mkitab > /dev/null 2>&1 \
      && command -v rmitab > /dev/null 2>&1 \
      && command -v lsitab > /dev/null 2>&1; then
      succeeded
    else
      failed
      error_exit "$LINENO" "Could not find required AIX commands on the system"
    fi
  fi
}

# latest_version gets the tag of the latest release, without the v prefix.
latest_version()
{
  http_get https://bdot.bindplane.com/latest
}

# HTTP helpers. They use curl when it is installed, and otherwise fall back to
# HTTP::Tiny from the base AIX perl.rte fileset, so AIX installs need no AIX Toolbox.
# Perl verifies TLS against Mozilla::CA (also in perl.rte) when it is available.
PERL_HTTP='
use strict; use HTTP::Tiny;
my ($mode, $url, $out) = @ARGV;
my %o = (verify_SSL => 1, timeout => ($mode eq "check" ? 20 : 300));
$o{SSL_options} = { SSL_ca_file => Mozilla::CA::SSL_ca_file() } if eval { require Mozilla::CA; 1 };
$o{proxy} = $ENV{BDOT_HTTP_PROXY} if $ENV{BDOT_HTTP_PROXY};
my $h = HTTP::Tiny->new(%o);
if ($mode eq "download") {
  open(my $fh, ">", $out) or die "cannot write $out: $!\n";
  binmode $fh;
  my $r = $h->request("GET", $url, { data_callback => sub { print {$fh} $_[0] if $_[1]{status} =~ /^2/ } });
  close $fh;
  print STDERR "$r->{status} $r->{reason}\n" unless $r->{success};
  exit($r->{success} ? 0 : 1);
}
my $r = $h->get($url);
exit($r->{status} == 599 ? 1 : 0) if $mode eq "check";
print $r->{content} if $r->{success};
exit($r->{success} ? 0 : 1);
'

# http_proxy_url prints the --proxy value with any --proxy-user credentials embedded, for Perl.
# ponytail: credentials are not URL-encoded; a password with @ or : needs curl.
http_proxy_url() {
  [ -n "$proxy" ] || return 0
  case "$proxy" in
    *://*) _scheme="${proxy%%://*}://"; _hostport="${proxy#*://}" ;;
    *) _scheme="http://"; _hostport="$proxy" ;;
  esac
  if [ -n "$proxy_user" ]; then
    command printf '%s%s:%s@%s' "$_scheme" "$proxy_user" "$proxy_password" "$_hostport"
  else
    command printf '%s%s' "$_scheme" "$_hostport"
  fi
}

# http_get prints the body of a small text response.
http_get() {
  if [ "$HTTP_CLIENT" = "perl" ]; then
    perl -e "$PERL_HTTP" get "$1"
  else
    curl -s "$1"
  fi
}

# http_check succeeds when the server answers within 20 seconds.
http_check() {
  if [ "$HTTP_CLIENT" = "perl" ]; then
    perl -e "$PERL_HTTP" check "$1"
  else
    curl --max-time 20 -s "$1" > /dev/null
  fi
}

# http_download downloads $1 to $2 through any configured proxy, failing on HTTP errors.
http_download() {
  if [ "$HTTP_CLIENT" = "perl" ]; then
    BDOT_HTTP_PROXY=$(http_proxy_url) perl -e "$PERL_HTTP" download "$1" "$2"
  else
    eval curl -L "$proxy_args" "\"\$1\"" -o "\"\$2\"" --progress-bar --fail
  fi
}

# This will install the package by downloading the archived agent,
# extracting the binaries, and then removing the archive.
install_package()
{
  banner "Installing Bindplane Agent"
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

    http_download "$collector_download_url" "$package_out_file_path" || error_exit "$LINENO" "Failed to download package"

    if [ -n "$proxy" ]; then
      info "Downloading GPG key tar file from $gpg_tar_download_url using proxy..."
    else 
      info "Downloading GPG key tar file from $gpg_tar_download_url..."
    fi

    http_download "$gpg_tar_download_url" "$gpg_tar_out_file_path" || error_exit "$LINENO" "Failed to download GPG tar file"

    # AIX packages are verified against a detached signature published beside the package
    if [ "$package_type" = "mkssys" ] && [ "$skip_gpg_check" != "true" ]; then
      info "Downloading package signature from ${collector_download_url}.gpg.sig..."
      http_download "${collector_download_url}.gpg.sig" "${package_out_file_path}.gpg.sig" || error_exit "$LINENO" "Failed to download package signature"
    fi
    succeeded
  fi

  info "Installing package..."

  # Verify the package signature
  # Capture GPG verification output to display failure details
  # Temporarily disable set -e to allow capture of failing command output
  set +e
  gpg_verify_output=$(verify_package 2>&1)
  gpg_verify_exit_code=$?
  set -e

  # Say why verification failed, even in quiet mode
  if [ -n "$gpg_verify_output" ]; then
    if [ $gpg_verify_exit_code -ne 0 ]; then error_mode=true; fi
    # The captured lines are already indented
    _indent=$indent; indent=""
    printf "%s\n" "$gpg_verify_output"
    indent=$_indent
    error_mode=false
  fi
  
  # Return code 3 never installs, even interactively
  if [ $gpg_verify_exit_code -eq 3 ]; then
    error_exit "$LINENO" "Refusing to install: the package signing key is revoked or expired, the package was altered, or the package or key bundle is malformed."
  fi

  if [ $gpg_verify_exit_code -ne 0 ]; then
    if [ "$non_interactive" = "true" ]; then
      # In quiet mode, fail immediately on GPG verification failure
      error_exit "$LINENO" "Failed to verify package signature. Use '--no-gpg-check' to skip verification."
    else
      # Interactive: explain the failure printed above, then ask
      increase_indent
      printf "\\n${indent}The package signature could not be verified. This may indicate:\n"
      printf "${indent}  - The GPG keys are not properly installed or accessible\n"
      printf "${indent}  - The package has been tampered with\n"
      printf "${indent}  - The package is unsigned, or signed by a key that is not in the BDOT key bundle\n"
      printf "\\n${indent}$(fg_yellow 'Continuing without signature verification is NOT RECOMMENDED unless you have independently verified the package authenticity.')\\n\\n"
      decrease_indent
      
      command printf "${indent}Do you wish to continue installation without GPG verification? "
      prompt "n"
      read -r gpg_override_input
      printf "\\n"
      
      if [ "$gpg_override_input" != "y" ] && [ "$gpg_override_input" != "Y" ]; then
        error_exit "$LINENO" "Installation aborted due to GPG verification failure."
      fi
      
      warn "Continuing installation without GPG verification. Ensure package authenticity has been verified through other means."
    fi
  fi
  unpack_package || error_exit "$LINENO" "Failed to extract package"
  succeeded

  # If an endpoint was specified, we need to write the manager.yaml
  if [ -n "$OPAMP_ENDPOINT" ]; then
    info "Creating manager yaml..."
    create_manager_yml "$MANAGEMENT_YML_PATH"
    succeeded
  fi

  if [ "$SVC_PRE" = "systemctl" ]; then
    if [ "$(systemctl is-enabled observiq-otel-collector)" = "enabled" ]; then
      # The unit is already enabled; It may be running, too, if this was an upgrade.
      # We'll want to restart, which will start it if it wasn't running already,
      # and restart in the case that this was an upgrade on a running agent.
      info "Restarting service..."
      systemctl restart observiq-otel-collector > /dev/null 2>&1 || error_exit "$LINENO" "Failed to restart service"
      succeeded
    else
      info "Enabling service..."
      systemctl enable --now observiq-otel-collector > /dev/null 2>&1 || error_exit "$LINENO" "Failed to enable service"
      succeeded
    fi
  elif [ "$SVC_PRE" = "mkssys" ]; then
    case "$(lssrc -s "$AIX_SRC_SUBSYSTEM" 2>/dev/null)" in
      *active*)
        # install_aix stops and starts the service for an upgrade, so it's already running.
        info "Service is running."
        succeeded
        ;;
      *inoperative*)
        info "Starting service..."
        startsrc -g "$AIX_SRC_GROUP" -e "$(cat "$AIX_ENV_FILE")" > /dev/null || error_exit "$LINENO" "Failed to start service"
        succeeded
        ;;
      *)
        info "Creating, enabling and starting service..."
        # The subsystem and inittab entry match what the updater writes, so an update
        # replaces them in place. SRC has no working directory, so every path is absolute.
        mkssys -s "$AIX_SRC_SUBSYSTEM" -G "$AIX_SRC_GROUP" -p "$BDOT_CONFIG_HOME/observiq-otel-collector" -u 0 -S -n 15 -f 9 -R \
          -a "--config $BDOT_CONFIG_HOME/config.yaml --manager $BDOT_CONFIG_HOME/manager.yaml --logging $BDOT_CONFIG_HOME/logging.yaml" > /dev/null \
          || error_exit "$LINENO" "Failed to create service"
        if lsitab "$AIX_SRC_GROUP" > /dev/null 2>&1; then
          rmitab "$AIX_SRC_GROUP"
        fi
        # shellcheck disable=SC2016
        # init evaluates the $(cat) at boot, so it stays literal here.
        mkitab "$AIX_SRC_GROUP"':23456789:once:startsrc -g '"$AIX_SRC_GROUP"' -e "$(cat '"$AIX_ENV_FILE"')"' || error_exit "$LINENO" "Failed to enable service"
        startsrc -g "$AIX_SRC_GROUP" -e "$(cat "$AIX_ENV_FILE")" > /dev/null || error_exit "$LINENO" "Failed to start service"
        succeeded
        ;;
    esac
  else
    case "$(service observiq-otel-collector status)" in
      *running*)
        # The service is running.
        # We'll want to restart.
        info "Restarting service..."
        service observiq-otel-collector restart > /dev/null 2>&1 || error_exit "$LINENO" "Failed to restart service"
        succeeded
        ;;
      *)
        info "Enabling and starting service..."
        chkconfig observiq-otel-collector on > /dev/null 2>&1 || error_exit "$LINENO" "Failed to enable service"
        service observiq-otel-collector start > /dev/null 2>&1 || error_exit "$LINENO" "Failed to start service"
        succeeded
        ;;
    esac
  fi

  success "Bindplane Agent installation complete!"
  decrease_indent
}

# verify_package returns 1 for a failure the user may override, and 3, which never installs,
# for a revoked or expired key, an altered package, or a malformed bundle.
verify_package() {
  # If GPG check is skipped, return success immediately
  if [ "$skip_gpg_check" = "true" ]; then
    warn "GPG signature verification is being bypassed with the '--no-gpg-check' flag."
    warn "This disables a critical security check and should only be used if your organization policies permit it."
    return 0
  fi

  _missing=$(verification_missing_tools)
  if [ -n "$_missing" ]; then
    if [ "$require_gpg_check" = "true" ]; then
      error "Package signature verification requires: [$_missing]. Install them, or drop '--require-gpg-check' to install without verification."
      return 1
    fi
    warn "Skipping package signature verification: this system lacks [$_missing]."
    warn "Install them to verify the package, or use '--require-gpg-check' to stop the install when they are missing."
    return 0
  fi

  # A private keyring, so the host's own keys never take part. Base AIX has no mktemp, and
  # mkdir fails rather than reuse a directory that already exists.
  if ! GPG_DIR=$(mktemp -d "$TMP_DIR/bdot-gpg.XXXXXX" 2> /dev/null || { mkdir -m 700 "$TMP_DIR/bdot-gpg.$$" && echo "$TMP_DIR/bdot-gpg.$$"; }); then
    error "Failed to create a temporary GPG directory"
    return 1
  fi
  trap 'gpg_cleanup; exit 1' HUP INT TERM

  _verify_rc=0
  # AIX base tar has no -z, so decompress separately there
  if [ "$OS_TYPE" = "AIX" ]; then
    _gpg_extract_ok=true
    gunzip -c "$gpg_tar_out_file_path" | (cd "$GPG_DIR" && tar -xf -) > /dev/null 2>&1 || _gpg_extract_ok=false
  elif tar -xzf "$gpg_tar_out_file_path" -C "$GPG_DIR" > /dev/null 2>&1; then
    _gpg_extract_ok=true
  else
    _gpg_extract_ok=false
  fi
  if [ "$_gpg_extract_ok" != "true" ]; then
    error "Failed to extract GPG key tar file"
    _verify_rc=1
  else
    case "$package_type" in
      deb) verify_package_deb || _verify_rc=$? ;;
      mkssys) verify_package_aix || _verify_rc=$? ;;
      rpm) verify_package_rpm || _verify_rc=$? ;;
      *)
        error "Unrecognized package type"
        _verify_rc=1
        ;;
    esac
  fi

  gpg_cleanup
  trap - HUP INT TERM
  return $_verify_rc
}

# gpg_cleanup stops gpg daemons in the temporary keyrings and removes them.
gpg_cleanup() {
  for _home in "$GPG_DIR" "$GPG_DIR"/*/; do
    [ -d "$_home" ] && GNUPGHOME="$_home" gpgconf --kill all > /dev/null 2>&1 || true
  done
  rm -rf "$GPG_DIR"
}

# verification_missing_tools prints the verification tools the host lacks.
verification_missing_tools() {
  _missing=""
  _tools="gpg tar gzip awk sed grep tr cut"
  [ "$package_type" = "deb" ] && _tools="$_tools ar"
  # od and tail feed the legacy rpm check
  if [ "$package_type" = "rpm" ] && rpm_lacks_subkey_support; then _tools="$_tools od tail"; fi
  if [ "$package_type" = "mkssys" ]; then
    # AIX verifies with base Perl and openssl when gpg is not installed
    _tools="perl tar gunzip awk sed grep tr cut"
    command -v gpg > /dev/null 2>&1 || command -v openssl > /dev/null 2>&1 || _missing="gpg or openssl"
  fi
  for _tool in $_tools; do
    command -v "$_tool" > /dev/null 2>&1 || _missing="${_missing:+$_missing, }$_tool"
  done
  command printf '%s' "$_missing"
}

# literal escapes text for the output functions, which treat it as a printf format.
literal() { command printf '%s' "$1" | tr -c '[:print:]\n' '?' | sed 's/[%\\]/&&/g'; }

# verification_check fails before download when --require-gpg-check is set and verification
# tools are missing. Without it, verify_package warns and skips verification. It needs
# package_type, so it runs after setup_installation.
verification_check() {
  [ "$skip_gpg_check" = "true" ] && return 0
  [ "$require_gpg_check" = "true" ] || return 0
  _missing=$(verification_missing_tools)
  if [ -n "$_missing" ]; then
    error_exit "$LINENO" "Package signature verification requires: [$_missing]. Install them, or drop '--require-gpg-check' to install without verification."
  fi
}

# gpg_import_bundle imports the bundle's keys and revocation certificates.
# gpg's exit code is ignored: gnupg2-minimal exits nonzero after a good import. Returns 3 when
# a certificate revokes no bundle key, since gpg would silently ignore it.
gpg_import_bundle() {
  GNUPGHOME="$GPG_DIR" gpg --batch --import "$GPG_DIR/bdot-public-gpg-key.asc" > /dev/null 2>&1 || true
  if ! GNUPGHOME="$GPG_DIR" gpg --batch --with-colons --list-keys 2>/dev/null | grep -q '^pub:'; then
    error "Failed to import public key"
    return 1
  fi
  for _cert in "$GPG_DIR/deb-revocations/"*; do
    [ -f "$_cert" ] || continue
    GNUPGHOME="$GPG_DIR" gpg --batch --import "$_cert" > /dev/null 2>&1 || true
    _revoked_id=$(LC_ALL=C GNUPGHOME="$GPG_DIR" gpg --batch --list-packets "$_cert" 2>/dev/null | \
      awk '/^:signature packet:/ { for (i = 1; i < NF; i++) if ($i == "keyid") id = toupper($(i + 1)) }
           / sigclass 0x20/ && id != "" { print id; exit }')
    if [ -z "$_revoked_id" ] || ! GNUPGHOME="$GPG_DIR" gpg --batch --with-colons --list-keys 2>/dev/null | \
        awk -F: -v id="$_revoked_id" '$1 == "pub" && $5 == id && $2 == "r" { found = 1 } END { exit !found }'; then
      error "Revocation certificate $(literal "${_cert##*/}") does not revoke a key in the BDOT key bundle"
      return 3
    fi
  done
  # Judge signers by this snapshot, not the live keyring, which a host gpg.conf can let
  # --verify extend. Doubled --with-fingerprint lists subkey fingerprints on gpg 2.0.
  if ! GNUPGHOME="$GPG_DIR" gpg --batch --with-colons --fixed-list-mode --with-fingerprint --with-fingerprint --list-keys > "$GPG_DIR/bundle-keys" 2> /dev/null; then
    error "Failed to list the bundle keys"
    return 1
  fi
}

# gpg_key_status <listing> <key ID or fingerprint> <signature epoch> prints ok, revoked,
# expired, unknown, or malformed. gpg --verify exits 0 for revoked and expired keys, so this
# reads the colon listing. v4 keys only; a signature made before expiry stays valid.
gpg_key_status() {
  # Read through cat so a missing listing still reaches the malformed-input check in END
  cat "$1" 2> /dev/null | awk -F: -v want="$(echo "$2" | tr '[:lower:]' '[:upper:]')" -v sigtime="$3" '
      function state(k) {
        if (val[k] == "r") return "revoked"
        if (expiry[k] != "" && sigtime + 0 >= expiry[k] + 0) return "expired"
        return "ok"
      }
      $1 == "pub" { prim = NR }
      $1 == "pub" || $1 == "sub" { cur = NR; id[cur] = $5; val[cur] = $2; expiry[cur] = $7; parent[cur] = prim }
      $1 == "fpr" { fpr[cur] = $10 }
      END {
        if ((length(want) != 16 && length(want) != 40) || want ~ /[^0-9A-F]/ || sigtime !~ /^[0-9]+$/ || sigtime + 0 <= 0) { print "malformed"; exit }
        for (k in id) {
          if (id[k] != substr(want, length(want) - 15)) continue
          if (length(want) == 40 && fpr[k] != want) continue
          s = state(parent[k]); if (s == "ok") s = state(k)
          print s; exit
        }
        print "unknown"
      }'
}

# gpg_key_verdict <key> <signature epoch> <label> maps gpg_key_status to an error and return code.
gpg_key_verdict() {
  case "$(gpg_key_status "$GPG_DIR/bundle-keys" "$1" "$2")" in
    ok) return 0 ;;
    revoked)
      error "$3 signing key $1 is revoked"
      return 3
      ;;
    expired)
      error "$3 signing key $1 had expired when it signed"
      return 3
      ;;
    malformed)
      error "$3 signature has no usable key ID or signing time"
      return 1
      ;;
    *)
      error "$3 was not signed by a key in the BDOT key bundle"
      return 1
      ;;
  esac
}

# gpg_verify_file <signature> <data> <label> checks a detached signature by gpg's untranslated
# status output. Returns 0 when valid, 1 when unverifiable, and 3 for a revoked or expired key
# or altered data; sets SIGNER_PRIMARY to the signer's primary key fingerprint.
gpg_verify_file() {
  SIGNER_PRIMARY=""
  OUTPUT=$(GNUPGHOME="$GPG_DIR" gpg --batch --keyserver-options no-auto-key-retrieve --status-fd 1 --verify "$1" "$2" 2> "$GPG_DIR/verify.err")
  EXIT_CODE=$?

  case "$OUTPUT" in
    *"[GNUPG:] REVKEYSIG"* | *"[GNUPG:] KEYREVOKED"*)
      error "$3 signing key is revoked"
      return 3
      ;;
    *"[GNUPG:] EXPSIG"*)
      error "$3 signature has expired"
      return 3
      ;;
    *"[GNUPG:] BADSIG"*)
      error "$3 was altered: its signature does not match its contents"
      return 3
      ;;
  esac
  # command printf: the script's printf wrapper prints nothing in quiet mode
  _validsig=$(command printf '%s\n' "$OUTPUT" | awk '$2 == "VALIDSIG" { print $3, $5; exit }')
  SIGNER_PRIMARY=$(command printf '%s\n' "$OUTPUT" | awk '$2 == "VALIDSIG" { print $12; exit }')
  if [ $EXIT_CODE -ne 0 ] || [ -z "$_validsig" ]; then
    error "$3 signature is invalid"
    [ -s "$GPG_DIR/verify.err" ] && error "$(literal "$(cat "$GPG_DIR/verify.err")")"
    return 1
  fi

  gpg_key_verdict "${_validsig% *}" "${_validsig#* }" "$3"
}

verify_package_deb() {
  # dpkg installs the first control.tar.* and data.tar.* it finds, but the signature covers
  # only these members, so the layout is pinned. A compression change must update it.
  _deb_members=$(ar t "$package_out_file_path" 2> /dev/null)
  if [ -z "$_deb_members" ]; then
    error "Package is not a valid Debian package; the download may be incomplete"
    return 1
  fi
  if [ "$_deb_members" = "$(command printf 'debian-binary\ncontrol.tar.gz\ndata.tar.gz')" ]; then
    error "Package is not signed, or its download is incomplete"
    return 1
  fi
  if [ "$_deb_members" != "$(command printf 'debian-binary\ncontrol.tar.gz\ndata.tar.gz\n_gpgorigin')" ]; then
    error "Package has unexpected contents: [$(literal "$(command printf '%s' "$_deb_members" | tr '\n' ' ')")]"
    return 3
  fi

  gpg_import_bundle || return $?

  if ! ar p "$package_out_file_path" _gpgorigin > "$GPG_DIR/_gpgorigin" 2> /dev/null; then
    error "Failed to extract package signature"
    return 1
  fi

  # Extract first, so a read failure is not taken for tampering
  if ! ar p "$package_out_file_path" debian-binary control.tar.gz data.tar.gz > "$GPG_DIR/signed-data" 2> /dev/null; then
    error "Failed to extract the signed package contents"
    return 1
  fi
  gpg_verify_file "$GPG_DIR/_gpgorigin" "$GPG_DIR/signed-data" "Package" || return $?

  success "Package signature is valid, and its key was neither revoked nor expired when it signed"
  return 0
}

# rpm_signing_key_check sets SIGNING_KEYID from the header signature packet and rejects an
# unknown, revoked, or expired key. It parses the packet since rpm translates its labels and
# prints a 1970 signing date for this field on RHEL.
rpm_signing_key_check() {
  _rpm_packet=$(LC_ALL=C rpm -qp --qf '%{RSAHEADER:armor}' "$package_out_file_path" 2> /dev/null | \
    LC_ALL=C GNUPGHOME="$GPG_DIR" gpg --batch --list-packets 2> /dev/null)
  SIGNING_KEYID=$(command printf '%s\n' "$_rpm_packet" | awk '/^:signature packet:/ { for (i = 1; i < NF; i++) if ($i == "keyid") { print toupper($(i + 1)); exit } }')
  _rpm_sig_time=$(command printf '%s\n' "$_rpm_packet" | awk '{ for (i = 1; i < NF; i++) if ($i == "created") { v = $(i + 1); sub(/,$/, "", v); print v; exit } }')
  case "$SIGNING_KEYID" in
    ????????????????) ;;
    *)
      error "Could not read the RPM signature"
      return 1
      ;;
  esac
  gpg_import_bundle || return $?
  gpg_key_verdict "$SIGNING_KEYID" "$_rpm_sig_time" "RPM"
}

# rpm_lacks_subkey_support succeeds for rpm older than 4.12, which cannot check subkey signatures.
rpm_lacks_subkey_support() {
  _rpm_version=$(LC_ALL=C rpm --version 2> /dev/null | awk '{ print $NF }')
  _rpm_major=${_rpm_version%%.*}
  _rpm_minor=${_rpm_version#*.}
  _rpm_minor=${_rpm_minor%%.*}
  case "$_rpm_major$_rpm_minor" in
    "" | *[!0-9]*) return 1 ;;
  esac
  [ "$_rpm_major" -lt 4 ] || { [ "$_rpm_major" -eq 4 ] && [ "$_rpm_minor" -lt 12 ]; }
}

# rpm_key_names <key ID> prints the lowercase names rpm --checksig may use for the key: its
# 8- and 16-hex IDs, its fingerprint, and its primary's fingerprint.
rpm_key_names() {
  cat "$GPG_DIR/bundle-keys" 2> /dev/null | \
    awk -F: -v id="$1" '
      $1 == "pub" { prim = 1 }
      $1 == "pub" || $1 == "sub" { hit = ($5 == id); next_fpr = 1 }
      $1 == "fpr" && next_fpr { if (prim) pfpr = $10; if (hit) f = $10; prim = 0; next_fpr = 0; if (hit) { print tolower(substr(id, 9) " " id " " f " " pfpr); exit } }'
}

# bundle_primary <rpm key version> prints the fingerprint of the bundle's primary key with that
# 8-hex ID (rpm 4) or fingerprint (rpm 6), and fails when there is none.
bundle_primary() {
  awk -F: -v v="$1" '
    $1 == "pub" { prim = 1; next }
    $1 == "fpr" && prim { f = tolower($10); prim = 0; if (f == v || substr(f, 33) == v) { print $10; found = 1; exit } }
    END { exit !found }' "$GPG_DIR/bundle-keys" 2> /dev/null
}

# rpm_entry_keyring <gpg-pubkey name> imports that rpm keyring entry into a new gpg home and
# prints the home's path.
rpm_entry_keyring() {
  _home=$(mktemp -d "$GPG_DIR/rpmkey.XXXXXX") || return 1
  rpm -qi "$1" 2> /dev/null | GNUPGHOME="$_home" gpg --batch --import > /dev/null 2>&1
  command printf '%s' "$_home"
}

# rpm_entry_has_key <gpg-pubkey name> <primary fingerprint> [key ID] succeeds when that rpm
# keyring entry carries the primary, and the key ID when given. A short-ID match alone could
# name another vendor's key.
rpm_entry_has_key() {
  _home=$(rpm_entry_keyring "$1") || return 1
  GNUPGHOME="$_home" gpg --batch --with-colons --fixed-list-mode --with-fingerprint --list-keys 2> /dev/null | \
    awk -F: -v f="$2" -v id="$3" '
      $1 == "pub" { p = 1 } $1 == "pub" || $1 == "sub" { if ($5 == id) hasid = 1 }
      $1 == "fpr" && p { if ($10 == f) found = 1; p = 0 }
      END { exit !(found && (id == "" || hasid)) }'
}

# rpm_version_matches <rpm key version> <fingerprint> succeeds when an rpm key name's version
# names the key: its 8-hex ID on rpm 4 or its fingerprint on rpm 6.
rpm_version_matches() {
  _lower=$(command printf '%s' "$2" | tr '[:upper:]' '[:lower:]')
  [ -n "$_lower" ] && { [ "$1" = "$_lower" ] || [ "$1" = "$(command printf '%s' "$_lower" | cut -c33-40)" ]; }
}

# rpm_list_revokes <primary fingerprint> succeeds when the revoked-key list names the key, by
# 8-hex ID (rpm 4) or fingerprint (rpm 6).
rpm_list_revokes() {
  for key in $_revoked_keys; do
    _version=${key#gpg-pubkey-}
    rpm_version_matches "${_version%-*}" "$1" && return 0
  done
  return 1
}

# rpm_refresh_stale_key <primary fingerprint> removes an installed copy of the signing key's
# primary that lacks the signing key, as an install before a subkey rotation leaves, so the
# import that follows can replace it: rpm 4 skips a key whose entry is already installed. It
# saves the old copy for rpm_restore_stale_key.
rpm_refresh_stale_key() {
  _stale_saved=""
  _key_names=$(rpm_key_names "$SIGNING_KEYID")
  [ "${_key_names##* }" = "$(command printf '%s' "$1" | tr '[:upper:]' '[:lower:]')" ] || return 0
  for _entry in $(rpm -qa 'gpg-pubkey*' 2> /dev/null); do
    _version=${_entry#gpg-pubkey-}
    rpm_version_matches "${_version%-*}" "$1" || continue
    rpm_entry_has_key "$_entry" "$1" || continue
    rpm_entry_has_key "$_entry" "$1" "$SIGNING_KEYID" && continue
    _home=$(rpm_entry_keyring "$_entry")
    GNUPGHOME="$_home" gpg --batch --armor --export > "$GPG_DIR/rpm-stale.asc" 2> /dev/null
    if ! rpm -e "$_entry" > /dev/null 2>&1; then
      error "Failed to remove outdated key $_entry"
      return 1
    fi
    _stale_saved=1
  done
}

# rpm_restore_stale_key puts back the copy rpm_refresh_stale_key removed.
rpm_restore_stale_key() {
  [ -n "$_stale_saved" ] && rpm --import "$GPG_DIR/rpm-stale.asc" > /dev/null 2>&1
  _stale_saved=""
}

# rpm_header_offset <rpm file> prints the byte offset of the main header, which follows the
# 96-byte lead and the signature header padded to 8 bytes.
rpm_header_offset() {
  # shellcheck disable=SC2046 # split od's bytes into the positional parameters
  set -- $(od -An -v -tu1 -j 96 -N 16 "$1" 2> /dev/null)
  [ $# -eq 16 ] && [ "$1 $2 $3 $4 $5 $6 $7 $8" = "142 173 232 1 0 0 0 0" ] || return 1
  _il=$((($9 << 24) + (${10} << 16) + (${11} << 8) + ${12}))
  _dl=$(((${13} << 24) + (${14} << 16) + (${15} << 8) + ${16}))
  command printf '%s' $((96 + (16 + 16 * _il + _dl + 7) / 8 * 8))
}

# rpm_legacy_verify checks SIGPGP with gpg over the header and payload rpm installs, for rpm
# older than 4.12. rpm cannot check the header signature there, so rpm_signing_key_check only
# screens its key ID. A package must carry a payload, since a header-only signature over the
# header alone would otherwise pass as SIGPGP.
rpm_legacy_verify() {
  LC_ALL=C rpm -qp --qf '%{SIGPGP:armor}' "$package_out_file_path" > "$GPG_DIR/rpm-payload.sig" 2> /dev/null
  if ! grep -q -- '-----BEGIN PGP SIGNATURE-----' "$GPG_DIR/rpm-payload.sig"; then
    error "RPM has no header and payload signature for gpg to check"
    return 1
  fi
  if ! _rpm_offset=$(rpm_header_offset "$package_out_file_path"); then
    error "Could not read the RPM header layout"
    return 1
  fi
  # shellcheck disable=SC2046 # split od's bytes into the positional parameters
  set -- $(od -An -v -tu1 -j "$_rpm_offset" -N 16 "$package_out_file_path" 2> /dev/null)
  if [ $# -ne 16 ] || [ "$1 $2 $3 $4" != "142 173 232 1" ]; then
    error "Could not read the RPM header layout"
    return 1
  fi
  _header_end=$((_rpm_offset + 16 + 16 * (($9 << 24) + (${10} << 16) + (${11} << 8) + ${12}) + ((${13} << 24) + (${14} << 16) + (${15} << 8) + ${16})))
  if [ "$(wc -c < "$package_out_file_path")" -le "$_header_end" ]; then
    error "RPM has no payload; the download may be incomplete"
    return 1
  fi
  if ! tail -c +$((_rpm_offset + 1)) "$package_out_file_path" > "$GPG_DIR/rpm-signed-data" 2> /dev/null; then
    error "Failed to extract the signed RPM contents"
    return 1
  fi
  gpg_verify_file "$GPG_DIR/rpm-payload.sig" "$GPG_DIR/rpm-signed-data" "RPM" || return $?
  # rpm verified no signature here, so apply the rpm revoked-key list to this signer too
  if [ -z "$SIGNER_PRIMARY" ]; then
    error "RPM signature names no primary key"
    return 1
  fi
  if rpm_list_revokes "$SIGNER_PRIMARY"; then
    error "RPM signing key $SIGNER_PRIMARY is revoked"
    return 3
  fi
}

# rpm_read_revoked_list sets _revoked_keys from RPM_GPG_KEYS_TO_REMOVE and the bundle's
# rpm-revocations.txt, and returns 3 for an entry that is not a bundle key's rpm name.
rpm_read_revoked_list() {
  # rpm -e runs as root: validate every entry before importing, with globbing off
  set -f
  # shellcheck disable=SC2046,SC2086 # split the list into one entry per line
  _revoked_keys=$(command printf '%s\n' $RPM_GPG_KEYS_TO_REMOVE $(tr -d '\r' 2> /dev/null < "$GPG_DIR/rpm-revocations.txt"))
  for key in $_revoked_keys; do
    if ! command printf '%s\n' "$key" | grep -qxE 'gpg-pubkey-[0-9a-f]+-[0-9a-f]+'; then
      set +f
      error "Revoked key list has an entry that is not an rpm key name: $(literal "$key")"
      return 3
    fi
    # Only bundle keys, so a bad bundle cannot strip host keys
    _version=${key#gpg-pubkey-}
    if ! bundle_primary "${_version%-*}" > /dev/null; then
      set +f
      error "Revoked key list names $key, which is not a key in the BDOT key bundle"
      return 3
    fi
  done
  # Entries are hex and dashes now, so globbing is harmless
  set +f
}

verify_package_rpm() {
  # Check the key against the bundle before rpm trusts it
  rpm_signing_key_check || return $?

  rpm_read_revoked_list || return $?

  # Revoked primaries are the listed ones plus any the bundle itself marks revoked. Remove
  # entries an earlier install left, matched by fingerprint so a wrong release or a foreign key
  # sharing the short ID stays.
  _remove_failed=""
  _revoked_fprs=$(awk -F: '$1 == "pub" { r = ($2 == "r"); next } $1 == "fpr" && r != "" { if (r) print $10; r = "" }' "$GPG_DIR/bundle-keys")
  for key in $_revoked_keys; do
    _version=${key#gpg-pubkey-}
    _revoked_fprs="$_revoked_fprs $(bundle_primary "${_version%-*}")"
  done
  for _fpr in $_revoked_fprs; do
    for _entry in $(rpm -qa 'gpg-pubkey*' 2> /dev/null); do
      _version=${_entry#gpg-pubkey-}
      rpm_version_matches "${_version%-*}" "$_fpr" || continue
      if rpm_entry_has_key "$_entry" "$_fpr" && ! rpm -e "$_entry" > /dev/null 2>&1; then
        error "Failed to remove revoked key $_entry"
        _remove_failed=1
      fi
    done
  done
  # Fail on a revoked signer before importing anything
  _key_names=$(rpm_key_names "$SIGNING_KEYID")
  if rpm_list_revokes "${_key_names##* }"; then
    error "RPM signing key $SIGNING_KEYID is revoked"
    return 3
  fi
  [ -z "$_remove_failed" ] || return 1

  # Import each key that is not revoked on its own, since rpm 4.11 merges a multi-key file into
  # one entry named after its last key. The imported keys stay even if a later check fails.
  for _fpr in $(awk -F: '$1 == "pub" { p = 1; next } $1 == "fpr" && p { print $10; p = 0 }' "$GPG_DIR/bundle-keys"); do
    case " $(command printf '%s' "$_revoked_fprs" | tr '\n' ' ') " in *" $_fpr "*) continue ;; esac
    rpm_refresh_stale_key "$_fpr" || return $?
    GNUPGHOME="$GPG_DIR" gpg --batch --armor --export "$_fpr" > "$GPG_DIR/rpm-key.asc" 2> /dev/null
    if ! IMPORT_OUTPUT=$(rpm --import "$GPG_DIR/rpm-key.asc" 2>&1); then
      error "Failed to import public key: $(literal "$IMPORT_OUTPUT")"
      rpm_restore_stale_key
      return 1
    fi
  done

  # rpm must hold the signing key, except before 4.12, where rpm cannot use it and gpg's
  # SIGPGP check decides. Read its keyring through gpg (--show-keys needs 2.1), taking only
  # the bundle's entries: gpg stops an import at a key it cannot parse, such as the OpenPGP v6
  # key EL10 ships, and skips every key after it.
  if ! rpm_lacks_subkey_support; then
    mkdir -m 700 "$GPG_DIR/rpmdb"
    for _fpr in $(awk -F: '$1 == "pub" { p = 1; next } $1 == "fpr" && p { print $10; p = 0 }' "$GPG_DIR/bundle-keys"); do
      for _rpm_key in $(rpm -qa 'gpg-pubkey*' 2> /dev/null); do
        _version=${_rpm_key#gpg-pubkey-}
        rpm_version_matches "${_version%-*}" "$_fpr" || continue
        rpm_entry_has_key "$_rpm_key" "$_fpr" || continue
        rpm -qi "$_rpm_key" 2> /dev/null
      done
    done | GNUPGHOME="$GPG_DIR/rpmdb" gpg --batch --import > /dev/null 2>&1 || true
    if ! GNUPGHOME="$GPG_DIR/rpmdb" gpg --batch --with-colons --list-keys 2> /dev/null | \
        awk -F: -v id="$SIGNING_KEYID" '($1 == "pub" || $1 == "sub") && $5 == id { found = 1 } END { exit !found }'; then
      error "RPM signing key $SIGNING_KEYID is not in the rpm keyring"
      return 1
    fi
  fi

  # rpm's exit code also fails for other keyless signatures, so judge the BDOT key's own line
  _checksig=$(LC_ALL=C rpm --checksig --verbose "$package_out_file_path" 2>&1 | tr '[:upper:]' '[:lower:]')
  # Any BAD line fails: a bad digest means the package was altered
  if command printf '%s\n' "$_checksig" | grep -qE ': bad( |$)'; then
    error "RPM signature is BAD"
    return 3
  fi

  _key_pattern=" ($(command printf '%s' "$_key_names" | tr ' ' '|')): "
  # rpm before 4.12 (EL6, EL7, Amazon Linux 2, SLES 12) cannot check subkey signatures and
  # leaves the payload unsigned, so gpg's SIGPGP check decides there, whatever rpm reports
  if rpm_lacks_subkey_support && command printf '%s\n' "$_checksig" | grep -qE "${_key_pattern}(ok|nokey)\$"; then
    rpm_legacy_verify || return $?
  elif command printf '%s\n' "$_checksig" | grep -qE "${_key_pattern}ok\$"; then
    :
  else
    error "RPM signature could not be checked against the BDOT key"
    return 1
  fi

  success "Package signature is valid, and its key was neither revoked nor expired when it signed"
  return 0
}

# PERL_PGP_PREPARE lets base openssl check an OpenPGP detached signature on AIX hosts
# without gpg. It reads the key bundle, any revocation certificates, and the signature,
# rejects revoked keys and signatures made outside the key's validity, then writes the
# signing key as PEM, the raw RSA signature, and the signed trailer for openssl to verify.
# Args: <key.asc> <revocation dir> <signature> <output dir>. Prints the openssl digest name.
PERL_PGP_PREPARE='
use strict; use MIME::Base64; use Digest::SHA qw(sha1);
my ($asc, $revdir, $sigfile, $out) = @ARGV;
# reject exits 3, which never installs; a plain die exits 255 for an unverifiable package
sub reject { print STDERR @_; exit 3 }
sub slurp { my $f = shift; open(my $h, "<", $f) or die "cannot read $f\n"; binmode $h; local $/; my $d = <$h>; close $h; $d }
sub dearmor {
  my $d = shift;
  return $d unless $d =~ /-----BEGIN PGP/;
  my ($bin, $in, $body, $b64) = ("", 0, 0, "");
  for my $l (split /\r?\n/, $d) {
    if ($l =~ /^-----BEGIN PGP/) { ($in, $body, $b64) = (1, 0, ""); next }
    next unless $in;
    if ($l =~ /^-----END PGP/) { $bin .= decode_base64($b64); $in = 0; next }
    if (!$body) { $body = 1 if $l eq ""; next }
    next if $l =~ /^=/;
    $b64 .= $l;
  }
  $bin
}
sub packets {
  my $d = shift; my @p; my $i = 0;
  while ($i < length $d) {
    my $h = ord substr($d, $i++, 1); die "malformed OpenPGP data\n" unless $h & 0x80;
    my ($tag, $len);
    if ($h & 0x40) {
      $tag = $h & 0x3f; my $o = ord substr($d, $i++, 1);
      if ($o < 192) { $len = $o }
      elsif ($o < 224) { $len = (($o - 192) << 8) + ord(substr($d, $i++, 1)) + 192 }
      elsif ($o == 255) { $len = unpack("N", substr($d, $i, 4)); $i += 4 }
      else { die "unsupported OpenPGP partial length\n" }
    } else {
      $tag = ($h >> 2) & 0x0f; my $lt = $h & 3;
      if ($lt == 0) { $len = ord substr($d, $i, 1); $i += 1 }
      elsif ($lt == 1) { $len = unpack("n", substr($d, $i, 2)); $i += 2 }
      elsif ($lt == 2) { $len = unpack("N", substr($d, $i, 4)); $i += 4 }
      else { die "unsupported OpenPGP indeterminate length\n" }
    }
    die "truncated OpenPGP packet\n" if $i + $len > length $d;
    push @p, [$tag, substr($d, $i, $len)]; $i += $len;
  }
  @p
}
sub mpi { my ($b, $p) = @_; my $n = int((unpack("n", substr($b, $p, 2)) + 7) / 8); (substr($b, $p + 2, $n), $p + 2 + $n) }
sub subpackets {
  my ($s, $d, $hashed) = @_; my $i = 0;
  while ($i < length $d) {
    my $o = ord substr($d, $i++, 1); my $l;
    if ($o < 192) { $l = $o } elsif ($o < 255) { $l = (($o - 192) << 8) + ord(substr($d, $i++, 1)) + 192 } else { $l = unpack("N", substr($d, $i, 4)); $i += 4 }
    die "malformed signature subpacket\n" if $l < 1 || $i + $l > length $d;
    my $t = ord(substr($d, $i, 1)) & 0x7f; my $v = substr($d, $i + 1, $l - 1); $i += $l;
    if ($t == 16) { $s->{issuer} = uc unpack("H*", $v) }
    elsif ($t == 33) { $s->{issuer} = uc substr(unpack("H*", substr($v, 1)), -16) }
    next unless $hashed;
    if ($t == 2) { $s->{created} = unpack("N", $v) }
    elsif ($t == 9) { $s->{keyexp} = unpack("N", $v) }
    elsif ($t == 27) { $s->{flags} = ord $v }
  }
}
sub signature {
  my $b = shift; die "unsupported OpenPGP signature version\n" unless ord(substr($b, 0, 1)) == 4;
  my ($type, $pk, $ha) = map { ord substr($b, $_, 1) } 1 .. 3;
  my $hl = unpack("n", substr($b, 4, 2)); my $ul = unpack("n", substr($b, 6 + $hl, 2));
  my %s = (type => $type, pk => $pk, ha => $ha, created => 0,
           trailer => substr($b, 0, 6 + $hl) . "\x04\xff" . pack("N", 6 + $hl));
  subpackets(\%s, substr($b, 6, $hl), 1); subpackets(\%s, substr($b, 8 + $hl, $ul), 0);
  ($s{sig}) = mpi($b, 8 + $hl + $ul + 2) if $pk == 1 || $pk == 3;
  \%s
}
my (%keys, $primary, $current);
sub key_packet {
  my ($b, $is_primary) = @_;
  return undef unless ord(substr($b, 0, 1)) == 4;
  my %k = (created => unpack("N", substr($b, 1, 4)), algo => ord(substr($b, 5, 1)), primary => $is_primary, revoked => 0, keyexp => 0, selfsig => -1);
  if ($k{algo} == 1 || $k{algo} == 3) { my $p; ($k{n}, $p) = mpi($b, 6); ($k{e}) = mpi($b, $p) }
  $k{id} = uc substr(unpack("H*", sha1("\x99" . pack("n", length $b) . $b)), -16);
  $keys{$k{id}} = \%k
}
for my $f ($asc, glob("$revdir/*")) {
  next unless -f $f;
  ($primary, $current) = (undef, undef);
  for my $pk (packets(dearmor(slurp($f)))) {
    my ($tag, $b) = @$pk;
    if ($tag == 6) { $primary = $current = key_packet($b, 1) }
    elsif ($tag == 14) { $current = key_packet($b, 0); $current->{parent} = $primary if $current }
    elsif ($tag == 2) {
      my $s = signature($b); my $t = $s->{type};
      if ($t == 0x20) {
        my $k = $primary || $keys{$s->{issuer} || ""};
        # A certificate for a key outside the bundle means the bundle is wrong; gpg would ignore it
        reject "Revocation certificate " . (split m{/}, $f)[-1] . " does not revoke a key in the BDOT key bundle\n" unless $k;
        $k->{revoked} = 1;
      }
      elsif ($t == 0x28) { $current->{revoked} = 1 if $current && !$current->{primary} }
      elsif (($t >= 0x10 && $t <= 0x13) || $t == 0x1f || $t == 0x18) {
        my $k = $t == 0x18 ? $current : $primary;
        next unless $k && $s->{created} >= $k->{selfsig};
        $k->{selfsig} = $s->{created}; $k->{keyexp} = $s->{keyexp} || 0; $k->{flags} = $s->{flags};
      }
    }
  }
}
my @sigs = grep { $_->[0] == 2 } packets(dearmor(slurp($sigfile)));
die "expected exactly one signature\n" unless @sigs == 1;
my $s = signature($sigs[0][1]);
my %hash = (8 => "sha256", 9 => "sha384", 10 => "sha512", 11 => "sha224");
die "unsupported signature type\n" unless $s->{type} == 0x00;
die "unsupported signature algorithm\n" unless $s->{pk} == 1 || $s->{pk} == 3;
die "unsupported signature digest\n" unless $hash{$s->{ha}};
die "signature has no creation time\n" unless $s->{created};
my $k = $keys{$s->{issuer} || ""} or die "signature was not made by a key in the bundle\n";
die "signing key is not an RSA key\n" unless $k->{n};
die "key is not allowed to sign\n" if defined $k->{flags} && !($k->{flags} & 0x02);
for my $c ($k, $k->{parent} || ()) {
  reject "signing key $c->{id} is revoked\n" if $c->{revoked};
  reject "signature predates signing key $c->{id}\n" if $s->{created} < $c->{created};
  reject "signing key $c->{id} had expired when the signature was made\n" if $c->{keyexp} && $s->{created} >= $c->{created} + $c->{keyexp};
}
sub der_len { my $l = shift; return chr $l if $l < 128; my $b = ""; while ($l) { $b = chr($l & 255) . $b; $l >>= 8 } chr(0x80 | length $b) . $b }
sub der { my ($t, $c) = @_; chr($t) . der_len(length $c) . $c }
sub der_int { my $x = shift; $x =~ s/^\x00+//; $x = "\x00$x" if ord($x) & 0x80; der(2, $x) }
my $spki = der(0x30, der(0x30, der(6, "\x2a\x86\x48\x86\xf7\x0d\x01\x01\x01") . der(5, "")) . der(3, "\x00" . der(0x30, der_int($k->{n}) . der_int($k->{e}))));
my $pem = encode_base64($spki, ""); $pem =~ s/(.{1,64})/$1\n/g;
my $sig = ("\x00" x (length($k->{n}) - length($s->{sig}))) . $s->{sig};
for (["key.pem", "-----BEGIN PUBLIC KEY-----\n$pem-----END PUBLIC KEY-----\n"], ["sig.raw", $sig], ["sig.trailer", $s->{trailer}]) {
  open(my $h, ">", "$out/$_->[0]") or die "cannot write $out/$_->[0]\n"; binmode $h; print {$h} $_->[1]; close $h;
}
print "$hash{$s->{ha}}\n";
'

# verify_package_aix checks the package against its detached signature and the bundled
# BDOT public key. It uses gpg when installed, and otherwise base Perl plus openssl, so
# hosts without the AIX Toolbox still verify. Both paths reject revoked keys and
# signatures made after the signing key expired; gpg --verify alone exits 0 for those.
verify_package_aix() {
  _sig_file="${package_out_file_path}.gpg.sig"
  if [ ! -f "$_sig_file" ]; then
    error "GPG signature file not found: $_sig_file"
    return 1
  fi

  if command -v gpg > /dev/null 2>&1; then
    gpg_import_bundle || return $?
    gpg_verify_file "$_sig_file" "$package_out_file_path" "Package" || return $?
    success "Package signature is valid, and its key was neither revoked nor expired when it signed"
    return 0
  fi

  # Without gpg, base Perl checks the keys and openssl checks the signature
  set +e
  _digest=$(perl -e "$PERL_PGP_PREPARE" "$GPG_DIR/bdot-public-gpg-key.asc" "$GPG_DIR/deb-revocations" "$_sig_file" "$GPG_DIR" 2> "$GPG_DIR/verify.err")
  _rc=$?
  set -e
  if [ $_rc -ne 0 ]; then
    error "Package signature is invalid: $(literal "$(cat "$GPG_DIR/verify.err")")"
    [ $_rc -eq 3 ] && return 3
    return 1
  fi
  if ! cat "$package_out_file_path" "$GPG_DIR/sig.trailer" | openssl dgst "-$_digest" -verify "$GPG_DIR/key.pem" -signature "$GPG_DIR/sig.raw" > /dev/null 2>&1; then
    error "Package was altered: its signature does not match its contents"
    return 3
  fi

  success "Package signature is valid, and its key was neither revoked nor expired when it signed"
  return 0
}

# install_aix installs the AIX tar.gz into $BDOT_CONFIG_HOME, with the same
# ownership the deb and rpm postinstall script sets: everything bdot, the updater root.
install_aix()
{
  # The updater manages the service at this fixed path, so the install has to match it
  if [ "$BDOT_CONFIG_HOME" != "/opt/observiq-otel-collector" ]; then
    error_exit "$LINENO" "BDOT_CONFIG_HOME can't be changed on AIX"
  fi

  # Create the user and group if they don't already exist. Group must be first.
  if ! lsgroup "$COLLECTOR_GROUP" > /dev/null 2>&1; then
    mkgroup "$COLLECTOR_GROUP" > /dev/null || error_exit "$LINENO" "Failed to create group '$COLLECTOR_GROUP'"
  fi

  # Create the install & storage directories. Before the user, so mkuser leaves the home's mode alone.
  mkdir -p "$BDOT_CONFIG_HOME/storage" "$BDOT_CONFIG_HOME/log"

  if ! lsuser "$COLLECTOR_USER" > /dev/null 2>&1; then
    # No shell: AIX only accepts shells listed in /etc/security/login.cfg, and /usr/bin/false
    # is not there by default. The mkuser.default shell always is, so keep it and create the account locked.
    mkuser pgrp="$COLLECTOR_GROUP" home="$BDOT_CONFIG_HOME" account_locked=true login=false rlogin=false "$COLLECTOR_USER" || error_exit "$LINENO" "Failed to create user '$COLLECTOR_USER'"
    # mkuser copies in a .profile, and the locked account never reads it
    rm -f "$BDOT_CONFIG_HOME/.profile"
    created_aix_user=true
  fi

  # Extract to a staging directory, so a running collector is stopped only after the package unpacks
  _aix_stage="$BDOT_CONFIG_HOME/.staging"
  rm -rf "$_aix_stage"
  mkdir -p "$_aix_stage" || error_exit "$LINENO" "Failed to create staging directory"
  if ! gunzip -c "$package_out_file_path" | tar -xf - -C "$_aix_stage"; then
    rm -rf "$_aix_stage"
    error_exit "$LINENO" "Failed to extract package"
  fi

  # Keep the user's configs on an upgrade, as the deb and rpm packages do
  for _cfg in config.yaml logging.yaml; do
    [ -f "$BDOT_CONFIG_HOME/$_cfg" ] && rm -f "$_aix_stage/$_cfg"
  done

  _aix_was_running=false
  if lssrc -s "$AIX_SRC_SUBSYSTEM" 2>/dev/null | grep -q ' active'; then
    _aix_was_running=true
    # AIX has no restart, so stop, replace the files, then start
    info "Stopping service for binary replacement..."
    stopsrc -g "$AIX_SRC_GROUP" > /dev/null || error_exit "$LINENO" "Failed to request service stop"
    _stop_wait=0
    while ! lssrc -s "$AIX_SRC_SUBSYSTEM" | grep -q inoperative; do
      _stop_wait=$((_stop_wait + 1))
      if [ "$_stop_wait" -ge 30 ]; then
        rm -rf "$_aix_stage"
        error_exit "$LINENO" "Service did not stop within 30 seconds"
      fi
      sleep 1
    done
    # Release any cached shared libraries
    slibclean > /dev/null 2>&1 || true
  fi

  if ! cp -Rf "$_aix_stage"/* "$BDOT_CONFIG_HOME/"; then
    rm -rf "$_aix_stage"
    error_exit "$LINENO" "Failed to copy the package into $BDOT_CONFIG_HOME"
  fi
  rm -rf "$_aix_stage"

  # The service environment lives outside the install directory, like the systemd unit
  mv "$BDOT_CONFIG_HOME/install/observiq-otel-collector.env" "$AIX_ENV_FILE" || error_exit "$LINENO" "Failed to install $AIX_ENV_FILE"
  chmod 0644 "$AIX_ENV_FILE"

  chown -R "$COLLECTOR_USER":"$COLLECTOR_GROUP" "$BDOT_CONFIG_HOME"
  chown root:system "$BDOT_CONFIG_HOME/updater"

  if [ "$_aix_was_running" = true ]; then
    info "Restarting service with new binaries..."
    startsrc -g "$AIX_SRC_GROUP" -e "$(cat "$AIX_ENV_FILE")" > /dev/null || error_exit "$LINENO" "Failed to start service"
  fi
}

unpack_package()
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

# create_manager_yml creates the manager.yml at the specified path, containing opamp information.
create_manager_yml()
{
  manager_yml_path="$1"
  if [ ! -f "$manager_yml_path" ]; then
    # Note here: We create the file and change permissions of the file here BEFORE writing info to it
    # We do this because the file may contain a secret key, so we want 0 window when the
    # file is readable by anyone other than the agent & root
    command printf '' >> "$manager_yml_path"

    chgrp "$COLLECTOR_GROUP" "$manager_yml_path"
    chown "$COLLECTOR_USER" "$manager_yml_path"
    chmod 0640 "$manager_yml_path"

    command printf 'endpoint: "%s"\n' "$OPAMP_ENDPOINT" > "$manager_yml_path"
    [ -n "$OPAMP_LABELS" ] && command printf 'labels: "%s"\n' "$OPAMP_LABELS" >> "$manager_yml_path"
    [ -n "$OPAMP_SECRET_KEY" ] && command printf 'secret_key: "%s"\n' "$OPAMP_SECRET_KEY" >> "$manager_yml_path"
  fi
}

# This will display the results of an installation
display_results()
{
    banner 'Information'
    increase_indent
    info "Agent Home:         $(fg_cyan "${BDOT_CONFIG_HOME}")$(reset)"
    info "Agent Config:       $(fg_cyan "${BDOT_CONFIG_HOME}/config.yaml")$(reset)"
    if [ "$SVC_PRE" = "systemctl" ]; then
      info "Start Command:      $(fg_cyan "sudo systemctl start observiq-otel-collector")$(reset)"
      info "Stop Command:       $(fg_cyan "sudo systemctl stop observiq-otel-collector")$(reset)"
      info "Status Command:     $(fg_cyan "sudo systemctl status observiq-otel-collector")$(reset)"
    elif [ "$SVC_PRE" = "mkssys" ]; then
      info "Start Command:      $(fg_cyan "sudo startsrc -g $AIX_SRC_GROUP -e \"\$(cat $AIX_ENV_FILE)\"")$(reset)"
      info "Stop Command:       $(fg_cyan "sudo stopsrc -g $AIX_SRC_GROUP")$(reset)"
      info "Status Command:     $(fg_cyan "sudo lssrc -s $AIX_SRC_SUBSYSTEM")$(reset)"
    else
      info "Start Command:      $(fg_cyan "sudo service observiq-otel-collector start")$(reset)"
      info "Stop Command:       $(fg_cyan "sudo service observiq-otel-collector stop")$(reset)"
      info "Status Command:     $(fg_cyan "sudo service observiq-otel-collector status")$(reset)"
    fi
    info "Logs Command:       $(fg_cyan "sudo tail -F ${BDOT_CONFIG_HOME}/log/collector.log")$(reset)"
    info "Uninstall Command:  $(fg_cyan "sudo sh -c \"\$(curl -fsSlL ${DOWNLOAD_BASE}/v${version}/install_unix.sh)\" install_unix.sh -r")$(reset)"
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

uninstall_package()
{
  case "$package_type" in
    deb)
      dpkg -r "observiq-otel-collector" > /dev/null 2>&1
      ;;
    mkssys)
      rm -rf "$BDOT_CONFIG_HOME"
      rm -f "$AIX_ENV_FILE"
      ;;
    rpm)
      rpm -e "observiq-otel-collector" > /dev/null 2>&1
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
  set_package_type
  banner "Uninstalling Bindplane Agent"
  increase_indent

  info "Checking permissions..."
  root_check
  succeeded

  if [ "$SVC_PRE" = "systemctl" ]; then
    info "Stopping service..."
    systemctl stop observiq-otel-collector > /dev/null || error_exit "$LINENO" "Failed to stop service"
    succeeded

    info "Disabling service..."
    systemctl disable observiq-otel-collector > /dev/null 2>&1 || error_exit "$LINENO" "Failed to disable service"
    succeeded
  elif [ "$SVC_PRE" = "mkssys" ]; then
    info "Stopping service..."
    stopsrc -g "$AIX_SRC_GROUP" > /dev/null 2>&1 || true
    _stop_wait=0
    while lssrc -s "$AIX_SRC_SUBSYSTEM" 2>/dev/null | grep -q ' active' && [ "$_stop_wait" -lt 30 ]; do
      _stop_wait=$((_stop_wait + 1))
      sleep 1
    done
    succeeded

    info "Removing service registration..."
    if lssrc -s "$AIX_SRC_SUBSYSTEM" > /dev/null 2>&1; then
      rmssys -s "$AIX_SRC_SUBSYSTEM" > /dev/null 2>&1
    fi
    if lsitab "$AIX_SRC_GROUP" > /dev/null 2>&1; then
      rmitab "$AIX_SRC_GROUP" > /dev/null 2>&1
    fi
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

main()
{
  # We do these checks before we process arguments, because
  # some of these options bail early, and we'd like to be sure that those commands
  # (e.g. uninstall) can run

  bindplane_banner
  check_prereqs

  if [ $# -ge 1 ]; then
    while [ -n "$1" ]; do
      case "$1" in
        -q|--quiet)
          non_interactive="true" ; shift 1 ;;
        -v|--version)
          version=$2 ; shift 2 ;;
        -l|--url)
          url=$2 ; shift 2 ;;
        -gl|--gpg-tar-url)
          gpg_tar_url=$2 ; shift 2 ;;
        -f|--file)
          package_path=$2 ; shift 2 ;;
        -gf|--gpg-tar-file)
          gpg_tar_path=$2 ; shift 2 ;;
        -x|--proxy)
          proxy=$2 ; shift 2 ;;
        -U|--proxy-user)
          proxy_user=$2 ; shift 2 ;;
        -P|--proxy-password)
          proxy_password=$2 ; shift 2 ;;
        -e|--endpoint)
          opamp_endpoint=$2 ; shift 2 ;;
        -k|--labels)
          opamp_labels=$2 ; shift 2 ;;
        -s|--secret-key)
          opamp_secret_key=$2 ; shift 2 ;;
        -c|--check-bp-url)
          check_bp_url="true" ; shift 1 ;;
        -b|--base-url)
          base_url=$2 ; shift 2 ;;
        --no-gpg-check)
          skip_gpg_check="true" ; shift 1 ;;
        --require-gpg-check)
          require_gpg_check="true" ; shift 1 ;;
        -r|--uninstall)
          uninstall
          exit 0
          ;;
        -h|--help)
          usage
          exit 0
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

  if [ -z "$version" ] ; then
    # shellcheck disable=SC2153
    version=$COLLECTOR_VERSION
  fi

  if [ -z "$version" ] ; then
    version=$(latest_version)
  fi

  if [ -z "$version" ] ; then
    error_exit "$LINENO" "Could not determine version to install"
  fi

  validate_version
  interactive_check
  connection_check
  offline_check
  setup_installation
  verification_check
  install_package
  # The install succeeded, so keep any user install_aix created
  created_aix_user=false
  display_results
}

main "$@"
