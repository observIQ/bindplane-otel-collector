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

set -e
BASEDIR="$( cd -- "$(dirname "$0")" >/dev/null 2>&1 ; pwd -P )"
PROJECT_BASE="$BASEDIR/../.."

ARCH="${1:-amd64}"

# Map Go arch names to WiX candle -arch values
if [ "$ARCH" = "arm64" ]; then
    WIX_ARCH="arm64"
else
    WIX_ARCH="x64"
fi

# Empty storage directory required by wix.json
mkdir -p storage
touch storage/.include

cp "$PROJECT_BASE/dist/collector_windows_${ARCH}.exe" "observiq-otel-collector.exe"
cp "$PROJECT_BASE/dist/updater_windows_${ARCH}.exe" "updater.exe"

vagrant winrm -c \
    "cd C:/vagrant; go-msi.exe make -m observiq-otel-collector-${ARCH}.msi --version v0.0.1 --arch ${WIX_ARCH}"
