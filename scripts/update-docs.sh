#!/bin/sh
# Copyright  observIQ, Inc.
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


TARGET_VERSION=$1
if [ -z "$TARGET_VERSION" ]; then
    echo "Must specify a target version"
    exit 1
fi

CONTRIB_TARGET_VERSION=$2
if [ -z "$CONTRIB_TARGET_VERSION" ]; then
    echo "Must specify a target contrib version"
    exit 1
fi

BDOT_CONTRIB_VERSION=$3
if [ -z "$BDOT_CONTRIB_VERSION" ]; then
    echo "Must specify a target bindplane contrib version"
    exit 1
fi

read -r -d '' DOC_FILES << EOF
docs/processors.md
docs/extensions.md
docs/connectors.md
docs/exporters.md
docs/receivers.md
EOF

for doc in $DOC_FILES
do
    echo "$doc"
    # Point otel contrib links to new version
    sed -i '' -Ee \
        "s|https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v[^/]*|https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/$CONTRIB_TARGET_VERSION|" \
        "$doc"
    sed -i '' -Ee \
        "s|https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v[^/]*|https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/$CONTRIB_TARGET_VERSION|" \
        "$doc"
    # Point core links to new version
    sed -i '' -Ee \
        "s|https://github.com/open-telemetry/opentelemetry-collector/blob/v[^/]*|https://github.com/open-telemetry/opentelemetry-collector/blob/$TARGET_VERSION|" \
        "$doc"
    # Point bindplane contrib links to new version
    sed -i '' -Ee \
        "s|https://github.com/observiq/bindplane-otel-contrib/blob/v[^/]*|https://github.com/observiq/bindplane-otel-contrib/blob/$BDOT_CONTRIB_VERSION|" \
        "$doc"
    sed -i '' -Ee \
        "s|https://github.com/observiq/bindplane-otel-contrib/tree/v[^/]*|https://github.com/observiq/bindplane-otel-contrib/tree/$BDOT_CONTRIB_VERSION|" \
        "$doc"
    # Normalize any bindplane contrib links pointing to main
    sed -i '' -Ee \
        "s|https://github.com/observiq/bindplane-otel-contrib/blob/main|https://github.com/observiq/bindplane-otel-contrib/blob/$BDOT_CONTRIB_VERSION|" \
        "$doc"
    sed -i '' -Ee \
        "s|https://github.com/observiq/bindplane-otel-contrib/tree/main|https://github.com/observiq/bindplane-otel-contrib/tree/$BDOT_CONTRIB_VERSION|" \
        "$doc"
done
