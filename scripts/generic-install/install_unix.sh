#!/bin/sh
# Copyright observIQ, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

# Constants
DISTRIBUTION=""
REPOSITORY_URL=""
PREREQS="curl printf sed"

usage() {
    echo "Usage: $0 [options]"
    echo ""
    echo "Required arguments:"
    echo "  -d, --distribution NAME  Name of the distribution to install"
    echo ""
    echo "Optional arguments:"
    echo "  -f, --file PATH         Path to local package file (if provided, --version and --url are ignored)"
    echo "  -u, --url URL           GitHub repository URL (e.g., 'https://github.com/org/repo')"
    echo "  -v, --version VERSION   Specify version to install (default: latest)"
    echo "  -e, --endpoint URL      OpAMP endpoint (default: ws://localhost:3001/v1/opamp)"
    echo "  -s, --secret-key KEY    OpAMP secret key"
    echo "  -l, --labels LABELS     Comma-separated list of labels (e.g., 'env=prod,region=us-west')"
    echo "  -r, --uninstall         Uninstall the package"
    echo "  -h, --help              Show this help message"
    exit 1
}

manage_service() {
    echo "Managing service state..."
    if ! command -v systemctl >/dev/null; then
        echo "Warning: systemd not found, service not enabled"
        echo "You will need to manually start the service"
        return
    fi

    # Reload systemd to pick up any changes
    systemctl daemon-reload

    # Check if service is already enabled
    if ! systemctl is-enabled "$DISTRIBUTION" >/dev/null 2>&1; then
        echo "Enabling $DISTRIBUTION service..."
        systemctl enable "$DISTRIBUTION"
    fi

    # Check if service is already running
    if systemctl is-active "$DISTRIBUTION" >/dev/null 2>&1; then
        echo "Restarting $DISTRIBUTION service..."
        systemctl restart "$DISTRIBUTION"
    else
        echo "Starting $DISTRIBUTION service..."
        systemctl start "$DISTRIBUTION"
    fi

    # Verify service status
    if systemctl is-active "$DISTRIBUTION" >/dev/null 2>&1; then
        echo "Service is running"
    else
        echo "Warning: Service failed to start. Check status with: systemctl status $DISTRIBUTION"
        exit 1
    fi
}

