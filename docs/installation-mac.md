# macOS Installation

## Installing
The agent may be installed through a shell script.

This script may also be used to update an existing installation.

To install using the installation script, you may run:
```sh
sudo sh -c "$(curl -fsSlL https://github.com/observIQ/bindplane-otel-collector/releases/latest/download/install_macos.sh)" install_macos.sh
```

Installation artifacts are signed. Information on verifying the signature can be found at [Verifying Artifact Signatures](./verify-signature.md).

### OpAMP Management

To install the agent and connect the supervisor to an OpAMP management platform, set the following flags. 

```sh
sudo sh -c "$(curl -fsSlL https://github.com/observIQ/bindplane-otel-collector/releases/latest/download/install_macos.sh)" install_macos.sh -e <your_endpoint> -s <secret-key>
```

To read more about OpAMP management, see the [supervisor docs](./supervisor.md).

## Configuring the Agent

The agent is ran and managed by the [OpenTelemetry supervisor](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/cmd/opampsupervisor). The supervisor must receive the agent's configuration from an OpAMP management platform, after which it will stop and restart the agent with the new config.

The supervisor remembers the last config it received via OpAMP and always rewrites the agent's config file with it when it starts. This means you can't manually edit the agent's config file on disk. The best way to modify the configuration is to send a new one from the OpAMP platform the supervisor is connected to.

The agent configuration file is located at `/opt/observiq-otel-collector/supervisor_storage/effective.yaml`.

If this method of collector management does not work for your use case, see this [alternative option](./supervisor.md#alternatives)

**Logging**

Logs from the agent will appear in `/opt/observiq-otel-collector/supervisor_storage/agent.log`. You may run `sudo tail -F /opt/observiq-otel-collector/supervisor_storage/agent.log` to view them.

Stderr for the supervisor process can be found at `/var/log/observiq_collector.err`.

## Agent Services Commands

The agent uses `launchctl` to control the agent lifecycle using the following commands.

```sh
# Start the agent
sudo launchctl load /Library/LaunchDaemons/com.observiq.collector.plist

# Stop the agent
sudo launchctl unload /Library/LaunchDaemons/com.observiq.collector.plist
```

## Uninstalling

To uninstall an installation made with the install script, run:
```sh
sudo sh -c "$(curl -fsSlL https://github.com/observIQ/bindplane-otel-collector/releases/latest/download/install_macos.sh)" install_macos.sh -r
```
