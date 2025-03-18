module github.com/observiq/bindplane-otel-collector/exporter/chronicleexporter

go 1.23.6

require (
	github.com/golang/mock v1.6.0
	github.com/google/uuid v1.6.0
	github.com/observiq/bindplane-otel-collector/expr v1.73.0
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl v0.122.0
	github.com/shirou/gopsutil/v3 v3.24.5
	github.com/stretchr/testify v1.10.0
	go.opentelemetry.io/collector/component v1.28.0
	go.opentelemetry.io/collector/component/componenttest v0.122.0
	go.opentelemetry.io/collector/config/configretry v1.28.0
	go.opentelemetry.io/collector/confmap v1.28.0
	go.opentelemetry.io/collector/consumer v1.28.0
	go.opentelemetry.io/collector/consumer/consumererror v0.122.0
	go.opentelemetry.io/collector/exporter v0.122.0
	go.opentelemetry.io/collector/exporter/exportertest v0.122.0
	go.opentelemetry.io/collector/pdata v1.28.0
	go.opentelemetry.io/collector/semconv v0.122.0
	go.opentelemetry.io/otel v1.35.0
	go.opentelemetry.io/otel/metric v1.35.0
	go.opentelemetry.io/otel/sdk/metric v1.35.0
	go.opentelemetry.io/otel/trace v1.35.0
	go.uber.org/goleak v1.3.0
	go.uber.org/zap v1.27.0
	golang.org/x/exp v0.0.0-20240506185415-9bf2ced13842
	golang.org/x/oauth2 v0.27.0
	google.golang.org/genproto/googleapis/api v0.0.0-20250106144421-5f5ef82da422
	google.golang.org/grpc v1.71.0
	google.golang.org/protobuf v1.36.5
)

require (
	cloud.google.com/go/compute/metadata v0.6.0 // indirect
	github.com/alecthomas/participle/v2 v2.1.1 // indirect
	github.com/antchfx/xmlquery v1.4.4 // indirect
	github.com/antchfx/xpath v1.3.3 // indirect
	github.com/antonmedv/expr v1.15.5 // indirect
	github.com/cenkalti/backoff/v5 v5.0.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/elastic/go-grok v0.3.1 // indirect
	github.com/elastic/lunes v0.1.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-viper/mapstructure/v2 v2.2.1 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/iancoleman/strcase v0.3.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/knadh/koanf/maps v0.1.1 // indirect
	github.com/knadh/koanf/providers/confmap v0.1.0 // indirect
	github.com/knadh/koanf/v2 v2.1.2 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/magefile/mage v1.15.0 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal v0.122.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/power-devops/perfstat v0.0.0-20210106213030-5aafc221ea8c // indirect
	github.com/shoenig/go-m1cpu v0.1.6 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/tklauser/go-sysconf v0.3.12 // indirect
	github.com/tklauser/numcpus v0.6.1 // indirect
	github.com/twmb/murmur3 v1.1.8 // indirect
	github.com/ua-parser/uap-go v0.0.0-20240611065828-3a4781585db6 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/collector/consumer/consumertest v0.122.0 // indirect
	go.opentelemetry.io/collector/consumer/xconsumer v0.122.0 // indirect
	go.opentelemetry.io/collector/exporter/xexporter v0.122.0 // indirect
	go.opentelemetry.io/collector/extension v1.28.0 // indirect
	go.opentelemetry.io/collector/extension/xextension v0.122.0 // indirect
	go.opentelemetry.io/collector/featuregate v1.28.0 // indirect
	go.opentelemetry.io/collector/pdata/pprofile v0.122.0 // indirect
	go.opentelemetry.io/collector/pipeline v0.122.0 // indirect
	go.opentelemetry.io/collector/receiver v1.28.0 // indirect
	go.opentelemetry.io/collector/receiver/receivertest v0.122.0 // indirect
	go.opentelemetry.io/collector/receiver/xreceiver v0.122.0 // indirect
	go.opentelemetry.io/otel/sdk v1.35.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/net v0.37.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250115164207-1a7da9e5054f // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/observiq/bindplane-otel-collector/expr => ../../expr
