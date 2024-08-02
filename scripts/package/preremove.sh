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

handle_systemctl() {
  systemctl stop bindplane-otel-collector >/dev/null || {
    printf 'failed to stop service'
  }
  systemctl disable bindplane-otel-collector >/dev/null 2>&1 || {
    printf 'failed to disable service'
  }
}

handle_service() {
  service bindplane-otel-collector stop || {
    printf 'failed to stop service'
  }
  chkconfig bindplane-otel-collector on >/dev/null 2>&1 || {
    printf 'failed to disable service'
  }
}

remove() {
  # Determine if we need service or systemctl
  if command -v systemctl >/dev/null 2>&1; then
    handle_systemctl
  elif command -v service >/dev/null 2>&1; then
    handle_service
  fi
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
