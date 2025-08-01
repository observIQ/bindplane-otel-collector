module github.com/observiq/bindplane-otel-collector

go 1.24.4

require (
	github.com/google/uuid v1.6.0
	github.com/observiq/bindplane-otel-collector/exporter/azureblobexporter v1.81.0
	github.com/observiq/bindplane-otel-collector/exporter/chronicleexporter v1.81.0
	github.com/observiq/bindplane-otel-collector/exporter/chronicleforwarderexporter v1.81.0
	github.com/observiq/bindplane-otel-collector/exporter/googlecloudexporter v1.81.0
	github.com/observiq/bindplane-otel-collector/exporter/googlecloudstorageexporter v1.81.0
	github.com/observiq/bindplane-otel-collector/exporter/googlemanagedprometheusexporter v1.81.0
	github.com/observiq/bindplane-otel-collector/exporter/qradar v1.81.0
	github.com/observiq/bindplane-otel-collector/exporter/snowflakeexporter v1.81.0
	github.com/observiq/bindplane-otel-collector/exporter/webhookexporter v1.81.0
	github.com/observiq/bindplane-otel-collector/internal/measurements v1.81.0
	github.com/observiq/bindplane-otel-collector/internal/report v1.81.0
	github.com/observiq/bindplane-otel-collector/packagestate v1.81.0
	github.com/observiq/bindplane-otel-collector/processor/datapointcountprocessor v1.81.0
	github.com/observiq/bindplane-otel-collector/processor/logcountprocessor v1.81.0
	github.com/observiq/bindplane-otel-collector/processor/lookupprocessor v1.81.0
	github.com/observiq/bindplane-otel-collector/processor/maskprocessor v1.81.0
	github.com/observiq/bindplane-otel-collector/processor/metricextractprocessor v1.81.0
	github.com/observiq/bindplane-otel-collector/processor/metricstatsprocessor v1.81.0
	github.com/observiq/bindplane-otel-collector/processor/randomfailureprocessor v1.81.0
	github.com/observiq/bindplane-otel-collector/processor/removeemptyvaluesprocessor v1.81.0
	github.com/observiq/bindplane-otel-collector/processor/resourceattributetransposerprocessor v1.81.0
	github.com/observiq/bindplane-otel-collector/processor/samplingprocessor v1.81.0
	github.com/observiq/bindplane-otel-collector/processor/spancountprocessor v1.81.0
	github.com/observiq/bindplane-otel-collector/processor/throughputmeasurementprocessor v1.81.0
	github.com/observiq/bindplane-otel-collector/processor/unrollprocessor v1.81.0
	github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver v1.81.0
	github.com/observiq/bindplane-otel-collector/receiver/awss3rehydrationreceiver v1.81.0
	github.com/observiq/bindplane-otel-collector/receiver/azureblobrehydrationreceiver v1.81.0
	github.com/observiq/bindplane-otel-collector/receiver/googlecloudstoragerehydrationreceiver v1.81.0
	github.com/observiq/bindplane-otel-collector/receiver/httpreceiver v1.81.0
	github.com/observiq/bindplane-otel-collector/receiver/m365receiver v1.81.0
	github.com/observiq/bindplane-otel-collector/receiver/oktareceiver v1.81.0
	github.com/observiq/bindplane-otel-collector/receiver/pluginreceiver v1.81.0
	github.com/observiq/bindplane-otel-collector/receiver/routereceiver v1.81.0
	github.com/observiq/bindplane-otel-collector/receiver/sapnetweaverreceiver v1.81.0
	github.com/observiq/bindplane-otel-collector/receiver/splunksearchapireceiver v1.81.0
	github.com/observiq/bindplane-otel-collector/receiver/telemetrygeneratorreceiver v1.81.0
	github.com/observiq/bindplane-otel-collector/receiver/windowseventtracereceiver v1.81.0
	github.com/oklog/ulid/v2 v2.1.1
	github.com/open-telemetry/opamp-go v0.20.0
	github.com/open-telemetry/opentelemetry-collector-contrib/connector/countconnector v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/connector/failoverconnector v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/connector/roundrobinconnector v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/connector/routingconnector v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/connector/servicegraphconnector v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/connector/signaltometricsconnector v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/connector/spanmetricsconnector v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/alibabacloudlogserviceexporter v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/awscloudwatchlogsexporter v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/awsemfexporter v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/awskinesisexporter v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/awss3exporter v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/awsxrayexporter v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/azuremonitorexporter v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/carbonexporter v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickhouseexporter v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/coralogixexporter v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/elasticsearchexporter v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/fileexporter v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/googlecloudpubsubexporter v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/influxdbexporter v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/kafkaexporter v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/loadbalancingexporter v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/logzioexporter v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/lokiexporter v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/opencensusexporter v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/prometheusexporter v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/prometheusremotewriteexporter v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/sapmexporter v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/signalfxexporter v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/splunkhecexporter v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/sumologicexporter v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/syslogexporter v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/zipkinexporter v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/basicauthextension v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/bearertokenauthextension v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/cgroupruntimeextension v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/headerssetterextension v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/healthcheckextension v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/oauth2clientauthextension v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/oidcauthextension v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/pprofextension v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/sigv4authextension v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/storage/filestorage v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/golden v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/attributesprocessor v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/cumulativetodeltaprocessor v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/deltatorateprocessor v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/filterprocessor v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/geoipprocessor v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/groupbyattrsprocessor v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/groupbytraceprocessor v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/k8sattributesprocessor v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/logdedupprocessor v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/logstransformprocessor v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/metricsgenerationprocessor v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/metricstransformprocessor v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/probabilisticsamplerprocessor v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/resourcedetectionprocessor v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/resourceprocessor v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/routingprocessor v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/spanprocessor v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/tailsamplingprocessor v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/transformprocessor v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/activedirectorydsreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/aerospikereceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/apachereceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/apachesparkreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/awscloudwatchreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/awscontainerinsightreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/awsecscontainermetricsreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/awsfirehosereceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/awss3receiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/awsxrayreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/azureblobreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/azureeventhubreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/bigipreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/carbonreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/cloudflarereceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/cloudfoundryreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/collectdreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/couchdbreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/dockerstatsreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/elasticsearchreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/filelogreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/filestatsreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/flinkmetricsreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/fluentforwardreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/googlecloudpubsubreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/googlecloudspannerreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/httpcheckreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/iisreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/influxdbreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/jaegerreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/jmxreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/journaldreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/k8sclusterreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/k8seventsreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkametricsreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kafkareceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kubeletstatsreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/memcachedreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/mongodbatlasreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/mongodbreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/mysqlreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/nginxreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/opencensusreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/podmanreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/postgresqlreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/prometheusreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/rabbitmqreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/redisreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/riakreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/saphanareceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/sapmreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/simpleprometheusreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/snmpreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/splunkhecreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/sqlqueryreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/sqlserverreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/statsdreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/syslogreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/tcplogreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/udplogreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/vcenterreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/webhookeventreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/windowseventlogreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/windowsperfcountersreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/zipkinreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/zookeeperreceiver v0.130.0
	github.com/shirou/gopsutil/v3 v3.24.5
	github.com/spf13/pflag v1.0.6
	github.com/stretchr/testify v1.10.0
	go.opentelemetry.io/collector/component v1.36.1
	go.opentelemetry.io/collector/confmap v1.36.1
	go.opentelemetry.io/collector/confmap/provider/envprovider v1.36.1
	go.opentelemetry.io/collector/confmap/provider/fileprovider v1.36.1
	go.opentelemetry.io/collector/confmap/provider/httpsprovider v1.36.1
	go.opentelemetry.io/collector/confmap/provider/yamlprovider v1.36.1
	go.opentelemetry.io/collector/connector v0.130.1
	go.opentelemetry.io/collector/connector/forwardconnector v0.130.1
	go.opentelemetry.io/collector/consumer v1.36.1
	go.opentelemetry.io/collector/exporter v0.130.1
	go.opentelemetry.io/collector/exporter/debugexporter v0.130.1
	go.opentelemetry.io/collector/exporter/nopexporter v0.130.1
	go.opentelemetry.io/collector/exporter/otlpexporter v0.130.1
	go.opentelemetry.io/collector/exporter/otlphttpexporter v0.130.1
	go.opentelemetry.io/collector/extension v1.36.1
	go.opentelemetry.io/collector/extension/zpagesextension v0.130.1
	go.opentelemetry.io/collector/featuregate v1.36.1
	go.opentelemetry.io/collector/otelcol v0.130.1
	go.opentelemetry.io/collector/pdata v1.36.1
	go.opentelemetry.io/collector/processor v1.36.1
	go.opentelemetry.io/collector/processor/batchprocessor v0.130.1
	go.opentelemetry.io/collector/processor/memorylimiterprocessor v0.130.1
	go.opentelemetry.io/collector/receiver v1.36.1
	go.opentelemetry.io/collector/receiver/otlpreceiver v0.130.1
	go.uber.org/multierr v1.11.0
	go.uber.org/zap v1.27.0
	golang.org/x/sys v0.34.0
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/observiq/bindplane-otel-collector/exporter/azureloganalyticsexporter v1.81.0
	github.com/observiq/bindplane-otel-collector/extension/awss3eventextension v1.81.0
	github.com/observiq/bindplane-otel-collector/processor/topologyprocessor v1.81.0
	github.com/observiq/bindplane-otel-collector/receiver/bindplaneauditlogs v1.81.0
	github.com/open-telemetry/opentelemetry-collector-contrib/confmap/provider/aesprovider v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/otelarrowexporter v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/encoding/avrologencodingextension v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/encoding/jsonlogencodingextension v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/storage/redisstorageextension v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/deltatocumulativeprocessor v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/intervalprocessor v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/redactionprocessor v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/githubreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/libhoneyreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/netflowreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/otelarrowreceiver v0.130.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/otlpjsonfilereceiver v0.130.0
	go.opentelemetry.io/collector/extension/extensiontest v0.130.1
	go.opentelemetry.io/collector/processor/processorhelper v0.130.1
	go.opentelemetry.io/collector/processor/processortest v0.130.1
)

