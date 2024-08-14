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

manage_systemd_service() {
  # Ensure sysv script isn't present, and if it is remove it
  if [ -f /etc/init.d/observiq-otel-collector ]; then
    rm -f /etc/init.d/observiq-otel-collector
  fi

  systemctl daemon-reload

  echo "configured systemd service"

  cat <<EOF

The "observiq-otel-collector" service has been configured!

The collector's config file can be found here: 
  /opt/observiq-otel-collector/supervisor_storage/effective.yaml

To view logs from the collector, run:
  sudo tail -F /opt/observiq-otel-collector/supervisor_storage/agent.log

For more information on configuring the collector, see the docs:
  https://github.com/observIQ/bindplane-otel-collector/tree/main#observiq-opentelemetry-collector

To stop the observiq-otel-collector service, run:
  sudo systemctl stop observiq-otel-collector

To start the observiq-otel-collector service, run:
  sudo systemctl start observiq-otel-collector

To restart the observiq-otel-collector service, run:
  sudo systemctl restart observiq-otel-collector

To enable the service on startup, run:
  sudo systemctl enable observiq-otel-collector

If you have any other questions please contact us at support@observiq.com
EOF
}

manage_sysv_service() {
  chmod 755 /etc/init.d/observiq-otel-collector
  chmod 644 /etc/sysconfig/observiq-otel-collector
  echo "configured sysv service"
}

init_type() {
  # Determine if we need service or systemctl for prereqs
  if command -v systemctl >/dev/null 2>&1; then
    command printf "systemd"
    return
  elif command -v service >/dev/null 2>&1; then
    command printf "service"
    return
  fi

  command printf "unknown"
  return
}

manage_service() {
  service_type="$(init_type)"
  case "$service_type" in
  systemd)
    manage_systemd_service
    ;;
  service)
    manage_sysv_service
    ;;
  *)
    echo "could not detect init system, skipping service configuration"
    ;;
  esac
}

finish_permissions() {
  # Goreleaser does not set plugin file permissions, so do them here
  # We also change the owner of the binary to observiq-otel-collector
  chown -R observiq-otel-collector:observiq-otel-collector \
    /opt/observiq-otel-collector/observiq-otel-collector \
    /opt/observiq-otel-collector/opampsupervisor \
    /opt/observiq-otel-collector/plugins/*

  chmod 0640 /opt/observiq-otel-collector/plugins/*
}

install() {
  finish_permissions
  manage_service
}

# upgrade should perform the same actions as install
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
