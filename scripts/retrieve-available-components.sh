#!/bin/bash
# Copyright observIQ, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Check if version parameter is provided
if [ $# -ne 1 ]; then
  echo "Usage: $0 <version>"
  exit 1
fi

VERSION=$1
TAG="tags/$VERSION"
OUTPUT_DIR="local/available-components/yaml"
TEMP_DIR=$(mktemp -d 2>/dev/null || mktemp -d -t 'tmpdir') # MacOS compatible temp dir creation

# Create output directory if it doesn't exist
mkdir -p "$OUTPUT_DIR"

git checkout $TAG

touch $OUTPUT_DIR/$VERSION.yaml

echo 'kind: AvailableComponents' >"$OUTPUT_DIR/$VERSION.yaml"
echo 'metadata:' >>"$OUTPUT_DIR/$VERSION.yaml"
echo "  name: available-components-$VERSION" >>"$OUTPUT_DIR/$VERSION.yaml"
echo 'spec:' >>"$OUTPUT_DIR/$VERSION.yaml"
echo "  hash: available-components-$VERSION" >>"$OUTPUT_DIR/$VERSION.yaml"
echo '  components:' >>"$OUTPUT_DIR/$VERSION.yaml"

cat go.mod | grep -E '	(go.opentelemetry.io/collector|(github.com/(open-telemetry/opentelemetry-collector-contrib|observiq/bindplane-otel-collector|observiq/observiq-otel-collector|observiq/bindplane-agent)))/receiver/' | grep -v "// indirect" | grep -v "go.opentelemetry.io/collector/receiver/receivertest" | sed -E 's/^[[:space:]]*//;s/[[:space:]]*$//' | awk -F'/' 'BEGIN {
    printf "    receivers:\n      sub_component_details:\n"
    myMap["activedirectoryds"] = "active_directory_ds"
    myMap["dockerstats"] = "docker_stats"
    myMap["k8scluster"] = "k8s_cluster"
    myMap["k8sevents"] = "k8s_events"
    myMap["podman"] = "podman_stats"
    myMap["simpleprometheus"] = "prometheus_simple"
    myMap["receivercreator"] = "receiver_creator"
    myMap["splunkhec"] = "splunk_hec"
    myMap["huaweicloudcesreceiver"] = "huaweicloudcesreceiver"
    myMap["awscontainerinsightreceiver"] = "awscontainerinsightreceiver"
    myMap["telemetrygenerator"] = "telemetrygeneratorreceiver"
    myMap["dotnetdiagnostics"] = "dotnet_diagnostics"
    myMap["awss3event"] = "s3event"
    myMap["mongodbatlas"] = "mongodb_atlas"
    myMap["azureeventhub"] = "azure_event_hub"
} {
  split($NF, parts, " ")
  name=parts[1]
  sub("receiver$", "", name)
  if (name in myMap) {
    name = myMap[name]
  }
  namespace=$0
  
  if (!first) {
    first=1
  } else {
    printf "\n"
  }
  
  printf "        %s:\n          metadata:\n          - key: code.namespace\n            value: %s", name, namespace

} END { 
  printf "\n"
}' >>"$OUTPUT_DIR/$VERSION.yaml"

cat go.mod | grep -E '	(go.opentelemetry.io/collector|(github.com/(open-telemetry/opentelemetry-collector-contrib|observiq/bindplane-otel-collector|observiq/observiq-otel-collector|observiq/bindplane-agent)))/connector/' | grep -v "// indirect" | grep -v "go.opentelemetry.io/collector/connector/connectortest" | sed -E 's/^[[:space:]]*//;s/[[:space:]]*$//' | awk -F'/' 'BEGIN {
  printf "    connectors:\n      sub_component_details:\n"
} {
  split($NF, parts, " ")
  name=parts[1]
  sub("connector$", "", name)
  namespace=$0
  
  if (!first) {
    first=1
  } else {
    printf "\n"
  }
  
  printf "        %s:\n          metadata:\n          - key: code.namespace\n            value: %s", name, namespace

  } END { printf "\n"
}' >>"$OUTPUT_DIR/$VERSION.yaml"