require (
	cel.dev/expr v0.23.0 // indirect
	cloud.google.com/go/auth v0.16.2 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/storage v1.53.0 // indirect
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.18.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.10.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.11.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/monitor/ingestion/azlogs v1.0.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5 v5.7.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v4 v4.3.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/storage/azblob v1.6.1 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.4.2 // indirect
	github.com/BurntSushi/toml v1.4.0 // indirect
	github.com/ClickHouse/ch-go v0.66.1 // indirect
	github.com/ClickHouse/clickhouse-go/v2 v2.37.2 // indirect
	github.com/Code-Hex/go-generics-cache v1.5.1 // indirect
	github.com/DataDog/agent-payload/v5 v5.0.157 // indirect
	github.com/DataDog/datadog-agent/comp/core/config v0.67.0 // indirect
	github.com/DataDog/datadog-agent/comp/core/flare/builder v0.67.0 // indirect
	github.com/DataDog/datadog-agent/comp/core/flare/types v0.67.0 // indirect
	github.com/DataDog/datadog-agent/comp/core/hostname/hostnameinterface v0.67.0 // indirect
	github.com/DataDog/datadog-agent/comp/core/log/def v0.67.0 // indirect
	github.com/DataDog/datadog-agent/comp/core/secrets v0.67.0 // indirect
	github.com/DataDog/datadog-agent/comp/core/status v0.67.0 // indirect
	github.com/DataDog/datadog-agent/comp/core/tagger/origindetection v0.67.0 // indirect
	github.com/DataDog/datadog-agent/comp/core/telemetry v0.67.0 // indirect
	github.com/DataDog/datadog-agent/comp/def v0.67.0 // indirect
	github.com/DataDog/datadog-agent/comp/forwarder/defaultforwarder v0.67.0 // indirect
	github.com/DataDog/datadog-agent/comp/forwarder/orchestrator/orchestratorinterface v0.67.0 // indirect
	github.com/DataDog/datadog-agent/comp/logs/agent/config v0.67.0 // indirect
	github.com/DataDog/datadog-agent/comp/otelcol/logsagentpipeline v0.67.0-devel.0.20250507144401-a64863d1e01a // indirect
	github.com/DataDog/datadog-agent/comp/otelcol/logsagentpipeline/logsagentpipelineimpl v0.67.0-devel.0.20250507144401-a64863d1e01a // indirect
	github.com/DataDog/datadog-agent/comp/otelcol/otlp/components/exporter/logsagentexporter v0.67.0-devel.0.20250507144401-a64863d1e01a // indirect
	github.com/DataDog/datadog-agent/comp/otelcol/otlp/components/exporter/serializerexporter v0.69.0-devel.0.20250630223051-02255cc64e1f // indirect
	github.com/DataDog/datadog-agent/comp/otelcol/otlp/components/metricsclient v0.67.0 // indirect
	github.com/DataDog/datadog-agent/comp/serializer/logscompression v0.67.0 // indirect
	github.com/DataDog/datadog-agent/comp/serializer/metricscompression v0.67.0 // indirect
	github.com/DataDog/datadog-agent/comp/trace/compression/def v0.67.0 // indirect
	github.com/DataDog/datadog-agent/comp/trace/compression/impl-gzip v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/aggregator/ckey v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/api v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/collector/check/defaults v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/config/create v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/config/env v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/config/mock v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/config/model v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/config/nodetreemodel v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/config/setup v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/config/structure v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/config/teeconfig v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/config/utils v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/config/viperconfig v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/fips v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/logs/auditor v0.67.0-devel.0.20250507144401-a64863d1e01a // indirect
	github.com/DataDog/datadog-agent/pkg/logs/client v0.67.0-devel.0.20250507144401-a64863d1e01a // indirect
	github.com/DataDog/datadog-agent/pkg/logs/diagnostic v0.67.0-devel.0.20250507144401-a64863d1e01a // indirect
	github.com/DataDog/datadog-agent/pkg/logs/message v0.67.0-devel.0.20250507144401-a64863d1e01a // indirect
	github.com/DataDog/datadog-agent/pkg/logs/metrics v0.67.0-devel.0.20250507144401-a64863d1e01a // indirect
	github.com/DataDog/datadog-agent/pkg/logs/pipeline v0.67.0-devel.0.20250507144401-a64863d1e01a // indirect
	github.com/DataDog/datadog-agent/pkg/logs/processor v0.67.0-devel.0.20250507144401-a64863d1e01a // indirect
	github.com/DataDog/datadog-agent/pkg/logs/sds v0.67.0-devel.0.20250507144401-a64863d1e01a // indirect
	github.com/DataDog/datadog-agent/pkg/logs/sender v0.67.0-devel.0.20250507144401-a64863d1e01a // indirect
	github.com/DataDog/datadog-agent/pkg/logs/sources v0.67.0-devel.0.20250507144401-a64863d1e01a // indirect
	github.com/DataDog/datadog-agent/pkg/logs/status/statusinterface v0.67.0-devel.0.20250507144401-a64863d1e01a // indirect
	github.com/DataDog/datadog-agent/pkg/logs/status/utils v0.67.0-devel.0.20250507144401-a64863d1e01a // indirect
	github.com/DataDog/datadog-agent/pkg/metrics v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/obfuscate v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/orchestrator/model v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/process/util/api v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/proto v0.69.0-devel // indirect
	github.com/DataDog/datadog-agent/pkg/remoteconfig/state v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/serializer v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/status/health v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/tagger/types v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/tagset v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/telemetry v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/template v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/trace v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/util/backoff v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/util/buf v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/util/cgroups v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/util/common v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/util/compression v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/util/executable v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/util/filesystem v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/util/fxutil v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/util/hostname/validate v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/util/http v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/util/json v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/util/log v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/util/option v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/util/otel v0.66.1 // indirect
	github.com/DataDog/datadog-agent/pkg/util/pointer v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/util/scrubber v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/util/sort v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/util/startstop v0.66.1 // indirect
	github.com/DataDog/datadog-agent/pkg/util/statstracker v0.66.1 // indirect
	github.com/DataDog/datadog-agent/pkg/util/system v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/util/system/socket v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/util/winutil v0.67.0 // indirect
	github.com/DataDog/datadog-agent/pkg/version v0.67.0 // indirect
	github.com/DataDog/datadog-api-client-go/v2 v2.42.0 // indirect
	github.com/DataDog/datadog-go/v5 v5.6.0 // indirect
	github.com/DataDog/dd-sensitive-data-scanner/sds-go/go v0.0.0-20240816154533-f7f9beb53a42 // indirect
	github.com/DataDog/go-sqllexer v0.1.6 // indirect
	github.com/DataDog/go-tuf v1.1.0-0.5.2 // indirect
	github.com/DataDog/gohai v0.0.0-20230524154621-4316413895ee // indirect
	github.com/DataDog/mmh3 v0.0.0-20210722141835-012dc69a9e49 // indirect
	github.com/DataDog/opentelemetry-mapping-go/pkg/inframetadata v0.29.1 // indirect
	github.com/DataDog/opentelemetry-mapping-go/pkg/otlp/attributes v0.29.1 // indirect
	github.com/DataDog/opentelemetry-mapping-go/pkg/otlp/logs v0.29.1 // indirect
	github.com/DataDog/opentelemetry-mapping-go/pkg/otlp/metrics v0.29.1 // indirect
	github.com/DataDog/opentelemetry-mapping-go/pkg/quantile v0.29.1 // indirect
	github.com/DataDog/sketches-go v1.4.7 // indirect
	github.com/DataDog/viper v1.14.1-0.20250612143030-1b15c8822ed4 // indirect
	github.com/DataDog/zstd v1.5.6 // indirect
	github.com/DataDog/zstd_0 v0.0.0-20210310093942-586c1286621f // indirect
	github.com/GehirnInc/crypt v0.0.0-20230320061759-8cc1b52080c5 // indirect
	github.com/GoogleCloudPlatform/grpc-gcp-go/grpcgcp v1.5.3 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric v0.51.0 // indirect
	github.com/HdrHistogram/hdrhistogram-go v1.1.2 // indirect
	github.com/IBM/sarama v1.45.2 // indirect
	github.com/Khan/genqlient v0.8.1 // indirect
	github.com/KimMachineGun/automemlimit v0.7.3 // indirect
	github.com/aerospike/aerospike-client-go/v8 v8.2.2 // indirect
	github.com/antchfx/xmlquery v1.4.4 // indirect
	github.com/antchfx/xpath v1.3.4 // indirect
	github.com/apache/arrow-go/v18 v18.2.0 // indirect
	github.com/apache/arrow/go/v16 v16.1.0 // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/aws/aws-lambda-go v1.48.0 // indirect
	github.com/aws/aws-msk-iam-sasl-signer-go v1.0.4 // indirect
	github.com/aws/aws-sdk-go-v2/feature/s3/manager v1.17.83 // indirect
	github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs v1.51.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.230.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/servicediscovery v1.35.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/sqs v1.38.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/xray v1.31.7 // indirect
	github.com/axiomhq/hyperloglog v0.0.0-20230201085229-3ddf4bad03dc // indirect
	github.com/bboreham/go-loser v0.0.0-20230920113527-fcc2c21820a3 // indirect
	github.com/benbjohnson/clock v1.3.5 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/briandowns/spinner v1.23.0 // indirect
	github.com/cenkalti/backoff/v5 v5.0.2 // indirect
	github.com/cespare/xxhash v1.1.0 // indirect
	github.com/cihub/seelog v0.0.0-20170130134532-f561c5e57575 // indirect
	github.com/containerd/containerd/api v1.9.0 // indirect
	github.com/containerd/errdefs v1.0.0 // indirect
	github.com/containerd/errdefs/pkg v0.3.0 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/containerd/typeurl/v2 v2.2.3 // indirect
	github.com/coreos/go-oidc/v3 v3.14.1 // indirect
	github.com/dgryski/go-metro v0.0.0-20180109044635-280f6062b5bc // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/dimchansky/utfbom v1.1.1 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/ebitengine/purego v0.8.4 // indirect
	github.com/edsrzf/mmap-go v1.2.0 // indirect
	github.com/elastic/elastic-transport-go/v8 v8.7.0 // indirect
	github.com/elastic/go-docappender/v2 v2.10.0 // indirect
	github.com/elastic/go-elasticsearch/v8 v8.18.1 // indirect
	github.com/elastic/go-freelru v0.16.0 // indirect
	github.com/elastic/go-grok v0.3.1 // indirect
	github.com/elastic/go-windows v1.0.2 // indirect
	github.com/elastic/lunes v0.1.0 // indirect
	github.com/envoyproxy/go-control-plane/envoy v1.32.4 // indirect
	github.com/expr-lang/expr v1.17.5 // indirect
	github.com/facette/natsort v0.0.0-20181210072756-2cd4dd1e2dcb // indirect
	github.com/foxboron/go-tpm-keyfiles v0.0.0-20250323135004-b31fac66206e // indirect
	github.com/fxamacker/cbor/v2 v2.7.0 // indirect
	github.com/go-faster/city v1.0.1 // indirect
	github.com/go-faster/errors v0.7.1 // indirect
	github.com/go-jose/go-jose/v3 v3.0.4 // indirect
	github.com/go-jose/go-jose/v4 v4.0.5 // indirect
	github.com/go-openapi/analysis v0.23.0 // indirect
	github.com/go-openapi/errors v0.22.0 // indirect
	github.com/go-openapi/loads v0.22.0 // indirect
	github.com/go-openapi/spec v0.21.0 // indirect
	github.com/go-openapi/strfmt v0.23.0 // indirect
	github.com/go-openapi/validate v0.24.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.3.0 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/goccy/go-yaml v1.18.0 // indirect
	github.com/gofrs/flock v0.12.1 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.2 // indirect
	github.com/golang/mock v1.7.0-rc.1 // indirect
	github.com/google/gnostic-models v0.6.8 // indirect
	github.com/google/go-github/v72 v72.0.0 // indirect
	github.com/google/go-tpm v0.9.5 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/gophercloud/gophercloud/v2 v2.7.0 // indirect
	github.com/grafana/loki/pkg/push v0.0.0-20240514112848-a1b1eeb09583 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/hashicorp/hcl v1.0.1-vault-7 // indirect
	github.com/hectane/go-acl v0.0.0-20230122075934-ca0b05cb1adb // indirect
	github.com/hetznercloud/hcloud-go/v2 v2.21.1 // indirect
	github.com/influxdata/influxdb-observability/otel2influx v0.5.12 // indirect
	github.com/itchyny/timefmt-go v0.1.6 // indirect
	github.com/jaegertracing/jaeger-idl v0.5.0 // indirect
	github.com/jcmturner/goidentity/v6 v6.0.1 // indirect
	github.com/jellydator/ttlcache/v3 v3.4.0 // indirect
	github.com/jmoiron/sqlx v1.4.0 // indirect
	github.com/jonboulle/clockwork v0.5.0 // indirect
	github.com/julienschmidt/httprouter v1.3.0 // indirect
	github.com/kelseyhightower/envconfig v1.4.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.10 // indirect
	github.com/knadh/koanf/v2 v2.2.1 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/leodido/go-syslog/v4 v4.2.0 // indirect
	github.com/lestrrat-go/strftime v1.1.0 // indirect
	github.com/libp2p/go-reuseport v0.4.0 // indirect
	github.com/linkedin/goavro/v2 v2.14.0 // indirect
	github.com/magefile/mage v1.15.0 // indirect
	github.com/magiconair/properties v1.8.10 // indirect
	github.com/mdlayher/socket v0.4.1 // indirect
	github.com/mdlayher/vsock v1.2.1 // indirect
	github.com/microsoft/go-mssqldb v1.9.2 // indirect
	github.com/minio/sha256-simd v1.0.1 // indirect
	github.com/mitchellh/hashstructure/v2 v2.0.2 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/sys/userns v0.1.0 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/netsampler/goflow2/v2 v2.2.3 // indirect
	github.com/nginx/nginx-prometheus-exporter v1.4.1 // indirect
	github.com/nxadm/tail v1.4.11 // indirect
	github.com/observiq/bindplane-otel-collector/counter v1.81.0 // indirect
	github.com/observiq/bindplane-otel-collector/expr v1.81.0 // indirect
	github.com/observiq/bindplane-otel-collector/internal/aws v1.81.0 // indirect
	github.com/observiq/bindplane-otel-collector/internal/rehydration v1.81.0 // indirect
	github.com/observiq/bindplane-otel-collector/internal/storageclient v1.81.0 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/okta/okta-sdk-golang/v2 v2.20.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/googlemanagedprometheusexporter v0.130.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/ackextension v0.130.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/encoding v0.130.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/encoding/awscloudwatchmetricstreamsencodingextension v0.130.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/k8sleaderelector v0.130.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/opampcustommessages v0.130.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/sumologicextension v0.130.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/collectd v0.130.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/datadog v0.130.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/exp/metrics v0.130.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/gopsutilenv v0.130.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/grpcutil v0.130.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/kafka v0.130.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/otelarrow v0.130.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/pdatautil v0.130.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/sqlquery v0.130.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/core/xidutils v0.130.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/datadog v0.130.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/kafka/configkafka v0.130.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/kafka/topic v0.130.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/sampling v0.130.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/azure v0.130.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/azurelogs v0.130.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/scraper/zookeeperscraper v0.130.0 // indirect
	github.com/open-telemetry/otel-arrow/go v0.39.0 // indirect
	github.com/opencontainers/cgroups v0.0.2 // indirect
	github.com/oschwald/geoip2-golang v1.13.0 // indirect
	github.com/oschwald/maxminddb-golang v1.13.0 // indirect
	github.com/outcaste-io/ristretto v0.2.3 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/paulmach/orb v0.11.1 // indirect
	github.com/pbnjay/memory v0.0.0-20210728143218-7b4eea64cf58 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10 // indirect
	github.com/prometheus/alertmanager v0.28.1 // indirect
	github.com/prometheus/common/assets v0.2.0 // indirect
	github.com/prometheus/exporter-toolkit v0.14.0 // indirect
	github.com/prometheus/otlptranslator v0.0.0-20250717125610-8549f4ab4f8f // indirect
	github.com/prometheus/sigv4 v0.2.0 // indirect
	github.com/puzpuzpuz/xsync/v3 v3.5.1 // indirect
	github.com/rdforte/gomaxecs v1.1.1 // indirect
	github.com/redis/go-redis/v9 v9.11.0 // indirect
	github.com/richardartoul/molecule v1.0.1-0.20240531184615-7ca0df43c0b3 // indirect
	github.com/secure-systems-lab/go-securesystemslib v0.9.0 // indirect
	github.com/segmentio/asm v1.2.0 // indirect
	github.com/shirou/gopsutil/v4 v4.25.6 // indirect
	github.com/shoenig/go-m1cpu v0.1.6 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/shurcooL/httpfs v0.0.0-20230704072500-f1e31cf0ba5c // indirect
	github.com/spf13/afero v1.14.0 // indirect
	github.com/spf13/cast v1.9.2 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spiffe/go-spiffe/v2 v2.5.0 // indirect
	github.com/stackitcloud/stackit-sdk-go/core v0.17.2 // indirect
	github.com/stormcat24/protodep v0.1.8 // indirect
	github.com/tg123/go-htpasswd v1.2.4 // indirect
	github.com/thda/tds v0.1.7 // indirect
	github.com/tilinna/clock v1.1.0 // indirect
	github.com/twmb/franz-go v1.18.1 // indirect
	github.com/twmb/franz-go/pkg/kmsg v1.11.2 // indirect
	github.com/twmb/franz-go/pkg/sasl/kerberos v1.1.0 // indirect
	github.com/twmb/franz-go/plugin/kzap v1.1.2 // indirect
	github.com/twmb/murmur3 v1.1.8 // indirect
	github.com/ua-parser/uap-go v0.0.0-20240611065828-3a4781585db6 // indirect
	github.com/valyala/fastjson v1.6.4 // indirect
	github.com/vektah/gqlparser/v2 v2.5.22 // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	github.com/wadey/gocovmerge v0.0.0-20160331181800-b5bfa59ec0ad // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/zeebo/errs v1.4.0 // indirect
	github.com/zeebo/xxh3 v1.0.2 // indirect
	go.elastic.co/fastjson v1.5.0 // indirect
	go.mongodb.org/mongo-driver/v2 v2.2.2 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/collector v0.130.1 // indirect
	go.opentelemetry.io/collector/client v1.36.1 // indirect
	go.opentelemetry.io/collector/component/componentstatus v0.130.1 // indirect
	go.opentelemetry.io/collector/component/componenttest v0.130.1 // indirect
	go.opentelemetry.io/collector/config/configauth v0.130.1 // indirect
	go.opentelemetry.io/collector/config/configcompression v1.36.1 // indirect
	go.opentelemetry.io/collector/config/configgrpc v0.130.1 // indirect
	go.opentelemetry.io/collector/config/confighttp v0.130.1 // indirect
	go.opentelemetry.io/collector/config/configmiddleware v0.130.1 // indirect
	go.opentelemetry.io/collector/config/confignet v1.36.1 // indirect
	go.opentelemetry.io/collector/config/configopaque v1.36.1 // indirect
	go.opentelemetry.io/collector/config/configoptional v0.130.1 // indirect
	go.opentelemetry.io/collector/config/configretry v1.36.1 // indirect
	go.opentelemetry.io/collector/config/configtelemetry v0.130.1 // indirect
	go.opentelemetry.io/collector/config/configtls v1.36.1 // indirect
	go.opentelemetry.io/collector/confmap/xconfmap v0.130.1 // indirect
	go.opentelemetry.io/collector/connector/connectortest v0.130.1 // indirect
	go.opentelemetry.io/collector/connector/xconnector v0.130.1 // indirect
	go.opentelemetry.io/collector/consumer/consumererror v0.130.1 // indirect
	go.opentelemetry.io/collector/consumer/consumererror/xconsumererror v0.130.1 // indirect
	go.opentelemetry.io/collector/consumer/consumertest v0.130.1 // indirect
	go.opentelemetry.io/collector/consumer/xconsumer v0.130.1 // indirect
	go.opentelemetry.io/collector/exporter/exporterhelper/xexporterhelper v0.130.1 // indirect
	go.opentelemetry.io/collector/exporter/exportertest v0.130.1 // indirect
	go.opentelemetry.io/collector/exporter/xexporter v0.130.1 // indirect
	go.opentelemetry.io/collector/extension/extensionauth v1.36.1 // indirect
	go.opentelemetry.io/collector/extension/extensioncapabilities v0.130.1 // indirect
	go.opentelemetry.io/collector/extension/extensionmiddleware v0.130.1 // indirect
	go.opentelemetry.io/collector/extension/xextension v0.130.1 // indirect
	go.opentelemetry.io/collector/filter v0.130.1 // indirect
	go.opentelemetry.io/collector/internal/fanoutconsumer v0.130.1 // indirect
	go.opentelemetry.io/collector/internal/memorylimiter v0.130.1 // indirect
	go.opentelemetry.io/collector/internal/sharedcomponent v0.130.1 // indirect
	go.opentelemetry.io/collector/internal/telemetry v0.130.1 // indirect
	go.opentelemetry.io/collector/pdata/pprofile v0.130.1 // indirect
	go.opentelemetry.io/collector/pdata/testdata v0.130.1 // indirect
	go.opentelemetry.io/collector/pdata/xpdata v0.130.1 // indirect
	go.opentelemetry.io/collector/pipeline v0.130.1 // indirect
	go.opentelemetry.io/collector/pipeline/xpipeline v0.130.1 // indirect
	go.opentelemetry.io/collector/processor/processorhelper/xprocessorhelper v0.130.1 // indirect
	go.opentelemetry.io/collector/processor/xprocessor v0.130.1 // indirect
	go.opentelemetry.io/collector/receiver/receiverhelper v0.130.1 // indirect
	go.opentelemetry.io/collector/receiver/receivertest v0.130.1 // indirect
	go.opentelemetry.io/collector/receiver/xreceiver v0.130.1 // indirect
	go.opentelemetry.io/collector/scraper v0.130.1 // indirect
	go.opentelemetry.io/collector/scraper/scraperhelper v0.130.1 // indirect
	go.opentelemetry.io/collector/service v0.130.1 // indirect
	go.opentelemetry.io/collector/service/hostcapabilities v0.130.1 // indirect
	go.opentelemetry.io/contrib/bridges/otelzap v0.12.0 // indirect
	go.opentelemetry.io/contrib/detectors/gcp v1.36.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace v0.61.0 // indirect
	go.opentelemetry.io/contrib/otelconf v0.17.0 // indirect
	go.opentelemetry.io/ebpf-profiler v0.0.0-20250212075250-7bf12d3f962f // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc v0.13.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp v0.13.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.37.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.37.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.37.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.37.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.37.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutlog v0.13.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.37.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.37.0 // indirect
	go.opentelemetry.io/otel/log v0.13.0 // indirect
	go.opentelemetry.io/otel/sdk/log v0.13.0 // indirect
	go.opentelemetry.io/proto/otlp v1.7.0 // indirect
	go.uber.org/automaxprocs v1.6.0 // indirect
	go.uber.org/dig v1.19.0 // indirect
	go.uber.org/fx v1.24.0 // indirect
	go.uber.org/goleak v1.3.0 // indirect
	go.uber.org/zap/exp v0.3.0 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250603155806-513f23925822 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250603155806-513f23925822 // indirect
	gopkg.in/evanphx/json-patch.v4 v4.12.0 // indirect
	gopkg.in/zorkian/go-datadog-api.v2 v2.30.0 // indirect
	sigs.k8s.io/controller-runtime v0.20.4 // indirect
)

