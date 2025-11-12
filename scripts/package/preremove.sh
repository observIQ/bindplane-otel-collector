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

service_name="observiq-otel-collector"

# Check if this is an uninstall or an upgrade
# RPM: $1 is the number of packages remaining that provide this package
#      If $1 == 0, it's a complete uninstall; if $1 > 0, it's an upgrade
# DEB: $1 is "remove", "purge", or "upgrade"
#      If $1 is "remove" or "purge", it's an uninstall; if "upgrade", it's an upgrade
is_uninstall() {
    # Check for DEB format first (string arguments)
    case "$1" in
        remove|purge)
            return 0  # uninstall
            ;;
        upgrade)
            return 1  # upgrade
            ;;
    esac
    
    # Check for RPM format (numeric argument)
    # If $1 is numeric and equals 0, it's an uninstall
    if [ -n "$1" ] && [ "$1" -eq 0 ] 2>/dev/null; then
        return 0  # uninstall
    fi
    
    # Default to upgrade if we can't determine
    return 1  # upgrade
}

# Only stop and disable service on uninstall, not on upgrade
if is_uninstall "$1"; then
    if command -v systemctl > /dev/null 2>&1; then
        if systemctl is-active --quiet "$service_name" 2>/dev/null; then
            echo "Stopping service: $service_name"
            systemctl stop "$service_name" || true
        fi
        
        if systemctl is-enabled --quiet "$service_name" 2>/dev/null; then
            echo "Disabling service: $service_name"
            systemctl disable "$service_name" || true
        fi
        
        echo "Reloading systemd daemon"
        systemctl daemon-reload || true
    fi
else
    echo "Upgrade detected, skipping service stop and disable"
fi

