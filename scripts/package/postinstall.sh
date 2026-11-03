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

# Never create a file that group or others can write, whatever umask the
# package manager runs with. This keeps the systemd unit and the User=
# drop-in root-only from the moment they are created.
umask go-w

# Read's optional package overrides. Users should deploy the override
# file before installing BDOT for the first time. The override should
# not be modified unless uninstalling and re-installing.
[ -f /etc/default/observiq-otel-collector ] && . /etc/default/observiq-otel-collector
[ -f /etc/sysconfig/observiq-otel-collector ] && . /etc/sysconfig/observiq-otel-collector

# The collectors installation directory
: "${BDOT_CONFIG_HOME:=/opt/observiq-otel-collector}"

# Whether or not to run the collector as an unprivileged user.
: "${BDOT_UNPRIVILEGED:=false}"

# Configurable runtime user/group
: "${BDOT_USER:=bdot}"
: "${BDOT_GROUP:=bdot}"


# run_as_home_owner runs a command that writes in BDOT_CONFIG_HOME. When
# unprivileged, the runtime user owns BDOT_CONFIG_HOME and can replace any
# entry in it with a symbolic link, which root would follow. So the command
# runs as that user and can only reach files the user could already change.
# Otherwise nothing runs as the runtime user, and the command runs as root.
run_as_home_owner() {
    if [ "${BDOT_UNPRIVILEGED}" = "true" ]; then
        runuser -u "$BDOT_USER" -g "$BDOT_GROUP" -- "$@"
    else
        "$@"
    fi
}

