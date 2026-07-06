module github.com/observiq/bindplane-otel-collector/updater

go 1.26.4

require (
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/open-telemetry/opamp-go v0.23.0
	github.com/spf13/pflag v1.0.10
	github.com/stretchr/testify v1.11.1
	go.uber.org/zap v1.28.0
	golang.org/x/sys v0.46.0
)

require gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/observiq/bindplane-otel-collector/internal/extension/opampconnectionextension v1.103.0
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/observiq/bindplane-otel-collector/internal/extension/opampconnectionextension => ../internal/extension/opampconnectionextension

replace github.com/observiq/bindplane-otel-collector/internal/report => ../internal/report
