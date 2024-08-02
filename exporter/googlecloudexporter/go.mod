module github.com/observiq/bindplane-agent/exporter/googlecloudexporter

go 1.22.7

require (
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/collector v0.49.0
	github.com/observiq/bindplane-agent/internal/version v0.0.0-20240717172323-a1d3e6ed4aab
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/googlecloudexporter v0.114.0
	github.com/stretchr/testify v1.10.0
	go.opentelemetry.io/collector/component v0.114.0
	go.opentelemetry.io/collector/consumer v0.114.0
	go.opentelemetry.io/collector/exporter v0.114.0
	go.opentelemetry.io/collector/exporter/exportertest v0.114.0
	go.opentelemetry.io/collector/pdata v1.20.0
	go.opentelemetry.io/collector/processor v0.114.0
	go.opentelemetry.io/collector/processor/batchprocessor v0.114.0
	go.uber.org/multierr v1.11.0
	google.golang.org/api v0.209.0
)

require (
	cloud.google.com/go/logging v1.12.0 // indirect
	cloud.google.com/go/monitoring v1.21.2 // indirect
	cloud.google.com/go/trace v1.11.2 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace v1.25.0 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/googleapis/gax-go/v2 v2.14.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/knadh/koanf v1.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/otel v1.32.0
	go.opentelemetry.io/otel/metric v1.32.0 // indirect
	go.opentelemetry.io/otel/sdk v1.32.0 // indirect
	go.opentelemetry.io/otel/trace v1.32.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/zap v1.27.0
	golang.org/x/net v0.31.0 // indirect
	golang.org/x/oauth2 v0.24.0 // indirect
	golang.org/x/sync v0.9.0 // indirect
	golang.org/x/text v0.20.0 // indirect
	google.golang.org/genproto v0.0.0-20241113202542-65e8d215514f // indirect
	google.golang.org/grpc v1.67.1 // indirect
	google.golang.org/protobuf v1.35.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

require (
	cloud.google.com/go v0.116.0 // indirect
	cloud.google.com/go/auth v0.10.2 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.5 // indirect
	cloud.google.com/go/compute/metadata v0.5.2 // indirect
	cloud.google.com/go/longrunning v0.6.2 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/internal/resourcemapping v0.49.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.8.0 // indirect
	github.com/google/s2a-go v0.1.8 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.4 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/tinylru v1.2.1 // indirect
	github.com/tidwall/wal v1.1.7 // indirect
	go.opentelemetry.io/collector/client v1.20.0 // indirect
	go.opentelemetry.io/collector/component/componenttest v0.114.0 // indirect
	go.opentelemetry.io/collector/config/configretry v1.20.0 // indirect
	go.opentelemetry.io/collector/config/configtelemetry v0.114.0 // indirect
	go.opentelemetry.io/collector/consumer/consumererror v0.114.0 // indirect
	go.opentelemetry.io/collector/consumer/consumerprofiles v0.114.0 // indirect
	go.opentelemetry.io/collector/consumer/consumertest v0.114.0 // indirect
	go.opentelemetry.io/collector/exporter/exporterprofiles v0.114.0 // indirect
	go.opentelemetry.io/collector/extension v0.114.0 // indirect
	go.opentelemetry.io/collector/extension/experimental/storage v0.114.0 // indirect
	go.opentelemetry.io/collector/featuregate v1.20.0 // indirect
	go.opentelemetry.io/collector/pdata/pprofile v0.114.0 // indirect
	go.opentelemetry.io/collector/pipeline v0.114.0 // indirect
	go.opentelemetry.io/collector/receiver v0.114.0 // indirect
	go.opentelemetry.io/collector/receiver/receiverprofiles v0.114.0 // indirect
	go.opentelemetry.io/collector/receiver/receivertest v0.114.0 // indirect
	go.opentelemetry.io/collector/semconv v0.114.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.56.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.56.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.32.0 // indirect
	golang.org/x/crypto v0.29.0 // indirect
	golang.org/x/sys v0.27.0 // indirect
	golang.org/x/time v0.8.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20241104194629-dd2ea8efbc28 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241113202542-65e8d215514f // indirect
	google.golang.org/grpc/stats/opentelemetry v0.0.0-20241028142157-ada6787961b3 // indirect
)

// Needed until 'version' package changes are included in a release
replace github.com/observiq/bindplane-agent/internal/version v1.56.0 => github.com/observiq/bindplane-agent/internal/version v0.0.0-20240717172323-a1d3e6ed4aab