require (
	github.com/Azure/azure-amqp-common-go/v4 v4.2.0 // indirect
	github.com/bmatcuk/doublestar/v4 v4.8.1 // indirect
	github.com/hooklift/gowsdl v0.5.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatautil v0.130.0 // indirect; indir2ct
	github.com/ovh/go-ovh v1.8.0 // indirect
	github.com/relvacode/iso8601 v1.6.0 // indirect
)

require (
	github.com/99designs/go-keychain v0.0.0-20191008050251-8e49817e8af4 // indirect
	github.com/99designs/keyring v1.2.2 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.3.36 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.7.4 // indirect
	github.com/danieljoos/wincred v1.2.2 // indirect
	github.com/dvsekhvalnov/jose2go v1.6.0 // indirect
	github.com/godbus/dbus v0.0.0-20190726142602-4481cbc300e2 // indirect
	github.com/gsterjov/go-libsecret v0.0.0-20161001094733-a6f4afe4910c // indirect
	github.com/mtibben/percent v0.2.1 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter v0.130.0 // indirect; indir2ct
)

require (
	cloud.google.com/go v0.121.2 // indirect
	cloud.google.com/go/compute/metadata v0.7.0 // indirect
	cloud.google.com/go/iam v1.5.2 // indirect
	cloud.google.com/go/logging v1.13.0 // indirect
	cloud.google.com/go/longrunning v0.6.7 // indirect
	cloud.google.com/go/monitoring v1.24.2 // indirect
	cloud.google.com/go/pubsub v1.49.0 // indirect
	cloud.google.com/go/spanner v1.83.0 // indirect
	cloud.google.com/go/trace v1.11.6 // indirect
	code.cloudfoundry.org/clock v1.0.0 // indirect
	code.cloudfoundry.org/go-diodes v0.0.0-20241007161556-ec30366c7912 // indirect
	code.cloudfoundry.org/go-loggregator v7.4.0+incompatible // indirect
	code.cloudfoundry.org/rfc5424 v0.0.0-20201103192249-000122071b78 // indirect
	github.com/Azure/azure-event-hubs-go/v3 v3.6.2 // indirect
	github.com/Azure/azure-sdk-for-go v68.0.0+incompatible // indirect
	github.com/Azure/go-amqp v1.0.2 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.11.29 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.23 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.0 // indirect
	github.com/Azure/go-autorest/autorest/to v0.4.0 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.3.1 // indirect
	github.com/Azure/go-autorest/logger v0.2.1 // indirect
	github.com/Azure/go-autorest/tracing v0.6.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp v1.27.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/collector v0.53.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/collector/googlemanagedprometheus v0.52.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace v1.29.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/internal/resourcemapping v0.53.0 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/SAP/go-hdb v1.13.11 // indirect
	github.com/Showmax/go-fqdn v1.0.0 // indirect
	github.com/alecthomas/participle/v2 v2.1.4 // indirect
	github.com/alecthomas/units v0.0.0-20240927000941-0f3dac36c52b // indirect
	github.com/aliyun/aliyun-log-go-sdk v0.1.83 // indirect
	github.com/andybalholm/brotli v1.1.1 // indirect
	github.com/antonmedv/expr v1.15.5 // indirect
	github.com/apache/thrift v0.22.0 // indirect
	github.com/armon/go-metrics v0.4.1 // indirect
	github.com/aws/aws-sdk-go v1.55.7 // indirect
	github.com/aws/aws-sdk-go-v2 v1.36.5 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.6.11 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.29.17 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.17.70 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.32 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.36 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.36 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.12.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.12.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.18.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/kinesis v1.35.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/s3 v1.83.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.25.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.30.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.34.0 // indirect
	github.com/aws/smithy-go v1.22.4 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cenkalti/backoff v2.2.1+incompatible // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/census-instrumentation/opencensus-proto v0.4.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cilium/ebpf v0.17.3 // indirect
	github.com/cloudfoundry-incubator/uaago v0.0.0-20190307164349-8136b7bbe76e // indirect
	github.com/cncf/xds/go v0.0.0-20250326154945-ae57f3c0d45f // indirect
	github.com/containerd/ttrpc v1.2.7 // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/cyphar/filepath-securejoin v0.4.1 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dennwc/varint v1.0.0 // indirect
	github.com/devigned/tab v0.1.1 // indirect
	github.com/digitalocean/godo v1.152.0 // indirect
	github.com/docker/docker v28.2.2+incompatible // indirect
	github.com/docker/go-connections v0.5.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/eapache/go-resiliency v1.7.0 // indirect
	github.com/eapache/go-xerial-snappy v0.0.0-20230731223053-c322873962e3 // indirect
	github.com/eapache/queue v1.1.0 // indirect
	github.com/elastic/go-structform v0.0.12 // indirect
	github.com/emicklei/go-restful/v3 v3.11.0 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.2.1 // indirect
	github.com/euank/go-kmsg-parser v2.0.0+incompatible // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.7 // indirect
	github.com/go-kit/kit v0.13.0 // indirect
	github.com/go-kit/log v0.2.1 // indirect
	github.com/go-logfmt/logfmt v0.6.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/go-resty/resty/v2 v2.16.5 // indirect
	github.com/go-sql-driver/mysql v1.9.3 // indirect
	github.com/go-zookeeper/zk v1.0.4 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	github.com/gofrs/uuid v4.4.0+incompatible // indirect
	github.com/gogo/googleapis v1.4.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.2 // indirect
	github.com/golang-sql/civil v0.0.0-20220223132316-b832511892a9 // indirect
	github.com/golang-sql/sqlexp v0.1.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/golang/snappy v1.0.0
	// (Dakota): cadvisor only works on version `v0.49.1-0.20240628164550-89f779d86055` until they release v0.50.0 with updated docker dep
	// https://github.com/open-telemetry/opentelemetry-collector-contrib/pull/33870
	github.com/google/cadvisor v0.53.0 // indirect
	github.com/google/flatbuffers v25.2.10+incompatible // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.6 // indirect
	github.com/googleapis/gax-go/v2 v2.14.2 // indirect
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/gosnmp/gosnmp v1.41.0 // indirect
	github.com/grafana/regexp v0.0.0-20240518133315-a468a5bfb3bc // indirect
	github.com/grobie/gomemcache v0.0.0-20230213081705-239240bbc445 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.1 // indirect
	github.com/hashicorp/consul/api v1.32.0 // indirect
	github.com/hashicorp/cronexpr v1.1.2 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.7 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	github.com/hashicorp/golang-lru v1.0.2 // indirect
	github.com/hashicorp/nomad/api v0.0.0-20241218080744-e3ac00f30eec // indirect
	github.com/hashicorp/serf v0.10.1 // indirect
	github.com/iancoleman/strcase v0.3.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/influxdata/influxdb-observability/common v0.5.12 // indirect
	github.com/influxdata/influxdb-observability/influx2otel v0.5.12 // indirect
	github.com/influxdata/line-protocol/v2 v2.2.1 // indirect
	github.com/ionos-cloud/sdk-go/v6 v6.3.4 // indirect
	github.com/jaegertracing/jaeger v1.66.0 // indirect
	github.com/jcmturner/aescts/v2 v2.0.0 // indirect
	github.com/jcmturner/dnsutils/v2 v2.0.0 // indirect
	github.com/jcmturner/gofork v1.7.6 // indirect
	github.com/jcmturner/gokrb5/v8 v8.4.4 // indirect
	github.com/jcmturner/rpc/v2 v2.0.3 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/jpillora/backoff v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/karrick/godirwalk v1.17.0 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/knadh/koanf v1.5.0 // indirect
	github.com/kolo/xmlrpc v0.0.0-20220921171641-a4b6fa1dd06b // indirect
	github.com/leodido/ragel-machinery v0.0.0-20190525184631-5f46317e436b // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/lightstep/go-expohisto v1.0.0 // indirect
	github.com/linode/linodego v1.52.1 // indirect
	github.com/lufia/plan9stats v0.0.0-20250317134145-8bc96cf8fc35 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/microsoft/ApplicationInsights-Go v0.4.4 // indirect
	github.com/miekg/dns v1.1.66 // indirect
	github.com/mistifyio/go-zfs v2.1.2-0.20190413222219-f784269be439+incompatible // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.5.1-0.20231216201459-8508981c8b6c // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/sys/mountinfo v0.7.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/mongodb-forks/digest v1.1.0 // indirect
	github.com/mostynb/go-grpc-compression v1.2.3 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/mwitkow/go-conntrack v0.0.0-20190716064945-2f068394615f // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/googlecloudexporter v0.130.0 // indirect; indi72.0
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/aws/awsutil v0.130.0 // indirect; indi72.0
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/aws/containerinsight v0.130.0 // indirect; indi72.0
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/aws/cwlogs v0.130.0 // indirect; indi72.0
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/aws/ecsutil v0.130.0 // indirect; indi72.0
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/aws/k8s v0.130.0 // indirect; indi72.0
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/aws/metrics v0.130.0 // indirect; indi72.0
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/aws/proxy v0.130.0 // indirect; indi72.0
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/aws/xray v0.130.0 // indirect; indi72.0
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/common v0.130.0 // indirect; indi72.0
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal v0.130.0 // indirect; indi72.0
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/docker v0.130.0 // indirect; indi72.0
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/k8sconfig v0.130.0 // indirect; indi72.0
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/kubelet v0.130.0 // indirect; indi72.0
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/metadataproviders v0.130.0 // indirect; indi72.0
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/sharedcomponent v0.130.0 // indirect; indi72.0
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/splunk v0.130.0 // indirect; indi72.0
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/batchperresourceattr v0.130.0 // indirect; indi72.0
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/batchpersignal v0.130.0 // indirect; indi72.0
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/experimentalmetricmetadata v0.130.0 // indirect; indi72.0
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl v0.130.0 // indirect; indi72.0
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/resourcetotelemetry v0.130.0 // indirect; indi72.0
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza v0.130.0 // indirect; indi72.0
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/jaeger v0.130.0 // indirect; indi72.0
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/loki v0.130.0 // indirect; indi72.0
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/opencensus v0.130.0 // indirect; indi72.0
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/prometheus v0.130.0 // indirect; indi72.0
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/prometheusremotewrite v0.130.0 // indirect; indi72.0
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/signalfx v0.130.0 // indirect; indi72.0
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/zipkin v0.130.0 // indirect; indi72.0
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/winperfcounters v0.130.0 // indirect; indi72.0
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/opencontainers/runtime-spec v1.2.1 // indirect
	github.com/openshift/api v3.9.0+incompatible // indirect
	github.com/openshift/client-go v0.0.0-20241203091221-452dfb8fa071 // indirect
	github.com/openzipkin/zipkin-go v0.4.3 // indirect
	github.com/philhofer/fwd v1.1.3-0.20240916144458-20a13a1f6b7c // indirect
	github.com/pierrec/lz4 v2.6.1+incompatible // indirect
	github.com/pierrec/lz4/v4 v4.1.22 // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/prometheus/client_golang v1.22.0 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.65.0 // indirect
	github.com/prometheus/procfs v0.17.0 // indirect
	github.com/prometheus/prometheus v0.304.3-0.20250703114031-419d436a447a // indirect
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475 // indirect
	github.com/rs/cors v1.11.1 // indirect
	github.com/scaleway/scaleway-sdk-go v1.0.0-beta.33 // indirect
	github.com/signalfx/com_signalfx_metrics_protobuf v0.0.3 // indirect
	github.com/signalfx/sapm-proto v0.17.0 // indirect
	github.com/sijms/go-ora/v2 v2.9.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/snowflakedb/gosnowflake v1.15.0 // indirect
	github.com/soheilhy/cmux v0.1.5 // indirect
	github.com/spf13/cobra v1.9.1 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/tinylru v1.2.1 // indirect
	github.com/tidwall/wal v1.1.8 // indirect
	github.com/tinylib/msgp v1.3.0 // indirect
	github.com/tklauser/go-sysconf v0.3.15 // indirect
	github.com/tklauser/numcpus v0.10.0 // indirect
	github.com/vmware/govmomi v0.50.0 // indirect
	github.com/vultr/govultr/v2 v2.17.2 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78 // indirect
	github.com/yuin/gopher-lua v1.1.1 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	go.etcd.io/bbolt v1.4.2 // indirect
	go.mongodb.org/atlas v0.38.0 // indirect
	go.mongodb.org/mongo-driver v1.17.3 // indirect
	go.opentelemetry.io/collector/receiver/nopreceiver v0.130.1
	go.opentelemetry.io/collector/semconv v0.128.1-0.20250610090210-188191247685 // indirect; indir7.0
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.62.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.62.0 // indirect
	go.opentelemetry.io/contrib/propagators/b3 v1.37.0 // indirect
	go.opentelemetry.io/contrib/zpages v0.62.0 // indirect
	go.opentelemetry.io/otel v1.37.0 // indirect
	go.opentelemetry.io/otel/exporters/prometheus v0.59.1 // indirect
	go.opentelemetry.io/otel/metric v1.37.0 // indirect
	go.opentelemetry.io/otel/sdk v1.37.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.37.0
	go.opentelemetry.io/otel/trace v1.37.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/crypto v0.39.0 // indirect
	golang.org/x/exp v0.0.0-20250620022241-b7579e27df2b // indirect
	golang.org/x/mod v0.25.0 // indirect
	golang.org/x/net v0.41.0 // indirect
	golang.org/x/oauth2 v0.30.0 // indirect
	golang.org/x/sync v0.15.0 // indirect
	golang.org/x/term v0.32.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	golang.org/x/time v0.12.0 // indirect
	golang.org/x/tools v0.34.0 // indirect
	golang.org/x/xerrors v0.0.0-20240903120638-7835f813f4da // indirect
	gonum.org/v1/gonum v0.16.0 // indirect
	google.golang.org/api v0.241.0 // indirect
	google.golang.org/genproto v0.0.0-20250505200425-f936aa4a68b2 // indirect
	google.golang.org/grpc v1.73.0 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	k8s.io/api v0.32.3 // indirect
	k8s.io/apimachinery v0.32.3 // indirect
	k8s.io/client-go v0.32.3 // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
	k8s.io/kube-openapi v0.0.0-20241105132330-32ad38e42d3f // indirect
	k8s.io/kubelet v0.32.3 // indirect
	k8s.io/utils v0.0.0-20250502105355-0f33e8f1c979 // indirect
	sigs.k8s.io/json v0.0.0-20241010143419-9aa6b5e7a4b3 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.5.0 // indirect
	sigs.k8s.io/yaml v1.5.0 // indirect
)

