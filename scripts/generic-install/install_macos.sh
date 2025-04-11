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
INSTALL_DIR="/opt/$DISTRIBUTION"
SUPERVISOR_YML_PATH="$INSTALL_DIR/supervisor-config.yaml"
PREREQS="curl printf sed"
DISTRIBUTION=""
REPOSITORY_URL=""

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
    # Assuming the plist file is included in the package and copied to INSTALL_DIR
    manage_service_plist_source="$INSTALL_DIR/service/com.$DISTRIBUTION.plist"

    if [ ! -f "$manage_service_plist_source" ]; then
        echo "Error: plist file not found in package"
        exit 1
    fi

    # Copy the plist to LaunchDaemons
    cp "$manage_service_plist_source" "/Library/LaunchDaemons/"
    chown root:wheel "/Library/LaunchDaemons/com.$DISTRIBUTION.plist"
    chmod 644 "/Library/LaunchDaemons/com.$DISTRIBUTION.plist"

    # Load/reload the service
    launchctl unload "/Library/LaunchDaemons/com.$DISTRIBUTION.plist" 2>/dev/null || true
    launchctl load "/Library/LaunchDaemons/com.$DISTRIBUTION.plist"

    echo "Service has been configured and started"
}

set_permissions() {
    echo "Setting permissions..."
    chown -R "$DISTRIBUTION:$DISTRIBUTION" "$INSTALL_DIR"
}

create_supervisor_config() {
    echo "Creating supervisor config..."
    if [ -z "$OPAMP_ENDPOINT" ]; then
        OPAMP_ENDPOINT="ws://localhost:3001/v1/opamp"
        echo "No OpAMP endpoint specified, using default: $OPAMP_ENDPOINT"
    fi

    # Create empty file and set permissions before writing sensitive data
    command printf '' >"$SUPERVISOR_YML_PATH"
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
    arm64)
        os_arch="arm64"
        ;;
    x86_64)
        os_arch="amd64"
        ;;
    *)
        echo "Unsupported macOS architecture: $os_arch"
        exit 1
        ;;
    esac
}

# latest_version gets the tag of the latest release, without the v prefix.
latest_version() {
    curl -sSL -H"Accept: application/vnd.github.v3+json" "$repository_url/releases/latest" |
        grep "\"tag_name\"" |
        sed -E 's/ *"tag_name": "v([0-9]+\.[0-9]+\.[0-9+])",/\1/'
}

verify_version() {
    if [ -z "$version" ]; then
        version=$(latest_version)
    fi
}

download_and_install() {
    if [ -n "$local_file" ]; then
        echo "Installing from local file: $local_file"
        mkdir -p "$INSTALL_DIR"
        tar xzf "$local_file" -C "$INSTALL_DIR"
    else
        verify_version
        set_os_arch
        download_and_install_url="${repository_url}/releases/download/v${version}/${DISTRIBUTION}_v${version}_darwin_${os_arch}.tar.gz"
        echo "Downloading: $download_and_install_url"
        mkdir -p "$INSTALL_DIR"
        curl -L "$download_and_install_url" | tar xz -C "$INSTALL_DIR"
    fi
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
    download_and_install
    create_supervisor_config
    set_permissions
    manage_service

    echo "Installation complete!"
    echo "Installation directory: $INSTALL_DIR"
    echo "Supervisor config: $SUPERVISOR_YML_PATH"
}

uninstall() {
    echo "Uninstalling $DISTRIBUTION..."
    # Unload and remove launchd service
    launchctl unload "/Library/LaunchDaemons/com.$DISTRIBUTION.plist" 2>/dev/null || true
    rm -f "/Library/LaunchDaemons/com.$DISTRIBUTION.plist"

    # Remove installation directory
    rm -rf "$INSTALL_DIR"
    echo "Uninstallation complete"
}

get_repository_url() {
    if [ -n "$url" ]; then
        repository_url="$url"
    elif [ -n "$REPOSITORY_URL" ]; then
        repository_url="$REPOSITORY_URL"
    else
        echo "Error: No repository URL specified. Please either:"
        echo "  1. Use the -u/--url option"
        echo "  2. Set the REPOSITORY_URL constant in the script"
        exit 1
    fi
    echo "Using repository URL: $repository_url"
}

verify_distribution() {
    # Check if distribution is provided
    if [ -z "$DISTRIBUTION" ]; then
        echo "Error: Distribution name is required"
        echo "   1. Use the -d/--distribution option"
        echo "   2. Set the DISTRIBUTION constant in the script"
        usage
    fi
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
