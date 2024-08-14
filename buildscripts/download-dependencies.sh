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

# HACK(dakota): workaround for getting supervisor binaries until releases are supported
# https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/24293
# download contrib repo and manually build supervisor repos
echo "Cloning supervisor repo"
SUPERVISOR_REPO="https://github.com/open-telemetry/opentelemetry-collector-contrib.git"
PLATFORMS=("linux/amd64" "linux/arm64" "linux/arm" "linux/ppc64" "linux/ppc64le" "darwin/amd64" "darwin/arm64" "windows/amd64")

mkdir "$DOWNLOAD_DIR/supervisor_bin"
$(cd $DOWNLOAD_DIR && git clone --depth 1 "$SUPERVISOR_REPO")
cd "$DOWNLOAD_DIR/opentelemetry-collector-contrib/cmd/opampsupervisor"
go get github.com/open-telemetry/opentelemetry-collector-contrib/cmd/opampsupervisor/supervisor
for PLATFORM in "${PLATFORMS[@]}"; do
    # Split the PLATFORM string into GOOS and GOARCH
    IFS="/" read -r GOOS GOARCH <<< "${PLATFORM}"
    if [ "$GOOS" == "windows" ]; then
        EXT=".exe"
    else
        EXT=""
    fi
    echo "Building supervisor for $GOOS/$GOARCH"
    GOOS="$GOOS" GOARCH="$GOARCH" go build main.go
    cp main $PROJECT_BASE/$DOWNLOAD_DIR/supervisor_bin/opampsupervisor_${GOOS}_${GOARCH}${EXT}
done
$(cd $PROJECT_BASE/$DOWNLOAD_DIR && rm -rf opentelemetry-collector-contrib)