// Local component replaces
replace github.com/observiq/bindplane-otel-collector/counter => ./counter

replace github.com/observiq/bindplane-otel-collector/expr => ./expr

replace github.com/observiq/bindplane-otel-collector/packagestate => ./packagestate

// Internal pkgs
replace github.com/observiq/bindplane-otel-collector/internal/aws => ./internal/aws

replace github.com/observiq/bindplane-otel-collector/internal/measurements => ./internal/measurements

replace github.com/observiq/bindplane-otel-collector/internal/rehydration => ./internal/rehydration

replace github.com/observiq/bindplane-otel-collector/internal/report => ./internal/report

replace github.com/observiq/bindplane-otel-collector/internal/storageclient => ./internal/storageclient

replace github.com/observiq/bindplane-otel-collector/internal/testutils => ./internal/testutils

// Extensions
replace github.com/observiq/bindplane-otel-collector/extension/awss3eventextension => ./extension/awss3eventextension

// Exporters
replace github.com/observiq/bindplane-otel-collector/exporter/azureblobexporter => ./exporter/azureblobexporter

replace github.com/observiq/bindplane-otel-collector/exporter/azureloganalyticsexporter => ./exporter/azureloganalyticsexporter

replace github.com/observiq/bindplane-otel-collector/exporter/chronicleexporter => ./exporter/chronicleexporter

