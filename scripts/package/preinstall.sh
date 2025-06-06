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

username="observiq-otel-collector"

install() {
    if [ "$BDOT_SKIP_RUNTIME_USER_CREATION" = "true" ]; then
        echo "BDOT_SKIP_RUNTIME_USER_CREATION is set to true, skipping user and group creation"
    else
        echo "Creating ${username} user and group"
        install_user
    fi
}

install_user() {
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

install
