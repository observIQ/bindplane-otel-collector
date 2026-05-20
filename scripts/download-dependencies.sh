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

# This script downloads common dependencies (plugins + the JMX receiver jar)
# Into the directory specified by the first argument. If no argument is specified,
# the dependencies will be downloaded into the current working directory.
set -e

DOWNLOAD_DIR="${1:-.}"
BASEDIR="$( cd -- "$(dirname "$0")" >/dev/null 2>&1 ; pwd -P )"
PROJECT_BASE="$BASEDIR/.."
JAVA_CONTRIB_VERSION="$(cat "$PROJECT_BASE/JAVA_CONTRIB_VERSION")"

echo "Retrieving java contrib at $JAVA_CONTRIB_VERSION"
curl -fL -o "$DOWNLOAD_DIR/opentelemetry-java-contrib-jmx-metrics.jar" \
    "https://github.com/open-telemetry/opentelemetry-java-contrib/releases/download/$JAVA_CONTRIB_VERSION/opentelemetry-jmx-metrics.jar" > /dev/null 2>&1
