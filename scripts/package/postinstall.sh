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

: "${BDOT_CONFIG_HOME:=/opt/bindplane-otel-collector}"

install() {
  mkdir -p "${BDOT_CONFIG_HOME}"
  chmod 0755 "${BDOT_CONFIG_HOME}"
  chown bindplane-otel-collector:bindplane-otel-collector "${BDOT_CONFIG_HOME}"
  rm -f "${BDOT_CONFIG_HOME}/bindplane-otel-collector" || true
  cp -r --preserve \
    /usr/share/bindplane-otel-collector/stage/bindplane-otel-collector/* \
    "${BDOT_CONFIG_HOME}"

  rm -rf /usr/share/bindplane-otel-collector
}

install_service() {
  if command -v systemctl >/dev/null 2>&1; then
    install_systemd_service
  else
    install_initd_service
  fi
}

install_systemd_service() {
  config_file="/usr/lib/systemd/system/bindplane-otel-collector.service"

  if [ ! -f "$config_file" ]; then
    echo "Installing systemd service to $config_file"
  else
    echo "Updating systemd service to $config_file"
  fi

  mkdir -p "$(dirname "$config_file")"

  cat <<EOF >"$config_file"
[Unit]
Description=observIQ's distribution of the OpenTelemetry collector
After=network.target
StartLimitIntervalSec=120
StartLimitBurst=5
[Service]
Type=simple
User=root
Group=bindplane-otel-collector
Environment=PATH=/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin
Environment=OIQ_OTEL_COLLECTOR_HOME=${BDOT_CONFIG_HOME}
Environment=OIQ_OTEL_COLLECTOR_STORAGE=${BDOT_CONFIG_HOME}/storage
WorkingDirectory=${BDOT_CONFIG_HOME}
ExecStart=${BDOT_CONFIG_HOME}/opampsupervisor --config supervisor.yaml
LimitNOFILE=65000
SuccessExitStatus=0
TimeoutSec=20
StandardOutput=journal
Restart=on-failure
RestartSec=5s
KillMode=control-group
[Install]
WantedBy=multi-user.target
EOF

  chown root:root "$config_file"
  chmod 0640 "$config_file"
}

install_initd_service() {
  config_file="/etc/init.d/bindplane-otel-collector"

  if [ ! -f "$config_file" ]; then
    echo "Installing init.d service to $config_file"
  else
    echo "Updating init.d service to $config_file"
  fi

  mkdir -p "$(dirname "$config_file")"

  cat <<EOF >"$config_file"
#!/bin/sh
# observIQ OTEL daemon
# chkconfig: 2345 99 05
# description: observIQ's distribution of the OpenTelemetry collector
# processname: bindplane-otel-collector
# pidfile: /var/run/bindplane-otel-collector.pid

### BEGIN INIT INFO
# Provides: bindplane-otel-collector
# Required-Start:
# Required-Stop:
# Should-Start:
# Default-Start: 3 5
# Default-Stop: 0 1 2 6  
# Description: Start the bindplane-otel-collector service
### END INIT INFO

# Source function library.
# RHEL
if [ -e /etc/init.d/functions ]; then
  STATUS=true
  # shellcheck disable=SC1091
  . /etc/init.d/functions
fi
# SUSE
if [ -e /etc/rc.status ]; then
  RCSTATUS=true
  # Shell functions sourced from /etc/rc.status:
  #      rc_check         check and set local and overall rc status
  #      rc_status        check and set local and overall rc status
  #      rc_status -v     ditto but be verbose in local rc status
  #      rc_status -v -r  ditto and clear the local rc status
  #      rc_failed        set local and overall rc status to failed
  #      rc_failed <num>  set local and overall rc status to <num><num>
  #      rc_reset         clear local rc status (overall remains)
  #      rc_exit          exit appropriate to overall rc status
  # shellcheck disable=SC1091
  . /etc/rc.status

  # First reset status of this service
  rc_reset
fi
# LSB Capable
if [ -e /lib/lsb/init-functions ]; then
  PROC=true
  # shellcheck disable=SC1091
  . /lib/lsb/init-functions
fi

# Return values acc. to LSB for all commands but status:
# 0 - success
# 1 - generic or unspecified error
# 2 - invalid or excess argument(s)
# 3 - unimplemented feature (e.g. "reload")
# 4 - insufficient privilege
# 5 - program is not installed
# 6 - program is not configured
# 7 - program is not running
#
# Note that, for LSB, starting an already running service, stopping
# or restarting a not-running service as well as the restart
# with force-reload (in case signalling is not supported) are
# considered a success.

BINARY=opampsupervisor
PROGRAM=${BDOT_CONFIG_HOME}/"\$BINARY"
START_CMD="nohup ${BDOT_CONFIG_HOME}/\$BINARY > /dev/null 2>&1 &"
LOCKFILE=/var/lock/"\$BINARY"
PIDFILE=/var/run/"\$BINARY".pid

# Exported variables are used by the collector process.
export OIQ_OTEL_COLLECTOR_HOME=${BDOT_CONFIG_HOME}
export OIQ_OTEL_COLLECTOR_STORAGE=${BDOT_CONFIG_HOME}/storage

RETVAL=0
start() {
  [ -x "\$PROGRAM" ] || exit 5

  # shellcheck disable=SC3037
  echo -n "Starting \$0: "

  # RHEL
  if [ "\$STATUS" ]; then
    umask 077

    daemon --pidfile="\$PIDFILE" "\$START_CMD"
    RETVAL=\$?
    # truncate the pid file, just in case
    : > "\$PIDFILE"
    # shellcheck disable=SC2005
    echo "\$(pidof "\$BINARY")" > "\$PIDFILE"
    [ "\$RETVAL" -eq 0 ] && touch "\$LOCKFILE"
  # SUSE
  elif [ "\$RCSTATUS" ]; then
    ## Start daemon with startproc(8). If this fails
    ## the echo return value is set appropriate.

    # NOTE: startproc return 0, even if service is
    # already running to match LSB spec.
    nohup "\$PROGRAM" --config supervisor.yaml > /dev/null 2>&1 &

    # Remember status and be verbose
    rc_status -v

    # truncate the pid file, just in case
    : > "\$PIDFILE"
    # shellcheck disable=SC2005
    echo "\$(pidof "\$BINARY")" > "\$PIDFILE"
  fi
  echo
}

stop() {
  # shellcheck disable=SC3037
  echo -n "Shutting down \$0: "
  # RHEL
  if [ "\$STATUS" ]; then
      killproc -p "\$PIDFILE" -d30 "\$BINARY"
      RETVAL=\$?
      echo
      [ "\$RETVAL" -eq 0 ] && rm -f "\$LOCKFILE"
      return "\$RETVAL"
  # SUSE
  elif [ "\$RCSTATUS" ]; then
      ## Stop daemon with killproc(8) and if this fails
      ## set echo the echo return value.
      killproc -t30 -p "\$PIDFILE" "\$BINARY"

      # Remember status and be verbose
      rc_status -v
  fi
  echo
}

# Currently unimplemented
reload() {
  # RHEL
  #if [ \$STATUS ]; then
  # SUSE
  #elif [ \$RCSTATUS ]; then
  #fi
  echo "Reload is not currently implemented for \$0"
  RETVAL=3
}

# Currently unimplemented
force_reload() {
  # RHEL
  #if [ \$STATUS ]; then
  # SUSE
  #elif [ \$RCSTATUS ]; then
  #fi
  echo "Reload is not currently implemented for \$0, redirecting to restart"
  restart
}

pid_not_running() {
  echo " * \$PROGRAM is not running"
  RETVAL=7
}

pid_status() {
  if [ -e "\$PIDFILE" ]; then
    if ps -p "\$(cat "\$PIDFILE")" > /dev/null; then
      echo " * \$PROGRAM" is running, pid="\$(cat "\$PIDFILE")"
    else
      pid_not_running
    fi
  else
    pid_not_running
  fi
}

otel_status() {
  if [ -e "\$PIDFILE" ]; then
    # shellcheck disable=SC3037
    echo -n "Status of \$0 (\$(cat "\$PIDFILE")) "
  else
    # shellcheck disable=SC3037
    echo -n "Status of \$0 (no pidfile found) "
  fi

  if [ "\$STATUS" ]; then
    status -p "\$PIDFILE" "\$PROGRAM"
    RETVAL=\$?
  elif [ "\$RCSTATUS" ]; then
    ## Check status with checkproc(8), if process is running
    ## checkproc will return with exit status 0.

    # Status has a slightly different for the status command:
    # 0 - service running
    # 1 - service dead, but /var/run/  pid  file exists
    # 2 - service dead, but /var/lock/ lock file exists
    # 3 - service not running

    # NOTE: checkproc returns LSB compliant status values.
    checkproc -p "\$PIDFILE" "\$PROGRAM"
    rc_status -v
  elif [ "\$PROC" ]; then
    status_of_proc -p "\$PIDFILE" "\$PROGRAM" "\$PROGRAM"
    RETVAL=\$?
  else
    pid_status
  fi
  echo
}

cd "\$OIQ_OTEL_COLLECTOR_HOME" || exit 1
case "\$1" in
  # Start the service
  start)
    start
    ;;
  # Stop the service
  stop)
    stop
    ;;
  # Get the status of the service
  status)
    otel_status
    ;;
  # Restart the service by stop, then restart
  restart)
    stop
    # sleep for 1 second to prevent false starts leaving us in a bad state
    sleep 1
    start
    ;;
  # Not currently implemented, but should reload the config file
  reload)
    reload
    ;;
  # Not currently implemented, but should reload the config file.
  # If it fails, restart
  force-reload)
    force_reload
    ;;
  # Conditionally restart the service (only if running already)
  condrestart|try-restart)
    otel_status >/dev/null 2>&1 || exit 0
    restart
    ;;
  *)
    echo "Usage: \$0 {start|stop|restart|condrestart|try-restart|reload|force-reload|status}"
    RETVAL=3
    ;;
esac
cd "\$OLDPWD" || exit 1

if [ "\$RCSTATUS" ]; then
  rc_exit
fi

exit "\$RETVAL"
EOF

  chown root:root "$config_file"
  chmod 0755 "$config_file"
}

manage_systemd_service() {
  # Ensure sysv script isn't present, and if it is remove it
  if [ -f /etc/init.d/bindplane-otel-collector ]; then
    rm -f /etc/init.d/bindplane-otel-collector
  fi

  systemctl daemon-reload

  echo "configured systemd service"

  cat <<EOF

The "bindplane-otel-collector" service has been configured!

The collector's config file can be found here: 
  ${BDOT_CONFIG_HOME}/supervisor_storage/effective.yaml

To view logs from the collector, run:
  sudo tail -F ${BDOT_CONFIG_HOME}/supervisor_storage/agent.log

For more information on configuring the collector, see the docs:
  https://github.com/observIQ/bindplane-otel-collector/tree/main#observiq-opentelemetry-collector

To stop the bindplane-otel-collector service, run:
  sudo systemctl stop bindplane-otel-collector

To start the bindplane-otel-collector service, run:
  sudo systemctl start bindplane-otel-collector

To restart the bindplane-otel-collector service, run:
  sudo systemctl restart bindplane-otel-collector

To enable the service on startup, run:
  sudo systemctl enable bindplane-otel-collector

If you have any other questions please contact us at support@observiq.com
EOF
}

manage_sysv_service() {
  chmod 755 /etc/init.d/bindplane-otel-collector
  chmod 644 /etc/sysconfig/bindplane-otel-collector
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
  # We also change the owner of the binary to bindplane-otel-collector
  chown -R bindplane-otel-collector:bindplane-otel-collector ${BDOT_CONFIG_HOME}/bindplane-otel-collector ${BDOT_CONFIG_HOME}/opampsupervisor ${BDOT_CONFIG_HOME}/plugins/*
  chmod 0640 ${BDOT_CONFIG_HOME}/plugins/*

  # Initialize the log file to ensure it is owned by bindplane-otel-collector.
  # This prevents the service (running as root) from assigning ownership to
  # the root user. By doing so, we allow the user to switch to bindplane-otel-collector
  # user for 'non root' installs.
  mkdir -p /opt/bindplane-otel-collector/supervisor_storage
  touch /opt/bindplane-otel-collector/supervisor_storage/agent.log
  chown bindplane-otel-collector:bindplane-otel-collector /opt/bindplane-otel-collector/supervisor_storage/agent.log
}

install
install_service
finish_permissions
manage_service
