#!/bin/bash
# Copyright observIQ, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This is the list of stable (v1.0.0+) modules
read -d '' STABLE_MODULES <<EOF
go.opentelemetry.io/collector/pdata
go.opentelemetry.io/collector/client
go.opentelemetry.io/collector/component
go.opentelemetry.io/collector/config/configtls
go.opentelemetry.io/collector/config/configretry
go.opentelemetry.io/collector/confmap
go.opentelemetry.io/collector/config/configopaque
go.opentelemetry.io/collector/confmap/provider/envprovider
go.opentelemetry.io/collector/confmap/provider/fileprovider
go.opentelemetry.io/collector/confmap/provider/httpsprovider
go.opentelemetry.io/collector/confmap/provider/yamlprovider
go.opentelemetry.io/collector/config/confignet
go.opentelemetry.io/collector/consumer
go.opentelemetry.io/collector/extension
go.opentelemetry.io/collector/featuregate
EOF

# Check if we are cleaning up
if [ "$#" -eq 1 ] && [ "$1" == "--cleanup" ]; then
    find manifests -type f -name "manifest.yaml.bak" -exec rm {} \;
    echo "Backup files cleaned up successfully."
    exit 0
fi

# Check if the correct number of arguments are provided
if [ "$#" -ne 4 ]; then
    echo "Usage: $0 <new_version_opentelemetry_contrib> <new_version_opentelemetry_collector> <new_version_bindplane_agent> <new_version_otel_stable>"
    exit 1
fi

# Assign arguments to variables
new_version_opentelemetry_contrib="$1"
new_version_opentelemetry_collector="$2"
new_version_bindplane_agent="$3"
new_version_otel_stable="$4"

# Convert STABLE_MODULES to sed pattern, escaping dots and pipes
stable_modules_pattern=$(echo "$STABLE_MODULES" | sed 's/\./\\./g' | paste -sd "|" - | sed 's/|/\\|/g')

# Find all manifest.yaml files in the manifests directory and its subdirectories
find manifests -type f -name "manifest.yaml" | while read -r file; do
    # Create a backup of the original file
    cp "$file" "${file}.bak"

    # First update all modules with their respective versions
    sed -i '' -E "s|(github.com/open-telemetry/opentelemetry-collector-contrib[^ ]*) v[0-9]+\.[0-9]+\.[0-9]+|\1 $new_version_opentelemetry_contrib|g" "$file"
    sed -i '' -E "s|(go.opentelemetry.io/collector[^ ]*) v[0-9]+\.[0-9]+\.[0-9]+|\1 $new_version_opentelemetry_collector|g" "$file"
    sed -i '' -E "s|(github.com/observiq/bindplane-otel-collector[^ ]*) v[0-9]+\.[0-9]+\.[0-9]+|\1 $new_version_bindplane_agent|g" "$file"

    # Then override stable modules with stable version
    sed -i '' -E "s|(${stable_modules_pattern}[^ ]*) v[0-9]+\.[0-9]+\.[0-9]+|\1 $new_version_otel_stable|g" "$file"

    echo "Versions updated successfully in $file."
done
