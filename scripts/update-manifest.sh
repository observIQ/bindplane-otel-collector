#!/bin/sh
# Copyright  Dynatrace LLC
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

# Bumps component versions in the collector manifest. Target versions come from
# the environment; only the groups that are set get touched:
#
#   OTEL_VERSION            core beta-line modules (v0.x), plus the DT edge gomod
#                           lines, whose version mirrors the OTel version
#   CONTRIB_VERSION         contrib beta-line modules (v0.x)
#   PDATA_VERSION           core stable modules (v1.x)
#   CONTRIB_STABLE_VERSION  contrib stable modules (v1.x)
#   DBDOT_CONTRIB_VERSION   dynatrace-bindplane-otel-contrib modules
#   MODULE_VERSION          in-tree dynatrace-bindplane-otel-collector modules
#
# Usage: OTEL_VERSION=v0.163.0 CONTRIB_VERSION=v0.163.0 PDATA_VERSION=v1.69.0 \
#        CONTRIB_STABLE_VERSION=v1.2.0 DBDOT_CONTRIB_VERSION=v1.3.0 ./scripts/update-manifest.sh
#
# By default the full, slim, and AIX manifests are all bumped. Set MANIFEST to a
# single path to bump only that one.
#
# Each group is bumped from the version most of its lines currently have, so a
# line deliberately pinned elsewhere (e.g. jmxreceiver) is left alone. The
# replaces: block is never touched; fork pins are updated by hand. Current
# versions are detected per-manifest, so manifests sitting on different versions
# (e.g. the in-tree modules) are each bumped from their own starting point.

set -e

MANIFESTS=${MANIFEST:-"manifests/observIQ/manifest.yaml \
manifests/observIQ/manifest-aix.yaml"}

# current_version MANIFEST PATH_PATTERN VERSION_PREFIX
# Prints the most common version among gomod lines whose module path contains
# PATH_PATTERN and whose version starts with VERSION_PREFIX.
# ponytail: assumes a group moves together; a stable module sitting on its own
# version is not bumped. The summary at the end shows anything left behind.
current_version() {
    grep -E "^ *- gomod: [^ ]*$2[^ ]* $3" "$1" |
        awk '{print $NF}' | sort | uniq -c | sort -rn | awk 'NR==1 {print $2}'
}

# bump MANIFEST PATH_PATTERN OLD_VERSION NEW_VERSION
bump() {
    if [ -z "$4" ]; then
        return 0
    fi
    if [ -z "$3" ]; then
        echo "  $2: no lines found, skipping"
        return 0
    fi
    echo "  $2: $3 -> $4"
    sed -i '' -E "s|(- gomod: [^ ]*$2[^ ]*) $3\$|\1 $4|" "$1"
}

update_manifest() {
    manifest=$1
    echo "Updating $manifest"
    bump "$manifest" 'go.opentelemetry.io/collector/'         "$(current_version "$manifest" 'go.opentelemetry.io/collector/' 'v0\.')"      "$OTEL_VERSION"
    bump "$manifest" 'opentelemetry-collector-contrib/'       "$(current_version "$manifest" 'opentelemetry-collector-contrib/' 'v0\.')"    "$CONTRIB_VERSION"
    bump "$manifest" 'go.opentelemetry.io/collector/'         "$(current_version "$manifest" 'go.opentelemetry.io/collector/' 'v1\.')"      "$PDATA_VERSION"
    bump "$manifest" 'opentelemetry-collector-contrib/'       "$(current_version "$manifest" 'opentelemetry-collector-contrib/' 'v1\.')"    "$CONTRIB_STABLE_VERSION"
    bump "$manifest" 'dynatrace-bindplane-otel-contrib/'      "$(current_version "$manifest" 'dynatrace-bindplane-otel-contrib/' 'v')"      "$DBDOT_CONTRIB_VERSION"
    bump "$manifest" 'dt-otelcol-edge-processing-components/' "$(current_version "$manifest" 'dt-otelcol-edge-processing-components/' 'v')" "$OTEL_VERSION"
    bump "$manifest" 'dynatrace-bindplane-otel-collector/'    "$(current_version "$manifest" 'dynatrace-bindplane-otel-collector/' 'v')"    "$MODULE_VERSION"

    echo "Versions now in $manifest:"
    grep -oE ' v[0-9][^ ]*$' "$manifest" | sort | uniq -c
}

for manifest in $MANIFESTS; do
    update_manifest "$manifest"
done
