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

# Check if we are cleaning up
if [ "$#" -eq 1 ] && [ "$1" == "--cleanup" ]; then
    find manifests -type f -name "manifest.yaml.bak" -exec rm {} \;
    echo "Backup files cleaned up successfully."
    exit 0
fi

# Check if the correct number of arguments are provided
if [ "$#" -ne 3 ]; then
    echo "Usage: $0 <new_version_opentelemetry_contrib> <new_version_opentelemetry_collector> <new_version_bindplane_agent>"
    exit 1
fi

# Assign arguments to variables
new_version_opentelemetry_contrib="$1"
new_version_opentelemetry_collector="$2"
new_version_bindplane_agent="$3"

# Find all manifest.yaml files in the manifests directory and its subdirectories
find manifests -type f -name "manifest.yaml" | while read -r file; do
    # Create a backup of the original file
    cp "$file" "${file}.bak"

    # Update the dist.otelcol_version value using yq
    # Easy install using `brew install yq`
    yq eval -i ".dist.otelcol_version = \"$new_version_opentelemetry_collector\"" "$file"

    # Use sed to update the version for matching lines
    sed -i '' -E "s|(github.com/open-telemetry/opentelemetry-collector-contrib[^ ]*) v[0-9]+\.[0-9]+\.[0-9]+|\1 $new_version_opentelemetry_contrib|g" "$file"
    sed -i '' -E "s|(go.opentelemetry.io/collector[^ ]*) v[0-9]+\.[0-9]+\.[0-9]+|\1 $new_version_opentelemetry_collector|g" "$file"
    sed -i '' -E "s|(github.com/observiq/bindplane-agent[^ ]*) v[0-9]+\.[0-9]+\.[0-9]+|\1 $new_version_bindplane_agent|g" "$file"

    echo "Versions updated successfully in $file."
done