replace github.com/observiq/bindplane-otel-collector/exporter/chronicleforwarderexporter => ./exporter/chronicleforwarderexporter

replace github.com/observiq/bindplane-otel-collector/exporter/googlecloudexporter => ./exporter/googlecloudexporter

replace github.com/observiq/bindplane-otel-collector/exporter/googlecloudstorageexporter => ./exporter/googlecloudstorageexporter

replace github.com/observiq/bindplane-otel-collector/exporter/googlemanagedprometheusexporter => ./exporter/googlemanagedprometheusexporter

replace github.com/observiq/bindplane-otel-collector/exporter/qradar => ./exporter/qradar

replace github.com/observiq/bindplane-otel-collector/exporter/snowflakeexporter => ./exporter/snowflakeexporter

replace github.com/observiq/bindplane-otel-collector/exporter/webhookexporter => ./exporter/webhookexporter

// Processors
replace github.com/observiq/bindplane-otel-collector/processor/datapointcountprocessor => ./processor/datapointcountprocessor

replace github.com/observiq/bindplane-otel-collector/processor/logcountprocessor => ./processor/logcountprocessor

replace github.com/observiq/bindplane-otel-collector/processor/lookupprocessor => ./processor/lookupprocessor

replace github.com/observiq/bindplane-otel-collector/processor/maskprocessor => ./processor/maskprocessor