create_supervisor_config() {
    echo "Creating supervisor config..."
    if [ -z "$OPAMP_ENDPOINT" ]; then
        OPAMP_ENDPOINT="ws://localhost:3001/v1/opamp"
        echo "No OpAMP endpoint specified, using default: $OPAMP_ENDPOINT"
    fi

    # Create empty file and set permissions before writing sensitive data
    command printf '' >"$SUPERVISOR_YML_PATH"
    chown "$DISTRIBUTION:$DISTRIBUTION" "$SUPERVISOR_YML_PATH"
    chmod 0600 "$SUPERVISOR_YML_PATH"

    # Write configuration line by line
    command printf 'server:\n' >"$SUPERVISOR_YML_PATH"
    command printf '  endpoint: "%s"\n' "$OPAMP_ENDPOINT" >>"$SUPERVISOR_YML_PATH"

    if [ -n "$OPAMP_SECRET_KEY" ]; then
        command printf '  headers:\n' >>"$SUPERVISOR_YML_PATH"
        command printf '    Authorization: "Secret-Key %s"\n' "$OPAMP_SECRET_KEY" >>"$SUPERVISOR_YML_PATH"
    fi

    command printf '  tls:\n' >>"$SUPERVISOR_YML_PATH"
    command printf '    insecure: true\n' >>"$SUPERVISOR_YML_PATH"
    command printf '    insecure_skip_verify: true\n' >>"$SUPERVISOR_YML_PATH"
    command printf 'capabilities:\n' >>"$SUPERVISOR_YML_PATH"
    command printf '  accepts_remote_config: true\n' >>"$SUPERVISOR_YML_PATH"
    command printf '  reports_remote_config: true\n' >>"$SUPERVISOR_YML_PATH"
    command printf 'agent:\n' >>"$SUPERVISOR_YML_PATH"
    command printf '  executable: "%s"\n' "$INSTALL_DIR/$DISTRIBUTION" >>"$SUPERVISOR_YML_PATH"
    command printf '  config_apply_timeout: 30s\n' >>"$SUPERVISOR_YML_PATH"
    command printf '  bootstrap_timeout: 5s\n' >>"$SUPERVISOR_YML_PATH"

    if [ -n "$OPAMP_LABELS" ]; then
        command printf '  description:\n' >>"$SUPERVISOR_YML_PATH"
        command printf '    non_identifying_attributes:\n' >>"$SUPERVISOR_YML_PATH"
        command printf '      service.labels: "%s"\n' "$OPAMP_LABELS" >>"$SUPERVISOR_YML_PATH"
    fi

    command printf 'storage:\n' >>"$SUPERVISOR_YML_PATH"
    command printf '  directory: "%s"\n' "$INSTALL_DIR/supervisor_storage" >>"$SUPERVISOR_YML_PATH"
    command printf 'telemetry:\n' >>"$SUPERVISOR_YML_PATH"
    command printf '  logs:\n' >>"$SUPERVISOR_YML_PATH"
    command printf '    level: 0\n' >>"$SUPERVISOR_YML_PATH"
    command printf '    output_paths: ["%s"]\n' "$INSTALL_DIR/supervisor.log" >>"$SUPERVISOR_YML_PATH"
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
        echo "Unsupported os arch: $os_arch"
        exit 1
        ;;
    esac
}

# latest_version gets the tag of the latest release, without the v prefix.
latest_version() {
    lv=$(curl -Ls -o /dev/null -w '%{url_effective}' "$repository_url/releases/latest")
    lv=${lv##*/} # Remove everything before the last '/'
    lv=${lv#v}   # Remove the 'v' prefix if it exists
    echo "$lv"
}

verify_version() {
    if [ -z "$version" ]; then
        version=$(latest_version)
    fi
}

download_and_install() {
    if [ -n "$local_file" ]; then
        echo "Installing from local file: $local_file"

        case "$pkg_type" in
        tar.gz)
            mkdir -p "$INSTALL_DIR"
            tar xz -C "$INSTALL_DIR"
            ;;
        deb)
            dpkg -i "$local_file"
            ;;
        rpm)
            rpm -U "$local_file"
            ;;
        esac
    else
        verify_version
        set_os_arch
        download_and_install_url="${repository_url}/releases/download/v${version}/${DISTRIBUTION}_v${version}_linux_${os_arch}.${pkg_type}"
        echo "Downloading: $download_and_install_url"

        case "$pkg_type" in
        tar.gz)
            mkdir -p "$INSTALL_DIR"
            curl -L "$download_and_install_url" | tar xz -C "$INSTALL_DIR"
            ;;
        deb)
            deb_tmp_file="/tmp/${DISTRIBUTION}_${version}.deb"
            curl -L "$download_and_install_url" -o "$deb_tmp_file"
            dpkg -i "$deb_tmp_file"
            rm -f "$deb_tmp_file"
            ;;
        rpm)
            rpm_tmp_file="/tmp/${DISTRIBUTION}_${version}.rpm"
            curl -L "$download_and_install_url" -o "$rpm_tmp_file"
            rpm -U "$rpm_tmp_file"
            rm -f "$rpm_tmp_file"
            ;;
        esac
    fi
}

detect_package_type() {
    if command -v dpkg >/dev/null 2>&1; then
        pkg_type="deb"
    elif command -v rpm >/dev/null 2>&1; then
        pkg_type="rpm"
    else
        pkg_type="tar.gz"
    fi
    echo "Auto-detected package type: $pkg_type"
}

