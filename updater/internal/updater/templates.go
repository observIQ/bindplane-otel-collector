// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package updater provides templates for service files.
package updater

// Constants for service file templates
const systemdServiceTemplate = `[Unit]
Description=observIQ's distribution of the OpenTelemetry collector
After=network.target
StartLimitIntervalSec=120
StartLimitBurst=5
[Service]
Type=simple
User=root
Group={{.Group}}
Environment=PATH=/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin
Environment=OIQ_OTEL_COLLECTOR_HOME={{.InstallDir}}
Environment=OIQ_OTEL_COLLECTOR_STORAGE={{.InstallDir}}/storage
WorkingDirectory={{.InstallDir}}
ExecStart={{.InstallDir}}/observiq-otel-collector --config config.yaml
LimitNOFILE=65000
SuccessExitStatus=0
TimeoutSec=20
StandardOutput=journal
Restart=on-failure
RestartSec=5s
KillMode=process
[Install]
WantedBy=multi-user.target`

const initScriptTemplate = `#!/bin/sh
# observIQ OTEL daemon
# chkconfig: 2345 99 05
# description: observIQ's distribution of the OpenTelemetry collector
# processname: observiq-otel-collector
# pidfile: /var/run/observiq-otel-collector.pid

### BEGIN INIT INFO
# Provides: observiq-otel-collector
# Required-Start: 
# Required-Stop: 
# Should-Start: 
# Default-Start: 3 5
# Default-Stop: 0 1 2 6  
# Description: Start the observiq-otel-collector service
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

BINARY=observiq-otel-collector
PROGRAM=/opt/observiq-otel-collector/"$BINARY"
START_CMD="nohup /opt/observiq-otel-collector/$BINARY > /dev/null 2>&1 &"
LOCKFILE=/var/lock/"$BINARY"
PIDFILE=/var/run/"$BINARY".pid

# Exported variables are used by the collector process.
export OIQ_OTEL_COLLECTOR_HOME=/opt/observiq-otel-collector
export OIQ_OTEL_COLLECTOR_STORAGE=/opt/observiq-otel-collector/storage

RETVAL=0
start() {
  [ -x "$PROGRAM" ] || exit 5

  # shellcheck disable=SC3037
  echo -n "Starting $0: "

  # RHEL
  if [ "$STATUS" ]; then
    umask 077

    daemon --pidfile="$PIDFILE" "$START_CMD"
    RETVAL=$?
    # truncate the pid file, just in case
    : > "$PIDFILE"
    # shellcheck disable=SC2005
    echo "$(pidof "$BINARY")" > "$PIDFILE"
    [ "$RETVAL" -eq 0 ] && touch "$LOCKFILE"
  # SUSE
  elif [ "$RCSTATUS" ]; then
    ## Start daemon with startproc(8). If this fails
    ## the echo return value is set appropriate.

    # NOTE: startproc return 0, even if service is
    # already running to match LSB spec.
    nohup "$PROGRAM" --config config.yaml > /dev/null 2>&1 &

    # Remember status and be verbose
    rc_status -v

    # truncate the pid file, just in case
    : > "$PIDFILE"
    # shellcheck disable=SC2005
    echo "$(pidof "$BINARY")" > "$PIDFILE"
  fi
  echo
}

stop() {
  # shellcheck disable=SC3037
  echo -n "Shutting down $0: "
  # RHEL
  if [ "$STATUS" ]; then
      killproc -p "$PIDFILE" -d30 "$BINARY"
      RETVAL=$?
      echo
      [ "$RETVAL" -eq 0 ] && rm -f "$LOCKFILE"
      return "$RETVAL"
  # SUSE
  elif [ "$RCSTATUS" ]; then
      ## Stop daemon with killproc(8) and if this fails
      ## set echo the echo return value.
      killproc -t30 -p "$PIDFILE" "$BINARY"

      # Remember status and be verbose
      rc_status -v
  fi
  echo
}

# Currently unimplemented
reload() {
  # RHEL
  #if [ $STATUS ]; then
  # SUSE
  #elif [ $RCSTATUS ]; then
  #fi
  echo "Reload is not currently implemented for $0"
  RETVAL=3
}

# Currently unimplemented
force_reload() {
  # RHEL
  #if [ $STATUS ]; then
  # SUSE
  #elif [ $RCSTATUS ]; then
  #fi
  echo "Reload is not currently implemented for $0, redirecting to restart"
  restart
}

pid_not_running() {
  echo " * $PROGRAM is not running"
  RETVAL=7
}

pid_status() {
  if [ -e "$PIDFILE" ]; then
    if ps -p "$(cat "$PIDFILE")" > /dev/null; then
      echo " * $PROGRAM" is running, pid="$(cat "$PIDFILE")"
    else
      pid_not_running
    fi
  else
    pid_not_running
  fi
}

otel_status() {
  if [ -e "$PIDFILE" ]; then
    # shellcheck disable=SC3037
    echo -n "Status of $0 ($(cat "$PIDFILE")) "
  else
    # shellcheck disable=SC3037
    echo -n "Status of $0 (no pidfile found) "
  fi

  if [ "$STATUS" ]; then
    status -p "$PIDFILE" "$PROGRAM"
    RETVAL=$?
  elif [ "$RCSTATUS" ]; then
    ## Check status with checkproc(8), if process is running
    ## checkproc will return with exit status 0.

    # Status has a slightly different for the status command:
    # 0 - service running
    # 1 - service dead, but /var/run/  pid  file exists
    # 2 - service dead, but /var/lock/ lock file exists
    # 3 - service not running

    # NOTE: checkproc returns LSB compliant status values.
    checkproc -p "$PIDFILE" "$PROGRAM"
    rc_status -v
  elif [ "$PROC" ]; then
    status_of_proc -p "$PIDFILE" "$PROGRAM" "$PROGRAM"
    RETVAL=$?
  else
    pid_status
  fi
  echo
}

cd "$OIQ_OTEL_COLLECTOR_HOME" || exit 1
case "$1" in
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
    echo "Usage: $0 {start|stop|restart|condrestart|try-restart|reload|force-reload|status}"
    RETVAL=3
    ;;
esac
cd "$OLDPWD" || exit 1

if [ "$RCSTATUS" ]; then
  rc_exit
fi

exit "$RETVAL"`
