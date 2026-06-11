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

# Idempotent migration script to enable hardened non-root mode for
# an existing observIQ OpenTelemetry Collector installation.
#
# This script:
#   1. Sets BDOT_HARDENED=true in the package override file
#   2. Changes ownership of the install directory to bdot:bdot
#   3. Updates the systemd unit or SysV init script to use the bdot user
#   4. Reloads the init system and restarts the service
#
# Usage: sudo ./bdot-harden.sh
#
# This script is safe to run multiple times.

set -e

# Must run as root
if [ "$(id -u)" -ne 0 ]; then
  echo "ERROR: This script must be run as root (or with sudo)." >&2
  exit 1
fi

# Load existing overrides
[ -f /etc/default/observiq-otel-collector ] && . /etc/default/observiq-otel-collector
[ -f /etc/sysconfig/observiq-otel-collector ] && . /etc/sysconfig/observiq-otel-collector

: "${BDOT_CONFIG_HOME:=/opt/observiq-otel-collector}"
: "${BDOT_USER:=bdot}"
: "${BDOT_GROUP:=bdot}"

# Determine the override file location
if [ -d /etc/sysconfig ]; then
  OVERRIDE_FILE="/etc/sysconfig/observiq-otel-collector"
elif [ -d /etc/default ]; then
  OVERRIDE_FILE="/etc/default/observiq-otel-collector"
else
  OVERRIDE_FILE="/etc/default/observiq-otel-collector"
  mkdir -p /etc/default
fi

# Step 1: Enable BDOT_HARDENED in the override file
if [ -f "$OVERRIDE_FILE" ] && grep -q "^BDOT_HARDENED=" "$OVERRIDE_FILE"; then
  sed -i 's/^BDOT_HARDENED=.*/BDOT_HARDENED=true/' "$OVERRIDE_FILE"
  echo "Updated BDOT_HARDENED=true in $OVERRIDE_FILE"
else
  echo "BDOT_HARDENED=true" >> "$OVERRIDE_FILE"
  echo "Added BDOT_HARDENED=true to $OVERRIDE_FILE"
fi

# Step 2: Change ownership of the install directory
echo "Changing ownership of ${BDOT_CONFIG_HOME} to ${BDOT_USER}:${BDOT_GROUP}"
chown -R "${BDOT_USER}:${BDOT_GROUP}" "${BDOT_CONFIG_HOME}"

# Step 3: Update service files
UNIT_FILE="/usr/lib/systemd/system/observiq-otel-collector.service"
INITD_FILE="/etc/init.d/observiq-otel-collector"

if [ -f "$UNIT_FILE" ]; then
  # Update systemd unit User= line
  if grep -q "^User=root" "$UNIT_FILE"; then
    sed -i "s/^User=root/User=${BDOT_USER}/" "$UNIT_FILE"
    echo "Updated User= in $UNIT_FILE to ${BDOT_USER}"
  elif grep -q "^User=${BDOT_USER}" "$UNIT_FILE"; then
    echo "User= already set to ${BDOT_USER} in $UNIT_FILE"
  else
    echo "WARNING: Unexpected User= value in $UNIT_FILE, updating to ${BDOT_USER}"
    sed -i "s/^User=.*/User=${BDOT_USER}/" "$UNIT_FILE"
  fi
fi

if [ -f "$INITD_FILE" ]; then
  # Update SysV init script RUNAS_USER variable
  if grep -q "^RUNAS_USER=" "$INITD_FILE"; then
    sed -i "s/^RUNAS_USER=.*/RUNAS_USER=${BDOT_USER}/" "$INITD_FILE"
    echo "Updated RUNAS_USER in $INITD_FILE to ${BDOT_USER}"
  else
    echo "WARNING: RUNAS_USER not found in $INITD_FILE, init script may need manual update"
  fi
fi

# Step 4: Clean up legacy BDOT_UNPRIVILEGED drop-in
LEGACY_OVERRIDE="/etc/systemd/system/observiq-otel-collector.service.d/10-package-customizations-username.conf"
if [ -f "$LEGACY_OVERRIDE" ]; then
  rm -f "$LEGACY_OVERRIDE"
  echo "Removed legacy unprivileged override at $LEGACY_OVERRIDE"
fi

# Step 5: Reload and restart
if command -v systemctl > /dev/null 2>&1 && [ -f "$UNIT_FILE" ]; then
  systemctl daemon-reload
  echo "Reloaded systemd daemon"

  if systemctl is-active --quiet observiq-otel-collector 2>/dev/null; then
    systemctl restart observiq-otel-collector
    echo "Restarted observiq-otel-collector service"
  else
    echo "Service not running, skipping restart"
  fi
elif command -v service > /dev/null 2>&1 && [ -f "$INITD_FILE" ]; then
  service observiq-otel-collector restart 2>/dev/null && \
    echo "Restarted observiq-otel-collector service via init.d" || \
    echo "Service not running or restart failed, skipping"
else
  echo "WARNING: Could not detect init system, please restart the service manually"
fi

echo ""
echo "Migration complete. The collector is now configured to run as ${BDOT_USER}."
if command -v systemctl > /dev/null 2>&1 && [ -f "$UNIT_FILE" ]; then
  echo "To verify: systemctl show -p User observiq-otel-collector"
else
  echo "To verify: check the running process user with 'ps aux | grep observiq-otel-collector'"
fi
