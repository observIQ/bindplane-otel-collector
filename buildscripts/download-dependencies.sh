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
SUPERVISOR_VERSION="${2:-0.145.0}"
BASEDIR="$( cd -- "$(dirname "$0")" >/dev/null 2>&1 ; pwd -P )"
PROJECT_BASE="$BASEDIR/.."
JAVA_CONTRIB_VERSION="$(cat "$PROJECT_BASE/JAVA_CONTRIB_VERSION")"

echo "Retrieving java contrib at $JAVA_CONTRIB_VERSION"
curl -fL -o "$DOWNLOAD_DIR/opentelemetry-java-contrib-jmx-metrics.jar" \
    "https://github.com/open-telemetry/opentelemetry-java-contrib/releases/download/$JAVA_CONTRIB_VERSION/opentelemetry-jmx-metrics.jar" > /dev/null 2>&1

# HACK(dakota): workaround for getting supervisor binaries until releases are supported
# https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/24293
# download contrib repo and manually build supervisor repos

mkdir "$DOWNLOAD_DIR/supervisor_bin"

# Build supervisor for all non-AIX platforms from upstream
echo "Cloning upstream supervisor repo"
SUPERVISOR_REPO="https://github.com/open-telemetry/opentelemetry-collector-contrib.git"
PLATFORMS=("linux/amd64" "linux/arm64" "linux/ppc64" "linux/ppc64le" "darwin/amd64" "darwin/arm64" "windows/amd64")

mkdir "$DOWNLOAD_DIR/opentelemetry-collector-contrib"
cd $DOWNLOAD_DIR/opentelemetry-collector-contrib
git init
git remote add -f origin "$SUPERVISOR_REPO"
git config core.sparseCheckout true
echo "cmd/opampsupervisor" >> .git/info/sparse-checkout
git pull origin main --depth 1

cd "cmd/opampsupervisor"
for PLATFORM in "${PLATFORMS[@]}"; do
    IFS="/" read -r GOOS GOARCH <<< "${PLATFORM}"
    if [ "$GOOS" == "windows" ]; then
        EXT=".exe"
    else
        EXT=""
    fi
    echo "Building supervisor for $GOOS/$GOARCH"
    GOOS="$GOOS" GOARCH="$GOARCH" CGO_ENABLED=0 go build -o $PROJECT_BASE/$DOWNLOAD_DIR/supervisor_bin/opampsupervisor_${GOOS}_${GOARCH}${EXT} .
done
$(cd $PROJECT_BASE/$DOWNLOAD_DIR && rm -rf opentelemetry-collector-contrib)

# Build supervisor for AIX from custom fork with AIX compatibility patches
echo "Cloning AIX supervisor repo (custom fork)"
AIX_SUPERVISOR_REPO="https://github.com/observIQ/opentelemetry-collector-contrib.git"
AIX_SUPERVISOR_BRANCH="patch/aix_config-v0.145.x"
AIX_PLATFORMS=("aix/ppc64")

mkdir "$DOWNLOAD_DIR/opentelemetry-collector-contrib-aix"
cd $DOWNLOAD_DIR/opentelemetry-collector-contrib-aix
git init
git remote add -f origin "$AIX_SUPERVISOR_REPO"
git config core.sparseCheckout true
echo "cmd/opampsupervisor" >> .git/info/sparse-checkout
git pull origin "$AIX_SUPERVISOR_BRANCH" --depth 1

cd "cmd/opampsupervisor"
for PLATFORM in "${AIX_PLATFORMS[@]}"; do
    IFS="/" read -r GOOS GOARCH <<< "${PLATFORM}"
    echo "Building AIX supervisor for $GOOS/$GOARCH"
    GOOS="$GOOS" GOARCH="$GOARCH" CGO_ENABLED=0 go build -o $PROJECT_BASE/$DOWNLOAD_DIR/supervisor_bin/opampsupervisor_${GOOS}_${GOARCH} .
done
$(cd $PROJECT_BASE/$DOWNLOAD_DIR && rm -rf opentelemetry-collector-contrib-aix)