replace github.com/observiq/bindplane-otel-collector/processor/metricextractprocessor => ./processor/metricextractprocessor

replace github.com/observiq/bindplane-otel-collector/processor/metricstatsprocessor => ./processor/metricstatsprocessor

replace github.com/observiq/bindplane-otel-collector/processor/randomfailureprocessor => ./processor/randomfailureprocessor

replace github.com/observiq/bindplane-otel-collector/processor/removeemptyvaluesprocessor => ./processor/removeemptyvaluesprocessor

replace github.com/observiq/bindplane-otel-collector/processor/resourceattributetransposerprocessor => ./processor/resourceattributetransposerprocessor

replace github.com/observiq/bindplane-otel-collector/processor/samplingprocessor => ./processor/samplingprocessor

replace github.com/observiq/bindplane-otel-collector/processor/spancountprocessor => ./processor/spancountprocessor

replace github.com/observiq/bindplane-otel-collector/processor/throughputmeasurementprocessor => ./processor/throughputmeasurementprocessor

replace github.com/observiq/bindplane-otel-collector/processor/topologyprocessor => ./processor/topologyprocessor

replace github.com/observiq/bindplane-otel-collector/processor/unrollprocessor => ./processor/unrollprocessor

// Receivers
replace github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver => ./receiver/awss3eventreceiver

