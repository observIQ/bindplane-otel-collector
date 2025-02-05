# Bindplane Agent Updater

The updater is a separate binary that runs as a separate process to update agent artifacts (including the agent itself) when managed by [Bindplane OP](https://observiq.com/).

Because the updater interacts with the service manager, and may edit privileged files, it needs elevated privileges to run (root on Linux + macOS, administrative privileges on Windows).

## Updating Overview

1. The agent receives PackagesAvailable message.
2. The agent downloads tarball containing new updater and updated artifacts (e.g. new agent, plugins, etc.) based on the contents of the PackagesAvailable message.
3. The agent unpacks tarball into `$INSTALL_DIR/tmp/latest`.
4. The agent copies the newest updater binary from `$INSTALL_DIR/tmp/latest` to the working directory.
5. The agent starts the updater in as a separate process in a new process group.
   * If the updater fails to stop the agent within 30 seconds, the agent will kill the updater and abort the update. 

6. The updater starts, then shuts down the agent through the service manager, orphaning the updater process.
7. The updater creates a backup of the current installation directory in `$INSTALL_DIR/tmp/rollback`.
   * If backing up fails for some reason, the updater starts the agent again and exits.
8. The updater installs new artifacts, copying the new files into the the installation directory.
   * If installation fails for some reason, a rollback is initiated.
9. The updater updates the service configuration.
10. The updater starts the agent again, monitoring for agent to be healthy.
    * If the agent is determined to be healthy, the updater exits
    * If the agent is determined unhealthy or doesn't report healthy within 10 seconds, a rollback is initiated. 
11. Upon exit, the updater removes the tmp directory.

## Agent Status Monitoring
The agent saves its current state (installing, installation failed, or installation successful) to a JSON file (`package_statuses.json`) on disk. The updater continuously polls this file for changes in order to detect whether the agent is healthy or not. 

If the agent starts, is able to run successfully, and has its expected version (the version we are upgrading to), the agent will write that the installation was successful to the JSON file. Otherwise, it will write that the installation failed, and will expect the updater to perform a rollback.

If the file indicates the installation failed, or the file still indicates the agent is in an installing state after 10 seconds, a rollback to the previous version is initiated by the updater.

If the file indicates the installation was successful, then the updater exits.

## Updater Rollback
While installing, the updater records a list of actions take (files copied, service actions taken). If something goes wrong during installation, or while monitoring for agent health, then a rollback is initiated.

The rollback will perform the reverse of each action in reverse order. For instance, if the agent binary is replaced with the new agent, then the OpenTelemetry JMX jar is replaced with a new jar, the rollback would copy the backup jar to its previous location, then it would copy the backup agent to its previous location.

Ultimately, this means the rollback process will put the system back into its original state.

After rollback is complete, the updater process exits.

## Debugging
In the event of update failure, you may look in the following places for more information:

* The agent produces logs in the log directory (by default, `$INSTALL_DIR/log/collector.log`).
* The updater produces logs in the agent's log directory, and is always set to log at a debug level (`$INSTALL_DIR/log/updater.log`).
* If the update was initiated through the Bindplane OP UI, an error message should be displayed in the UI.
