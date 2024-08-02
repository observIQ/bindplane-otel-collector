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

remove() {
  rm -f /usr/lib/systemd/system/bindplane-otel-collector.service || {
    printf 'failed to remove /usr/lib/systemd/system/bindplane-otel-collector.service'
  }

  rm -f /etc/sysconfig/bindplane-otel-collector || {
    printf 'failed to remove /etc/sysconfig/bindplane-otel-collector'
  }

  rm -f /etc/init.d/bindplane-otel-collector || {
    printf 'failed to remove /etc/init.d/bindplane-otel-collector'
  }

  # remove the entire folder
  # pkg manager will remove most files but this will delete the remaining
  rm -rf /opt/bindplane-otel-collector || {
    printf 'failed to remove /opt/bindplane-otel-collector'
  }
}

upgrade() {
  return
}

action="$1"

case "$action" in
"0" | "remove")
  remove
  ;;
"1" | "upgrade")
  upgrade
  ;;
*)
  remove
  ;;
esac
