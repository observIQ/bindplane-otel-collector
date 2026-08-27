module github.com/observiq/bindplane-otel-collector/updater

go 1.26.4

require (
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/open-telemetry/opamp-go v0.23.0
	github.com/spf13/pflag v1.0.10
	github.com/stretchr/testify v1.12.1
	go.uber.org/zap v1.28.0
	golang.org/x/sys v0.47.0
)

require go.yaml.in/yaml/v3 v3.0.5 // indirect

require (
	github.com/observiq/bindplane-otel-collector/internal/extension/opampconnectionextension v1.107.0
	github.com/stretchr/objx v0.5.3 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace github.com/observiq/bindplane-otel-collector/internal/extension/opampconnectionextension => ../internal/extension/opampconnectionextension

replace github.com/observiq/bindplane-otel-collector/internal/report => ../internal/report
