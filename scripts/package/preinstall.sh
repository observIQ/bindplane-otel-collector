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

# Read's optional package overrides. Users should deploy the override
# file before installing BDOT for the first time. The override should
# not be modified unless uninstalling and re-installing.
[ -f /etc/default/observiq-otel-collector ] && . /etc/default/observiq-otel-collector
[ -f /etc/sysconfig/observiq-otel-collector ] && . /etc/sysconfig/observiq-otel-collector

: "${BDOT_SKIP_RUNTIME_USER_CREATION:=false}"

username="bdot"
legacy_username="observiq-otel-collector"
service_name="observiq-otel-collector"

# Install creates the user and group for the collector
# service. This function is idempotent and safe to call
# multiple times.
install() {
    if [ "$BDOT_SKIP_RUNTIME_USER_CREATION" = "true" ]; then
        echo "BDOT_SKIP_RUNTIME_USER_CREATION is set to true, skipping user and group creation"
    else
        echo "Creating ${username} user and group"
        install_user
    fi
}

install_user() {
    # Return early without output if the user and group already exist.
    # This will help avoid confusion with the output in migrate_user().
    if id "$username" > /dev/null 2>&1 && getent group "$username" > /dev/null 2>&1; then
        return
    fi

    if getent group "$username" > /dev/null 2>&1; then
        echo "Group ${username} already exists."
    else
        groupadd "$username"
    fi

    if id "$username" > /dev/null 2>&1; then
        echo "User ${username} already exists"
        exit 0
    else
        useradd --shell /sbin/nologin --system "$username" -g "$username"
    fi
}

# migrate_user migrates the legacy user to the new username.
migrate_user() {
    _migrate_user
    _migrate_group

    echo "User migration complete."
}

_migrate_user() {
    if ! id "$legacy_username" > /dev/null 2>&1; then
        echo "Skipping user migration: Legacy user ${legacy_username} does not exist."
        return
    fi

    if id "$username" > /dev/null 2>&1; then
        echo "Skipping user migration: User ${username} already exists."
        return
    fi

    # Check if the service is running
    if systemctl is-active --quiet "$service_name"; then
        echo "Service $service_name is running"

        # Check if the service is running as the legacy user
        service_user=$(ps -o user= -p "$(systemctl show -p MainPID --value "$service_name")" 2>/dev/null || echo "")
        echo "Service is running as user: $service_user"
        if [ "$service_user" = "$legacy_username" ]; then
            echo "Service is running as user ${legacy_username}, stopping service before user migration"
            systemctl stop "$service_name"
            service_was_running=true
        else
            echo "Service is running but not as user ${legacy_username}, proceeding with user migration"
            service_was_running=false
        fi
    else
        echo "Service $service_name is not running"
        service_was_running=false
    fi

    echo "Renaming user ${legacy_username} to ${username}"
    usermod -l "$username" "$legacy_username"

    # Restart the service if it was running before
    if [ "$service_was_running" = "true" ]; then
        echo "Restarting service $service_name"
        systemctl start "$service_name"
    fi
}

_migrate_group() {
    if ! getent group "$legacy_username" > /dev/null 2>&1; then
        echo "Skipping group migration: Legacy group ${legacy_username} does not exist."
        return
    fi

    if getent group "$username" > /dev/null 2>&1; then
        echo "Skipping group migration: Group ${username} already exists."
        return
    fi

    echo "Renaming group ${legacy_username} to ${username}"

    # TODO(jsirianni /  Dylan-M): Groupmod will not work on AIX
    # Discussion: https://github.com/observIQ/bindplane-otel-collector/pull/2436
    groupmod -n "$username" "$legacy_username"
}

migrate_user
install
