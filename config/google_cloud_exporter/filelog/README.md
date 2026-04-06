# File Logging with Google Cloud

The File Log receiver can be used to read log files and send the entries to Google Cloud Logging.

## Limitations

The agent must be installed on the target system. The user running the agent must have permission to read the files targeted by the File Log receiver's configuration.

## Prerequisites

See the [prerequisites](../README.md) doc for Google Cloud prerequisites.

## Examples

**Raw log file**

File Log can read raw log files (without parsing) and forward to Google Cloud Logging. This a
basic implementation. Generally, you will want to parse the log entries in order to make searching
easier within Google Cloud Logging.

See [config.yaml](./config.yaml).

**Regex**

File Log can read raw log files, parse them with regex, and forward to Google Cloud Logging.

See [config_regex.yaml](./config_regex.yaml) for an example using Redis.

**Json**

File Log can read raw log files, parse them as json, and forward to Google Cloud Logging.

See [config_json.yaml](./config_json.yaml) for an example using arbitrary json.