cat go.mod | grep -E '	(go.opentelemetry.io/collector|(github.com/(open-telemetry/opentelemetry-collector-contrib|observiq/bindplane-otel-collector|observiq/observiq-otel-collector|observiq/bindplane-agent)))/exporter/' | grep -v "// indirect" | grep -v "go.opentelemetry.io/collector/exporter/exportertest" | sed -E 's/^[[:space:]]*//;s/[[:space:]]*$//' | awk -F'/' 'BEGIN {
  printf "    exporters:\n      sub_component_details:\n"
  myMap["splunkhec"] = "splunk_hec"
  myMap["tencentcloudlogservice"] = "tencentcloud_logservice"
  myMap["alibabacloudlogservice"] = "alibabacloud_logservice"
  myMap["jaegerthrifthttp"] = "jaeger_thrift"
  myMap["otlp"] = "otlp_grpc"
  myMap["otlphttp"] = "otlp_http"
} {
  split($NF, parts, " ")
  name=parts[1]
  sub("exporter$", "", name)
  if (name in myMap) {
    name = myMap[name]
  }
  namespace=$0

  if (!first) {
    first=1
  } else {
    printf "\n"
  }
  
  printf "        %s:\n          metadata:\n          - key: code.namespace\n            value: %s", name, namespace

  } END { printf "\n"
}' >>"$OUTPUT_DIR/$VERSION.yaml"

# add entry for github.com/honeycombio/enhance-indexing-s3-exporter/enhanceindexings3exporter with version
enhance_indexing_s3_exporter_version=$(grep -m1 'github.com/honeycombio/enhance-indexing-s3-exporter/enhanceindexings3exporter' go.mod | awk '{print $2}')
if [ -n "$enhance_indexing_s3_exporter_version" ]; then
  cat >>"$OUTPUT_DIR/$VERSION.yaml" <<EOF
        enhance_indexing_s3_exporter:
          metadata:
          - key: code.namespace
            value: github.com/honeycombio/enhance-indexing-s3-exporter/enhanceindexings3exporter ${enhance_indexing_s3_exporter_version}
EOF
fi

cat go.mod | grep -E '	(go.opentelemetry.io/collector|(github.com/(open-telemetry/opentelemetry-collector-contrib|observiq/bindplane-otel-collector|observiq/observiq-otel-collector|observiq/bindplane-agent)))/extension/' | grep -v "// indirect" | grep -v "go.opentelemetry.io/collector/extension/extensiontest" | sed -E 's/^[[:space:]]*//;s/[[:space:]]*$//' | awk -F'/' 'BEGIN {
  printf "    extensions:\n      sub_component_details:\n"
  myMap["filestorage"] = "file_storage"
  myMap["redisstorage"] = "redis_storage"
  myMap["dbstorage"] = "db_storage"
  myMap["httpforwarder"] = "http_forwarder"
  myMap["asapauth"] = "asapclient"
  myMap["oidcauth"] = "oidc"
  myMap["oauth2clientauth"] = "oauth2client"
  myMap["memorylimiter"] = "memory_limiter"
  myMap["healthcheck"] = "health_check"
  myMap["headerssetter"] = "headers_setter"
  myMap["ballast"] = "memory_ballast"
  myMap["awss3event"] = "s3event"
  myMap["jsonlogencoding"] = "json_log_encoding"
  myMap["avrologencoding"] = "avro_log_encoding"
  myMap["googlecloudlogentryencoding"] = "googlecloudlogentry_encoding"
  myMap["textencoding"] = "text_encoding"
} {
  split($NF, parts, " ")
  name=parts[1]
  sub("extension$", "", name)
  if (name in myMap) {
    name = myMap[name]
  }

  namespace=$0
  
  if (!first) {
    first=1
  } else {
    printf "\n"
  }
  
  printf "        %s:\n          metadata:\n          - key: code.namespace\n            value: %s", name, namespace

  } END { printf "\n"
}' >>"$OUTPUT_DIR/$VERSION.yaml"

cat go.mod | grep -E '	(go.opentelemetry.io/collector|(github.com/(open-telemetry/opentelemetry-collector-contrib|observiq/bindplane-otel-collector|observiq/observiq-otel-collector|observiq/bindplane-agent)))/processor/' | grep -v "// indirect" | grep -v "go.opentelemetry.io/collector/processor/processortest" | sed -E 's/^[[:space:]]*//;s/[[:space:]]*$//' | awk -F'/' 'BEGIN {
  printf "    processors:\n      sub_component_details:\n"
  myMap["tailsampling"] = "tail_sampling"
  myMap["probabilisticsampler"] = "probabilistic_sampler"
  myMap["memorylimiter"] = "memory_limiter"
  myMap["logdeduplication"] = "logdedup"
  myMap["signaltometrics"] = "signal_to_metrics"
  myMap["k8sattributes"] = "k8s_attributes"
} {
  split($NF, parts, " ")
  name=parts[1]
  sub("processor$", "", name)
  if (name in myMap) {
    name = myMap[name]
  }
  namespace=$0
  
  if (!first) {
    first=1
  } else {
    printf "\n"
  }
  
  printf "        %s:\n          metadata:\n          - key: code.namespace\n            value: %s", name, namespace
  } END { printf "\n"
}' >>"$OUTPUT_DIR/$VERSION.yaml"
