## Generic Installation Scripts

These scripts (`install_macos.sh` and `install_unix.sh`) provide automated installation and management of services across macOS and Unix/Linux systems. They handle downloading, configuring, and managing services with support for OpAMP integration.

### Usage

```bash
# Basic installation
sudo ./install_macos.sh -d mydistribution -u https://github.com/org/repo
sudo ./install_unix.sh -d mydistribution -u https://github.com/org/repo

# Installation with specific version
sudo ./install_macos.sh -d mydistribution -u https://github.com/org/repo -v 1.2.3

# Installation with OpAMP configuration
sudo ./install_unix.sh -d mydistribution \
    -u https://github.com/org/repo \
    -e ws://opamp.example.com/v1/opamp \
    -s your-secret-key \
    -l "env=prod,region=us-west"

# Uninstallation
sudo ./install_macos.sh -d mydistribution -r
sudo ./install_unix.sh -d mydistribution -r
```

### Command Line Arguments

| Argument             | Description                          | Required |
| -------------------- | ------------------------------------ | -------- |
| `-d, --distribution` | Name of the distribution to install  | Yes      |
| `-v, --version`      | Version to install (default: latest) | No       |
| `-u, --url`          | GitHub repository URL                | Yes\*    |
| `-e, --endpoint`     | OpAMP endpoint URL                   | No       |
| `-s, --secret-key`   | OpAMP secret key                     | No       |
| `-l, --labels`       | Comma-separated list of labels       | No       |
| `-r, --uninstall`    | Uninstall the package                | No       |
| `-h, --help`         | Show help message                    | No       |

\* Required unless `REPOSITORY_URL` is set in the script

### Installation Process

1. **Prerequisite Check**: Verifies required tools (curl, printf) are available
2. **Architecture Detection**: Determines system architecture for package selection
3. **Package Download**: Downloads appropriate package from repository
4. **Installation**:
   - macOS: Extracts tar.gz to `/opt/<distribution>`
   - Unix: Installs via package manager (deb/rpm) or extracts tar.gz
5. **Configuration**:
   - Creates supervisor configuration
   - Sets appropriate permissions
   - Configures OpAMP integration if specified
6. **Service Management**:
   - macOS: Configures and loads launchd service
   - Unix: Configures and starts systemd service

### Service Management

#### macOS

- Uses launchd for service management
- Service files stored in `/Library/LaunchDaemons/com.<distribution>.plist`
- Automatic service start on boot

#### Unix/Linux

- Uses systemd for service management
- Automatic service detection and configuration
- Supports service enable/disable/restart operations

### OpAMP Integration

The scripts support OpAMP (Open Agent Management Protocol) integration with:

- Configurable endpoints
- Secret key authentication
- Custom labels for agent identification
- Supervisor configuration for agent management
- Secure storage of credentials

### File Locations

- Installation Directory: `/opt/<distribution>`
- Supervisor Config: `/opt/<distribution>/supervisor-config.yaml`
- Supervisor Logs: `/opt/<distribution>/supervisor.log`
- Service Storage: `/opt/<distribution>/supervisor_storage`
- Collector Logs: `/opt/<distribution>/supervisor_storage/agent.log`

### Requirements

- Root/sudo access
- curl
- printf
- systemd (for Unix/Linux service management)
- launchd (for macOS service management)

### Security Considerations

- Scripts must run as root
- Supervisor configuration file permissions are set to 600
- Service files are owned by root with appropriate permissions
- Secret keys are stored securely in the supervisor configuration

### Troubleshooting

- Check service status:

  ```bash
  # macOS
  sudo launchctl list | grep <distribution>

  # Unix/Linux
  sudo systemctl status <distribution>
  ```

- View supervisor logs:
  ```bash
  sudo cat /opt/<distribution>/supervisor.log
  ```
- Verify supervisor configuration:
  ```bash
  sudo cat /opt/<distribution>/supervisor-config.yaml
  ```
