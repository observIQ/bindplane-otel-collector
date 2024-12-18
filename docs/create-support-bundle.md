# Create Support Bundle Scripts

There are three support bundle scripts that are used to collect information about the system and the BDOT. They must be run on the machine hosting the BDOT. The scripts are located in the [scripts/support](../scripts/support) directory. They produce output in the directory they are run from. All of the support scripts collect the following information:

    - Agent logs
    - Agent configuration
    - Agent panic log
    - System information

## Specific scripts:

1. `create-support-bundle.sh` - This script is used for linux systems to collect information about the system and the agent. It will package the information into a tar.gz file. It must be run with `sudo` or as root.

2. `create-support-bundle.bat` - This batch script is used for Windows systems without Powershell to collect information about the system and the agent. It will package the information into a directory.

3. `create-support-bundle.ps1` - This Powershell script is used for Windows systems to collect information about the system and the agent. It will package the information into a zip file.
