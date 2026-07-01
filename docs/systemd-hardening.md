# Systemd Hardening

The collector's systemd unit file includes security hardening directives that restrict
the process to the minimum privileges required for normal operation.

## Capabilities

The collector runs with two Linux capabilities:

- **CAP_NET_BIND_SERVICE** — Allows binding to privileged ports (< 1024).
- **CAP_DAC_READ_SEARCH** — Allows reading files regardless of file permissions.
  Required for receivers that read host files (e.g., `/proc`, `/sys`, log files).

### Stripping CAP_DAC_READ_SEARCH

If your workload does not need to read arbitrary host files, you can remove this
capability via a systemd drop-in override:

```ini
# /etc/systemd/system/observiq-otel-collector.service.d/strip-dac.conf
[Service]
CapabilityBoundingSet=CAP_NET_BIND_SERVICE
AmbientCapabilities=CAP_NET_BIND_SERVICE
```

Then reload and restart:

```sh
sudo systemctl daemon-reload
sudo systemctl restart observiq-otel-collector
```

### Adding CAP_SYS_PTRACE

If you use the process scraper receiver, you may need `CAP_SYS_PTRACE` to read
`/proc/<pid>/exe` and similar files for other processes:

```ini
# /etc/systemd/system/observiq-otel-collector.service.d/ptrace.conf
[Service]
CapabilityBoundingSet=CAP_NET_BIND_SERVICE CAP_DAC_READ_SEARCH CAP_SYS_PTRACE
AmbientCapabilities=CAP_NET_BIND_SERVICE CAP_DAC_READ_SEARCH CAP_SYS_PTRACE
```

## Filesystem Sandboxing

- **ProtectSystem=strict** — The entire filesystem is mounted read-only except for
  paths listed in `ReadWritePaths`.
- **ProtectHome=true** — `/home`, `/root`, and `/run/user` are inaccessible. Users who need access to home directory paths (e.g., AWS credentials in `~/.aws/`) can override this with a systemd drop-in: `systemctl edit observiq-otel-collector` and set `ProtectHome=read-only` or `ProtectHome=false`.
- **PrivateTmp=true** — The collector gets its own private `/tmp` and `/var/tmp`.
- **PrivateDevices=true** — Access to physical devices is denied.
- **ReadWritePaths** — The install directory, storage directory, and log directory
  are writable. By default these are all under `/opt/observiq-otel-collector`.

### Custom ReadWritePaths

If the collector needs to write to additional paths (e.g., a custom output directory),
add them via a drop-in:

```ini
# /etc/systemd/system/observiq-otel-collector.service.d/extra-paths.conf
[Service]
ReadWritePaths=/opt/observiq-otel-collector /var/data/custom-output
```

Note: `ReadWritePaths` is not additive across drop-ins — you must include all
required paths in the override.

## Process and Kernel Hardening

The following directives restrict the collector from performing privileged operations:

| Directive | Effect |
|---|---|
| NoNewPrivileges=yes | Prevents gaining new privileges via setuid/setgid |
| ProtectKernelTunables=yes | `/proc/sys`, `/sys` are read-only |
| ProtectKernelModules=yes | Cannot load kernel modules |
| ProtectKernelLogs=yes | Cannot read kernel log ring buffer |
| ProtectControlGroups=yes | `/sys/fs/cgroup` is read-only |
| ProtectClock=yes | Cannot change the system clock |
| ProtectHostname=yes | Cannot change the hostname |
| RestrictSUIDSGID=yes | Cannot create setuid/setgid files |
| RestrictNamespaces=yes | Cannot create new namespaces |
| RestrictRealtime=yes | Cannot use realtime scheduling |
| LockPersonality=yes | Cannot change execution domain |
| SystemCallArchitectures=native | Only native syscall ABI allowed |
| KeyringMode=private | Gets its own kernel keyring |
| SystemCallFilter=@system-service @network-io | Only system-service and network-io syscalls |

## Group Membership

To grant the collector access to specific resources without capabilities:

- **journald** — Add `bdot` to the `systemd-journal` group.
- **/var/log** — Add `bdot` to the `adm` group.
- **Docker socket** — Add `bdot` to the `docker` group.

```sh
sudo usermod -aG systemd-journal bdot
sudo usermod -aG adm bdot
sudo usermod -aG docker bdot
sudo systemctl restart observiq-otel-collector
```

## Overridable Directories

The storage and log directories can be configured independently at install time
via `/etc/default/observiq-otel-collector` or `/etc/sysconfig/observiq-otel-collector`:

```sh
BDOT_STORAGE=/mnt/data/collector-storage
BDOT_LOGS=/var/log/observiq-otel-collector
```

These must be set before the first package installation. The directories are
automatically included in `ReadWritePaths`.