install() {
    mkdir -p "${BDOT_CONFIG_HOME}"
    chmod 0755 "${BDOT_CONFIG_HOME}"
    chown "$BDOT_USER:$BDOT_GROUP" "${BDOT_CONFIG_HOME}"

    share_dir="/usr/share/observiq-otel-collector"
    stage_dir="${share_dir}/stage/observiq-otel-collector"

    # Rename binaries in staging directory to avoid Linux binary locking issues
    # during copy operation
    mv "${stage_dir}/observiq-otel-collector" "${stage_dir}/observiq-otel-collector.new"

    # Goreleaser does not set plugin file permissions. When unprivileged,
    # finish_permissions doesn't change files in BDOT_CONFIG_HOME, so set them
    # here while root still owns the stage.
    if [ "${BDOT_UNPRIVILEGED}" = "true" ]; then
        chmod 0640 "$stage_dir"/plugins/*
    fi

    # Prepare staged files with runtime ownership so destination does not need
    # post-copy ownership changes. This helps to ensure permissions do not flap
    # between root and the runtime user.
    chown -R "$BDOT_USER:$BDOT_GROUP" "$stage_dir"

    # Ensure updater is owned by root. When unprivileged, the runtime user
    # copies the files into BDOT_CONFIG_HOME, so it owns the updater anyway.
    if [ "${BDOT_UNPRIVILEGED}" != "true" ]; then
        chown root:root "$stage_dir/updater"
    fi

    # Seed default configs only when absent so upgrades/reinstalls preserve
    # user edits. The stage dir is ephemeral, so pruning it here is safe.
    for cfg in config.yaml logging.yaml; do
        if [ -f "${BDOT_CONFIG_HOME}/${cfg}" ]; then
            rm -f "${stage_dir}/${cfg}"
        fi
    done

    # Remove each existing file before copying over it instead of writing
    # into it. A root install leaves the updater owned by root, which the
    # runtime user can't write but can remove, since it owns
    # BDOT_CONFIG_HOME. A symbolic link in place of a file is replaced, not
    # followed.
    run_as_home_owner cp -r --preserve --remove-destination \
      "$stage_dir"/* \
      "${BDOT_CONFIG_HOME}"

    # Perform atomic moves for binary files to replace running binaries
    run_as_home_owner mv "${BDOT_CONFIG_HOME}/observiq-otel-collector.new" "${BDOT_CONFIG_HOME}/observiq-otel-collector"

    rm -rf "$share_dir"
}

install_service() {
  if command -v systemctl > /dev/null 2>&1; then
    install_systemd_service
  else
    install_initd_service
  fi
}

install_systemd_service() {
  config_file="/usr/lib/systemd/system/observiq-otel-collector.service"

  if [ ! -f "$config_file" ]; then
    echo "Installing systemd service to $config_file"
  else
    echo "Updating systemd service to $config_file"
  fi

  mkdir -p "$(dirname "$config_file")"

  cat << EOF > "$config_file"
[Unit]
Description=observIQ's distribution of the OpenTelemetry collector
After=network.target
StartLimitIntervalSec=120
StartLimitBurst=5
[Service]
Type=simple
User=root
Group=${BDOT_GROUP}
Environment=PATH=/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin
Environment=BINDPLANE_COLLECTOR_HOME=${BDOT_CONFIG_HOME}
Environment=BINDPLANE_COLLECTOR_STORAGE=${BDOT_CONFIG_HOME}/storage
WorkingDirectory=${BDOT_CONFIG_HOME}
ExecStart=${BDOT_CONFIG_HOME}/observiq-otel-collector --config config.yaml
LimitNOFILE=65000
SuccessExitStatus=0
TimeoutSec=20
StandardOutput=journal
Restart=on-failure
RestartSec=5s
KillMode=process
[Install]
WantedBy=multi-user.target
EOF

  chown root:root "$config_file"
  chmod 0640 "$config_file"

  # Ensure the override dir exists.
  override_dir="/etc/systemd/system/observiq-otel-collector.service.d"
  if [ ! -d "$override_dir" ]; then
    mkdir -p "$override_dir"
    echo "Created systemd override directory at $override_dir"
  fi
  # systemd applies these drop-ins to the service, so only root may change them.
  chown root:root "$override_dir"
  chmod 0755 "$override_dir"

  # Write the User= drop-in when BDOT_UNPRIVILEGED is true. Otherwise there is
  # none to remove: check_runtime_user_change refuses the install when one is
  # present, and preremove.sh removes it on uninstall.
  override_user_path="${override_dir}/10-package-customizations-username.conf"
  if [ "${BDOT_UNPRIVILEGED}" = "true" ]; then
    cat << EOF > "${override_user_path}"
[Service]
User=${BDOT_USER}
EOF
    chown root:root "${override_user_path}"
    chmod 0644 "${override_user_path}"
    echo "Configured systemd service to run as ${BDOT_USER} user in ${override_user_path}"
  fi
}

install_initd_service() {
  config_file="/etc/init.d/observiq-otel-collector"

  if [ ! -f "$config_file" ]; then
    echo "Installing init.d service to $config_file"
  else
    echo "Updating init.d service to $config_file"
  fi

  mkdir -p "$(dirname "$config_file")"

  cat << EOF > "$config_file"
#!/bin/sh
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
PROGRAM=${BDOT_CONFIG_HOME}/"\$BINARY"
START_CMD="nohup ${BDOT_CONFIG_HOME}/\$BINARY > /dev/null 2>&1 &"
LOCKFILE=/var/lock/"\$BINARY"
PIDFILE=/var/run/"\$BINARY".pid

# Exported variables are used by the collector process.
export BINDPLANE_COLLECTOR_HOME=${BDOT_CONFIG_HOME}
export BINDPLANE_COLLECTOR_STORAGE=${BDOT_CONFIG_HOME}/storage

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
    nohup "\$PROGRAM" --config config.yaml > /dev/null 2>&1 &

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

cd "\$BINDPLANE_COLLECTOR_HOME" || exit 1
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
  if [ -f /etc/init.d/observiq-otel-collector ]; then
    rm -f /etc/init.d/observiq-otel-collector
  fi

  systemctl daemon-reload

  echo "configured systemd service"

  cat << EOF

The "observiq-otel-collector" service has been configured!

The collector's config file can be found here: 
  ${BDOT_CONFIG_HOME}/config.yaml

To view logs from the collector, run:
  sudo tail -F ${BDOT_CONFIG_HOME}/log/collector.log

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
  echo "configured sysv service"
}

init_type() {
  # Determine if we need service or systemctl for prereqs
  if command -v systemctl > /dev/null 2>&1; then
    command printf "systemd"
    return
  elif command -v service > /dev/null 2>&1; then
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
  esac
}

finish_permissions() {
  # When unprivileged, root must not change files in BDOT_CONFIG_HOME, because
  # the runtime user owns it and can replace any entry with a symbolic link.
  # install already copied the files as that user and set the plugin modes in
  # the stage, and the collector creates its own log file.
  if [ "${BDOT_UNPRIVILEGED}" = "true" ]; then
    return 0
  fi

  # Goreleaser does not set plugin file permissions, so do them here
  # We also change the owner of the binary to observiq-otel-collector
  chown -R "$BDOT_USER:$BDOT_GROUP" ${BDOT_CONFIG_HOME}/observiq-otel-collector ${BDOT_CONFIG_HOME}/plugins/*
  chmod 0640 ${BDOT_CONFIG_HOME}/plugins/*

  # Initialize the log file to ensure it is owned by observiq-otel-collector.
  # This prevents the service (running as root) from assigning ownership to
  # the root user. By doing so, we allow the user to switch to observiq-otel-collector
  # user for 'non root' installs.
  touch ${BDOT_CONFIG_HOME}/log/collector.log
  chown "$BDOT_USER:$BDOT_GROUP" ${BDOT_CONFIG_HOME}/log/collector.log
}

# check_runtime_user_change fails the install before anything is changed when
# the package overrides would change the user that an unprivileged install
# runs as. That user owns BDOT_CONFIG_HOME and can replace any entry in it
# with a symbolic link, and an install for root or for another user would
# follow those links as root. It also refuses to make an existing root
# install unprivileged, which would start the collector as the runtime user
# against files root created, such as those in storage/. The package writes
# the User= drop-in and root owns it, so the runtime user can't change it.
# preinstall.sh runs the same check, so keep the two the same. This script
# runs it too because dpkg can run it again, for example with
# "dpkg --configure", after the overrides changed.
check_runtime_user_change() {
    user_dropin="/etc/systemd/system/observiq-otel-collector.service.d/10-package-customizations-username.conf"

    # Whether this replaces an installed package: dpkg passes "upgrade" to
    # preinst and the previous version to "configure", and rpm counts more
    # than one installed instance. dpkg also passes the previous version to
    # "configure" when it installs a package that was removed but not
    # purged. Removing the package deletes the collector binary, so without
    # it there is no install to replace.
    upgrade=false
    case "$1" in
        upgrade) upgrade=true ;;
        configure) [ -z "$2" ] || upgrade=true ;;
        [0-9]*) [ "$1" -lt 2 ] || upgrade=true ;;
    esac
    if [ ! -e "${BDOT_CONFIG_HOME}/observiq-otel-collector" ]; then
        upgrade=false
    fi

    if [ -f "$user_dropin" ]; then
        current_user="$(sed -n 's/^User=//p' "$user_dropin")"
        if [ "$BDOT_UNPRIVILEGED" = "true" ] && [ "$BDOT_USER" = "$current_user" ]; then
            return
        fi
        set_by="set by ${user_dropin}"
    elif [ "$BDOT_UNPRIVILEGED" = "true" ] && [ "$upgrade" = "true" ]; then
        current_user="root"
        set_by="since there is no ${user_dropin}"
    else
        return 0
    fi

    # Name the override file to set the values in. Both are sourced, and
    # /etc/sysconfig last, so it wins. rpm passes a number as $1, dpkg a word.
    override_file="/etc/default/observiq-otel-collector"
    if [ -f /etc/sysconfig/observiq-otel-collector ]; then
        override_file="/etc/sysconfig/observiq-otel-collector"
    elif [ ! -f "$override_file" ]; then
        case "$1" in
            [0-9]*) override_file="/etc/sysconfig/observiq-otel-collector" ;;
        esac
    fi

    echo "ERROR: The collector service runs as \"${current_user}\", ${set_by}." >&2
    echo "Changing BDOT_UNPRIVILEGED or BDOT_USER on an existing install is not supported. To change them, uninstall the package, remove ${BDOT_CONFIG_HOME}, and install it again." >&2
    echo "To keep the existing install, set these values in ${override_file}, as described at https://docs.bindplane.com/configuration/bindplane-otel-collector/linux-package-configuration:" >&2
    if [ -f "$user_dropin" ]; then
        echo "    BDOT_UNPRIVILEGED=true" >&2
        echo "    BDOT_USER=${current_user}" >&2
    else
        echo "    BDOT_UNPRIVILEGED=false" >&2
    fi
    echo "Then install the package again." >&2
    exit 1
}

# dpkg runs the installed package's postinst with abort-upgrade, abort-remove
# or abort-deconfigure when the new package's preinst or the prerm failed,
# for example when check_runtime_user_change refused the upgrade. The
# installed files are untouched and the stage directory is gone, so there is
# nothing to do, and failing here would leave the package half-installed.
# rpm passes a number instead.
case "$1" in
    abort-*)
        exit 0
        ;;
esac

check_runtime_user_change "$@"
install
install_service
finish_permissions
manage_service
