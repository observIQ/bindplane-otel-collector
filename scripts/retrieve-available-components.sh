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
BRANCH="release/$VERSION"
OUTPUT_DIR="local/available-components/yaml"
TEMP_DIR=$(mktemp -d 2>/dev/null || mktemp -d -t 'tmpdir') # MacOS compatible temp dir creation

# Create output directory if it doesn't exist
mkdir -p "$OUTPUT_DIR"

git checkout $BRANCH

touch $OUTPUT_DIR/$VERSION.yaml

echo 'kind: AvailableComponents' > "$OUTPUT_DIR/$VERSION.yaml"
echo 'metadata:' >> "$OUTPUT_DIR/$VERSION.yaml"
echo "  name: available-components-$VERSION" >> "$OUTPUT_DIR/$VERSION.yaml"
echo 'spec:' >> "$OUTPUT_DIR/$VERSION.yaml"
echo "  hash: available-components-$VERSION" >> "$OUTPUT_DIR/$VERSION.yaml"
echo '  components:' >> "$OUTPUT_DIR/$VERSION.yaml"

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
}' >> "$OUTPUT_DIR/$VERSION.yaml"

cat go.mod | grep -E '	(go.opentelemetry.io/collector|(github.com/(open-telemetry/opentelemetry-collector-contrib|observiq/bindplane-otel-collector|observiq/observiq-otel-collector|observiq/bindplane-agent)))/connector/' | grep -v "// indirect"| grep -v "go.opentelemetry.io/collector/connector/connectortest" | sed -E 's/^[[:space:]]*//;s/[[:space:]]*$//' | awk -F'/' 'BEGIN {
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
}' >> "$OUTPUT_DIR/$VERSION.yaml"

cat go.mod | grep -E '	(go.opentelemetry.io/collector|(github.com/(open-telemetry/opentelemetry-collector-contrib|observiq/bindplane-otel-collector|observiq/observiq-otel-collector|observiq/bindplane-agent)))/exporter/' | grep -v "// indirect" | grep -v "go.opentelemetry.io/collector/exporter/exportertest" | sed -E 's/^[[:space:]]*//;s/[[:space:]]*$//' | awk -F'/' 'BEGIN {
  printf "    exporters:\n      sub_component_details:\n"
  myMap["splunkhec"] = "splunk_hec"
  myMap["tencentcloudlogservice"] = "tencentcloud_logservice"
  myMap["alibabacloudlogservice"] = "alibabacloud_logservice"
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
}' >> "$OUTPUT_DIR/$VERSION.yaml"

cat go.mod | grep -E '	(go.opentelemetry.io/collector|(github.com/(open-telemetry/opentelemetry-collector-contrib|observiq/bindplane-otel-collector|observiq/observiq-otel-collector|observiq/bindplane-agent)))/extension/' | grep -v "// indirect"| grep -v "go.opentelemetry.io/collector/extension/extensiontest" | sed -E 's/^[[:space:]]*//;s/[[:space:]]*$//' | awk -F'/' 'BEGIN {
  printf "    extensions:\n      sub_component_details:\n"
  myMap["filestorage"] = "file_storage"
  myMap["redisstorage"] = "redis_storage"
  myMap["dbstorage"] = "db_storage"
  myMap["httpforwarder"] = "http_forwarder"
  myMap["asapauth"] = "asapclient"
  myMap["oidcauth"] = "oidc"
  myMap["oauth2clientauth"] = "oauth2client"
  myMap["memorylimiter"] = "memory_limiter"
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
}' >> "$OUTPUT_DIR/$VERSION.yaml"

cat go.mod | grep -E '	(go.opentelemetry.io/collector|(github.com/(open-telemetry/opentelemetry-collector-contrib|observiq/bindplane-otel-collector|observiq/observiq-otel-collector|observiq/bindplane-agent)))/processor/' | grep -v "// indirect"| grep -v "go.opentelemetry.io/collector/processor/processortest" | sed -E 's/^[[:space:]]*//;s/[[:space:]]*$//' | awk -F'/' 'BEGIN {
  printf "    processors:\n      sub_component_details:\n"
  myMap["tailsampling"] = "tail_sampling"
  myMap["probabilisticsampler"] = "probabilistic_sampler"
  myMap["memorylimiter"] = "memory_limiter"
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
}' >> "$OUTPUT_DIR/$VERSION.yaml"

git co main
git br -d $BRANCH
