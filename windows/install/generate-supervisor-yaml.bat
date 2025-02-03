@echo off
setlocal

set "install_dir=%~1"
set "endpoint=%~2"
set "secret_key=%~3"
set "labels=%~4"

echo %install_dir%
echo %endpoint%
echo %secret_key%
echo %labels%

if "%endpoint%"=="" (
    echo Endpoint not specified, using default value of 'ws://localhost:3001/v1/opamp'
    set "endpoint=ws://localhost:3001/v1/opamp"
)

set "supervisorFile=%install_dir%supervisor.yaml"
set "agentBinary=%install_dir%bindplane-otel-collector.exe"

echo Writing manager yaml

set "serverField=server:"
set "endpointField=  endpoint: "%endpoint%""
set "headersField=  headers:"
set "authorizationField=    Authorization: "Secret-Key %secret_key%""
set "tlsField=  tls:"
set "insecureField=    insecure: true"
set "insecureSkipField=    insecure_skip_verify: true"

set "capabilitiesField=capabilities:"
set "acceptsRemoteCfgField=  accepts_remote_config: true"
set "reportsRemoteCfgField=  reports_remote_config: true"

set "agentField=agent:"
set "executablePathField=  executable: '%agentBinary%'"
set "configApplyTimeoutField=  config_apply_timeout: 30s"
set "bootstrapTimeoutField=  bootstrap_timeout: 5s"
set "descriptionField=  description:"
set "nonIdentifyingAttributesField=    non_identifying_attributes:"
set "serviceLabelsField=      service.labels: "%labels%""

set "storageField=storage:"
set "directoryField=  directory: '%install_dir%supervisor_storage'"

set "telemetryField=telemetry:"
set "logsField=  logs:"
set "logsLevelField=    level: 0"
set "logsOutputPathsField=    output_paths: ['%install_dir%supervisor_storage\supervisor.log']"

echo %serverField% >"%supervisorFile%"
echo %endpointField% >>"%supervisorFile%"
echo %headersField% >>"%supervisorFile%"
echo %authorizationField% >>"%supervisorFile%"
echo %tlsField% >>"%supervisorFile%"
echo %insecureField% >>"%supervisorFile%"
echo %insecureSkipField% >>"%supervisorFile%"

echo %capabilitiesField% >>"%supervisorFile%"
echo %acceptsRemoteCfgField% >>"%supervisorFile%"
echo %reportsRemoteCfgField% >>"%supervisorFile%"

echo %agentField% >>"%supervisorFile%"
echo %executablePathField% >>"%supervisorFile%"
echo %descriptionField% >>"%supervisorFile%"
echo %nonIdentifyingAttributesField% >>"%supervisorFile%"
echo %serviceLabelsField% >>"%supervisorFile%"

echo %storageField% >>"%supervisorFile%"
echo %directoryField% >>"%supervisorFile%"

echo %telemetryField% >>"%supervisorFile%"
echo %logsField% >>"%supervisorFile%"
echo %logsLevelField% >>"%supervisorFile%"
echo %logsOutputPathsField% >>"%supervisorFile%"

endlocal
