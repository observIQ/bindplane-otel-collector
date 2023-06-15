module github.com/observiq/observiq-otel-collector/processor/logcountprocessor

go 1.19

require (
	github.com/observiq/observiq-otel-collector/counter v1.28.0
	github.com/observiq/observiq-otel-collector/expr v1.28.0
	github.com/observiq/observiq-otel-collector/receiver/routereceiver v1.28.0
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl v0.79.0
	github.com/stretchr/testify v1.8.4
	go.opentelemetry.io/collector v0.79.0
	go.opentelemetry.io/collector/component v0.79.0
	go.opentelemetry.io/collector/consumer v0.79.0
	go.opentelemetry.io/collector/pdata v1.0.0-rcv0012
	go.opentelemetry.io/collector/receiver v0.79.0
	go.uber.org/zap v1.24.0
)

require (
	github.com/benbjohnson/clock v1.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.opentelemetry.io/collector/confmap v0.79.0 // indirect
	go.opentelemetry.io/collector/featuregate v1.0.0-rcv0012 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

require (
	github.com/alecthomas/participle/v2 v2.0.0 // indirect
	github.com/antonmedv/expr v1.12.5 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/iancoleman/strcase v0.2.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/knadh/koanf v1.5.0 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal v0.79.0 // indirect
	go.opentelemetry.io/otel v1.16.0 // indirect
	go.opentelemetry.io/otel/metric v1.16.0 // indirect
	go.opentelemetry.io/otel/trace v1.16.0 // indirect
	go.uber.org/atomic v1.10.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/exp v0.0.0-20221205204356-47842c84f3db // indirect
	golang.org/x/net v0.10.0 // indirect
	golang.org/x/sys v0.8.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1 // indirect
	google.golang.org/grpc v1.55.0 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
)

replace github.com/observiq/observiq-otel-collector/receiver/routereceiver => ../../receiver/routereceiver

replace github.com/observiq/observiq-otel-collector/expr => ../../expr

replace github.com/observiq/observiq-otel-collector/counter => ../../counter

// Pull in changes to OTTL to allow body to be indexed
// Can be removed when ottl is updated to v0.79.0
// Points to this commit: https://github.com/open-telemetry/opentelemetry-collector-contrib/commit/85a618f8bb7204b63d3d7bf0f679cc61c0f42ea0
replace github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl v0.78.0 => github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl v0.78.1-0.20230524155147-85a618f8bb72
