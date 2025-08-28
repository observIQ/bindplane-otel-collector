module github.com/observiq/bindplane-otel-collector/exporter/googlecloudexporter

go 1.24.4

require (
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/collector v0.53.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/googlecloudexporter v0.133.0
	github.com/stretchr/testify v1.11.1
	go.opentelemetry.io/collector/component v1.39.0
	go.opentelemetry.io/collector/consumer v1.39.0
	go.opentelemetry.io/collector/exporter v0.133.0
	go.opentelemetry.io/collector/exporter/exportertest v0.133.0
	go.opentelemetry.io/collector/pdata v1.39.0
	go.opentelemetry.io/collector/processor v1.39.0
	go.opentelemetry.io/collector/processor/batchprocessor v0.133.0
	go.uber.org/multierr v1.11.0
	google.golang.org/api v0.244.0
)

require (
	cloud.google.com/go/logging v1.13.0 // indirect
	cloud.google.com/go/monitoring v1.24.2 // indirect
	cloud.google.com/go/trace v1.11.6 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace v1.29.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/googleapis/gax-go/v2 v2.15.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/knadh/koanf v1.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	go.opentelemetry.io/otel v1.37.0
	go.opentelemetry.io/otel/metric v1.37.0 // indirect
	go.opentelemetry.io/otel/sdk v1.37.0 // indirect
	go.opentelemetry.io/otel/trace v1.37.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/zap v1.27.0
	golang.org/x/net v0.42.0 // indirect
	golang.org/x/oauth2 v0.30.0 // indirect
	golang.org/x/sync v0.16.0 // indirect
	golang.org/x/text v0.27.0 // indirect
	google.golang.org/genproto v0.0.0-20250603155806-513f23925822 // indirect
	google.golang.org/grpc v1.75.0 // indirect
	google.golang.org/protobuf v1.36.7 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

require (
	cloud.google.com/go v0.120.0 // indirect
	cloud.google.com/go/auth v0.16.3 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/compute/metadata v0.7.0 // indirect
	cloud.google.com/go/longrunning v0.6.7 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/internal/resourcemapping v0.53.0 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.8.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.6 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	github.com/knadh/koanf/v2 v2.2.2 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/tinylru v1.2.1 // indirect
	github.com/tidwall/wal v1.1.8 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/collector/client v1.39.0 // indirect
	go.opentelemetry.io/collector/component/componenttest v0.133.0 // indirect
	go.opentelemetry.io/collector/config/configoptional v0.133.0 // indirect
	go.opentelemetry.io/collector/config/configretry v1.39.0 // indirect
	go.opentelemetry.io/collector/confmap v1.39.0 // indirect
	go.opentelemetry.io/collector/confmap/xconfmap v0.133.0 // indirect
	go.opentelemetry.io/collector/consumer/consumererror v0.133.0 // indirect
	go.opentelemetry.io/collector/consumer/consumertest v0.133.0 // indirect
	go.opentelemetry.io/collector/consumer/xconsumer v0.133.0 // indirect
	go.opentelemetry.io/collector/exporter/xexporter v0.133.0 // indirect
	go.opentelemetry.io/collector/extension v1.39.0 // indirect
	go.opentelemetry.io/collector/extension/xextension v0.133.0 // indirect
	go.opentelemetry.io/collector/featuregate v1.39.0 // indirect
	go.opentelemetry.io/collector/internal/telemetry v0.133.0 // indirect
	go.opentelemetry.io/collector/pdata/pprofile v0.133.0 // indirect
	go.opentelemetry.io/collector/pdata/xpdata v0.133.0 // indirect
	go.opentelemetry.io/collector/pipeline v1.39.0 // indirect
	go.opentelemetry.io/collector/receiver v1.39.0 // indirect
	go.opentelemetry.io/collector/receiver/receivertest v0.133.0 // indirect
	go.opentelemetry.io/collector/receiver/xreceiver v0.133.0 // indirect
	go.opentelemetry.io/collector/semconv v0.128.1-0.20250610090210-188191247685 // indirect
	go.opentelemetry.io/contrib/bridges/otelzap v0.12.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.61.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.61.0 // indirect
	go.opentelemetry.io/otel/log v0.13.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.37.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.40.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/time v0.12.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250707201910-8d1bb00bc6a7 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250728155136-f173205681a0 // indirect
)