replace github.com/observiq/bindplane-otel-collector/receiver/awss3rehydrationreceiver => ./receiver/awss3rehydrationreceiver

replace github.com/observiq/bindplane-otel-collector/receiver/azureblobrehydrationreceiver => ./receiver/azureblobrehydrationreceiver

replace github.com/observiq/bindplane-otel-collector/receiver/bindplaneauditlogs => ./receiver/bindplaneauditlogs

replace github.com/observiq/bindplane-otel-collector/receiver/googlecloudstoragerehydrationreceiver => ./receiver/googlecloudstoragerehydrationreceiver

replace github.com/observiq/bindplane-otel-collector/receiver/httpreceiver => ./receiver/httpreceiver

replace github.com/observiq/bindplane-otel-collector/receiver/m365receiver => ./receiver/m365receiver

replace github.com/observiq/bindplane-otel-collector/receiver/oktareceiver => ./receiver/oktareceiver

replace github.com/observiq/bindplane-otel-collector/receiver/pluginreceiver => ./receiver/pluginreceiver

replace github.com/observiq/bindplane-otel-collector/receiver/routereceiver => ./receiver/routereceiver

replace github.com/observiq/bindplane-otel-collector/receiver/sapnetweaverreceiver => ./receiver/sapnetweaverreceiver

