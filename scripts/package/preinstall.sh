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

username="bdot"
legacy_username="observiq-otel-collector"

# Install creates the user and group for the collector
# service. This function is idempotent and safe to call
# multiple times.
install() {
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

    echo "Renaming user ${legacy_username} to ${username}"
    usermod -l "$username" "$legacy_username"
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
