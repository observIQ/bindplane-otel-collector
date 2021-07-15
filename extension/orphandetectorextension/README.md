# Orphan Detector Extension

This extension checks on an interval if its parent process has died (and consequently, this process has become orphaned).
The extension will shut down the process if it discovers it has been orphaned.

## Configuration
The following configuration options are available:

- `interval` (default: `"5s"`): The interval on which the extension polls to check if it has become an orphan
- `ppid` (default: `os.Getppid()`): The parent processes ID. The default value is the parent's process ID at the time the config is created.
    It is recommended that you set this using the `--set` flag for the cli, as you otherwise introduce a race condition where the parent process may exit
    before `os.Getppid()` is called.
- `die_on_init_parent` (default: `false`): If this is true, a detected ppid of 1 will cause the extension to shut down this process on non-windows systems

The full list of settings exposed for this extension is listed [here](config.go)