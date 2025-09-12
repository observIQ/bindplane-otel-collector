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
[ -f /etc/default/bindplane-otel-collector ] && . /etc/default/bindplane-otel-collector
[ -f /etc/sysconfig/bindplane-otel-collector ] && . /etc/sysconfig/bindplane-otel-collector

# Configurable username and group for BDOT
: "${BDOT_USER:=bdot}"
: "${BDOT_GROUP:=bdot}"

install() {
    username="${BDOT_USER}"
    groupname="${BDOT_GROUP}"

    if getent group "$groupname" >/dev/null 2>&1; then
        echo "Group ${groupname} already exists."
    else
        groupadd "$groupname"
    fi

    if id "$username" >/dev/null 2>&1; then
        echo "User ${username} already exists"
        exit 0
    else
        useradd --shell /sbin/nologin --system "$username" -g "$groupname"
    fi
}

# Upgrade should perform the same steps as install
upgrade() {
    install
}

action="$1"

case "$action" in
"0" | "install")
    install
    ;;
"1" | "upgrade")
    upgrade
    ;;
*)
    install
    ;;
esac