replace github.com/observiq/bindplane-otel-collector/receiver/splunksearchapireceiver => ./receiver/splunksearchapireceiver

replace github.com/observiq/bindplane-otel-collector/receiver/telemetrygeneratorreceiver => ./receiver/telemetrygeneratorreceiver

replace github.com/observiq/bindplane-otel-collector/receiver/windowseventtracereceiver => ./receiver/windowseventtracereceiver

// Special case replaces
// Does not build with windows and only used in configschema executable
// Relevant issue https://github.com/mattn/go-ieproxy/issues/45
replace github.com/mattn/go-ieproxy => github.com/mattn/go-ieproxy v0.0.1

// Replaces below this are required by datadog exporter in v0.83.0 https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.83.0/exporter/datadogexporter/go.mod#L266-L275

// openshift removed all tags from their repo, use the pseudoversion from the release-3.9 branch HEAD
replace github.com/openshift/api v3.9.0+incompatible => github.com/openshift/api v0.0.0-20180801171038-322a19404e37

// github.com/open-telemetry/opentelemetry-collector-contrib/extension/cgroupruntimeextension.test is forcing this dep to be updated, but it isn't compatible with the linux build process, so replacing to last stable version
replace github.com/cilium/ebpf => github.com/cilium/ebpf v0.11.0

// this is a pair - when the upstream service issues are resolved, we can remove the second replace as well
replace go.opentelemetry.io/collector/service => go.opentelemetry.io/collector/service v0.127.0

replace go.opentelemetry.io/collector/internal/telemetry => go.opentelemetry.io/collector/internal/telemetry v0.127.0