dependencies_check() {
    FAILED_PREREQS=''
    for prerequisite in $PREREQS; do
        if command -v "$prerequisite" >/dev/null; then
            continue
        else
            if [ -z "$FAILED_PREREQS" ]; then
                FAILED_PREREQS="$prerequisite"
            else
                FAILED_PREREQS="$FAILED_PREREQS, $prerequisite"
            fi
        fi
    done

    if [ -n "$FAILED_PREREQS" ]; then
        echo "The following dependencies are required by this script: [$FAILED_PREREQS]"
        exit 1
    fi
}

install() {
    dependencies_check
    detect_package_type
    download_and_install
    create_supervisor_config
    manage_service

    echo "Installation complete!"
    echo "Installation directory: $INSTALL_DIR"
    echo "Supervisor config: $SUPERVISOR_YML_PATH"
}

uninstall() {
    echo "Uninstalling $DISTRIBUTION..."
    if command -v systemctl >/dev/null; then
        systemctl stop "$DISTRIBUTION" >/dev/null 2>&1 || true
        systemctl disable "$DISTRIBUTION" >/dev/null 2>&1 || true
    fi

    # Remove package if it was installed via package manager
    if command -v dpkg >/dev/null 2>&1; then
        dpkg -r "$DISTRIBUTION" >/dev/null 2>&1 || true
    elif command -v rpm >/dev/null 2>&1; then
        rpm -e "$DISTRIBUTION" >/dev/null 2>&1 || true
    fi

    rm -rf "$INSTALL_DIR"
    echo "Uninstallation complete"
}

verify_distribution() {
    # Check if distribution is provided
    if [ -z "$DISTRIBUTION" ]; then
        echo "Error: Distribution name is required"
        echo "   1. Use the -d/--distribution option"
        echo "   2. Set the DISTRIBUTION constant in the script"
        usage
    fi

    # Set constants dependent on distribution name
    INSTALL_DIR="/opt/$DISTRIBUTION"
    SUPERVISOR_YML_PATH="$INSTALL_DIR/supervisor_config.yaml"
}

get_repository_url() {
    if [ -n "$url" ]; then
        repository_url="$url"
    elif [ -n "$REPOSITORY_URL" ]; then
        repository_url="$REPOSITORY_URL"
    else
        echo "Error: No repository URL specified. Please either:"
        echo "  1. Use the -b/--base-url option"
        echo "  2. Set the REPOSITORY_URL constant in the script"
        exit 1
    fi
    echo "Using repository URL: $repository_url"
}

parse_args() {
    # Set default version
    version=""
    local_file=""

    while [ -n "$1" ]; do
        case "$1" in
        -d | --distribution)
            DISTRIBUTION=$2
            shift 2
            ;;
        -f | --file)
            local_file=$2
            if [ ! -f "$local_file" ]; then
                echo "Error: File not found: $local_file"
                exit 1
            fi
            shift 2
            ;;
        -u | --url)
            url=$2
            shift 2
            ;;
        -v | --version)
            version=$2
            shift 2
            ;;
        -e | --endpoint)
            OPAMP_ENDPOINT=$2
            shift 2
            ;;
        -s | --secret-key)
            OPAMP_SECRET_KEY=$2
            shift 2
            ;;
        -l | --labels)
            OPAMP_LABELS=$2
            shift 2
            ;;
        -r | --uninstall)
            verify_distribution
            uninstall
            exit 0
            ;;
        -h | --help)
            usage
            ;;
        *)
            echo "Invalid argument: $1"
            usage
            ;;
        esac
    done

    verify_distribution

    # Only get repository URL if we're not using a local file
    if [ -z "$local_file" ]; then
        get_repository_url
    fi
}

check_root() {
    if [ "$(id -u)" != "0" ]; then
        echo "This script must be run as root or using sudo"
        exit 1
    fi
}

main() {
    check_root
    parse_args "$@"
    install
}

main "$@"
