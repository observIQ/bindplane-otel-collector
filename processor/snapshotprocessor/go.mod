module github.com/observiq/bindplane-otel-collector/processor/snapshotprocessor

go 1.22.7

require (
	github.com/observiq/bindplane-otel-collector/internal/report v1.68.0
	github.com/open-telemetry/opamp-go v0.18.0
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/opampcustommessages v0.118.0
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/golden v0.118.0
	github.com/stretchr/testify v1.10.0
	go.opentelemetry.io/collector/component v0.118.0
	go.opentelemetry.io/collector/consumer v1.24.0
	go.opentelemetry.io/collector/consumer/consumertest v0.118.0
	go.opentelemetry.io/collector/pdata v1.24.0
	go.opentelemetry.io/collector/processor v0.118.0
	go.opentelemetry.io/collector/processor/processortest v0.118.0
	go.uber.org/zap v1.27.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatautil v0.118.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/rogpeppe/go-internal v1.11.0 // indirect
	go.opentelemetry.io/collector/component/componentstatus v0.118.0 // indirect
	go.opentelemetry.io/collector/component/componenttest v0.118.0 // indirect
	go.opentelemetry.io/collector/config/configtelemetry v0.118.0 // indirect
	go.opentelemetry.io/collector/consumer/xconsumer v0.118.0 // indirect
	go.opentelemetry.io/collector/pdata/pprofile v0.118.0 // indirect
	go.opentelemetry.io/collector/pdata/testdata v0.118.0 // indirect
	go.opentelemetry.io/collector/pipeline v0.118.0 // indirect
	go.opentelemetry.io/collector/processor/xprocessor v0.118.0 // indirect
	go.opentelemetry.io/otel v1.32.0 // indirect
	go.opentelemetry.io/otel/metric v1.32.0 // indirect
	go.opentelemetry.io/otel/sdk v1.32.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.32.0 // indirect
	go.opentelemetry.io/otel/trace v1.32.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/net v0.33.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241015192408-796eee8c2d53 // indirect
	google.golang.org/grpc v1.69.4 // indirect
	google.golang.org/protobuf v1.36.3 // indirect
)

replace github.com/observiq/bindplane-otel-collector/internal/report => ../../internal/report
