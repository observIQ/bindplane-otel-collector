# Linux Installation

## Installing

Installation is done through deb and rpm packages. Installing the agent will also install the `bindplane-otel-collector` service on systemd systems.

Installation artifacts are signed. Information on verifying the signature can be found at [Verifying Artifact Signatures](./verify-signature.md).

### Install/Update script

The agent may be installed through a shell script which will automatically determine which package to install.

This script may also be used to update an existing installation.

To install using the installation script, you may run:

```sh
sudo sh -c "$(curl -fsSlL https://github.com/observIQ/bindplane-otel-collector/releases/latest/download/install_unix.sh)" install_unix.sh
```

#### OpAMP Management

To install the agent and connect the supervisor to an OpAMP management platform, set the following flags.

```sh
sudo sh -c "$(curl -fsSlL https://github.com/observIQ/bindplane-otel-collector/releases/latest/download/install_unix.sh)" install_unix.sh -e <your_endpoint> -s <secret-key>
```

To read more about OpAMP management, see the [supervisor docs](./supervisor.md).

### Installation from local package

To install the agent from a local package use the `-f` with the path to the package.

```sh
sudo sh -c "$(curl -fsSlL https://github.com/observIQ/bindplane-otel-collector/releases/latest/download/install_unix.sh)" install_unix.sh -f <path_to_package>
```

### RPM Installation

First download the RPM package for your architecture from the [releases page](https://github.com/observIQ/bindplane-otel-collector/releases/latest).
Then you may install the package using `rpm`, see this example for installing the amd64 package:

**Note**: Replace `${VERSION}` with the version of the package you downloaded.

```sh
sudo rpm -U ./bindplane-otel-collector_v${VERSION}_linux_amd64.rpm
sudo systemctl enable --now bindplane-otel-collector
```

### DEB Installation

First download the DEB package for your architecture from the [releases page](https://github.com/observIQ/bindplane-otel-collector/releases/latest).
Then you may install the package using `dpkg`, see this example for installing the amd64 package:

**Note**: Replace `${VERSION}` with the version of the package you downloaded.

```sh
sudo dpkg -i ./bindplane-otel-collector_v${VERSION}_linux_amd64.deb
sudo systemctl enable --now bindplane-otel-collector
```

## Configuring the Agent

After installing, systems with systemd installed will have the `bindplane-otel-collector` service up and running!

**Configuration**

The config file for the agent can be found at `/opt/bindplane-otel-collector/supervisor_storage/effective.yaml`. If you modify this file, the supervisor will overwrite it on startup with the last config it received from an OpAMP platform. The best way to change the agent's configuration is to send a new config to the supervisor via OpAMP.

If this method of collector management does not work for your use case, see this [alternative option](./supervisor.md#alternatives)

**Logging**

Logs from the agent will appear in `/opt/bindplane-otel-collector/supervisor_storage/agent.log`. You may run `sudo tail -F /opt/bindplane-otel-collector/supervisor_storage/agent.log` to view them.

Stdout and stderr for the supervisor process are recorded via journald. You man run `sudo journalctl -u bindplane-otel-collector.service` to view them.

**Permissions**

By default, the `bindplane-otel-collector` service runs as the "root" user. Some OpenTelemetry components require root permissions in order to read log files owned by other users.

It may be desirable to run the agent as an unprivileged user. For example, a metrics only agent does not require root access.

To run the agent as the `bindplane-otel-collector` user, you may create a systemd override.

Run `sudo systemctl edit bindplane-otel-collector` and paste the following config:

```
[Service]
User=bindplane-otel-collector
```

Reload Systemd:

```shell
sudo systemctl daemon-reload
```

Restart the agent for these changes to take effect.

## Restarting the Agent

On systemd systems, the agent may be restarted with the following command:

```sh
systemctl restart bindplane-otel-collector
```

## Stopping the Agent

On systemd systems, the agent may be stopped with the following command:

```sh
systemctl stop bindplane-otel-collector
```

## Starting the Agent

On systemd systems, the agent may be started with the following command:

```sh
systemctl start bindplane-otel-collector
```

## Uninstalling

### RPM Uninstall

To uninstall the rpm package, run:

```sh
sudo rpm -e bindplane-otel-collector
```

### DEB Uninstall

To uninstall the deb package, run:

```sh
sudo dpkg -r bindplane-otel-collector
```

### Uninstall script

To uninstall an installation made with the install script, run:

```sh
sudo sh -c "$(curl -fsSlL https://github.com/observIQ/bindplane-otel-collector/releases/latest/download/install_unix.sh)" install_unix.sh -r
```
