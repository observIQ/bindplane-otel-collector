#!/usr/bin/env bash
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

# Bundles the release artifacts into tarballs that get attached to the release.
#
# Produces, in the current directory:
#   observiq-otel-collector-<version>-artifacts.tar.gz          every artifact
#   observiq-otel-collector-<version>-artifacts-linux.tar.gz    Linux only
#   observiq-otel-collector-<version>-artifacts-darwin.tar.gz   macOS only
#   observiq-otel-collector-<version>-artifacts-windows.tar.gz  Windows only
#
# The all-platform bundle is kept until consumers have moved to the
# per-platform bundles. Bindplane merges per-platform bundles uploaded for the
# same version, so each bundle is self-contained: it carries every artifact for
# its platforms, the full SHA256SUMS file, and gpg-keys.tar.gz. Verify a
# per-platform bundle's contents with `sha256sum -c --ignore-missing`.
#
# Usage: create-artifact-bundles.sh <version>
#
# Run from the repository root after goreleaser has populated ./dist and the
# MSIs have been downloaded to the root.

set -euo pipefail

if [ "$#" -ne 1 ]; then
  echo "usage: $0 <version>" >&2
  exit 1
fi

version="$1"
prefix="observiq-otel-collector-${version}"

# Fail on a glob that matches nothing rather than copying the literal pattern.
shopt -s failglob

# bundle <label> <file>... tars the given files into <prefix>-<label>.tar.gz.
bundle() {
  local label="$1"
  shift
  local dir="bundle-${label}"
  rm -rf "$dir"
  mkdir "$dir"
  cp "$@" "$dir"
  tar -czvf "${prefix}-${label}.tar.gz" -C "$dir" .
}

# Files every bundle needs, regardless of platform.
common=(
  ./release_deps/gpg-keys.tar.gz
  ./dist/*SHA256SUMS
)

bundle artifacts \
  "${common[@]}" \
  ./scripts/install/*.sh \
  ./scripts/install/*.ps1 \
  ./observiq-otel-collector.msi \
  ./observiq-otel-collector-arm64.msi \
  ./dist/*tar.gz \
  ./dist/*.zip \
  ./dist/*.rpm \
  ./dist/*.deb

bundle artifacts-linux \
  "${common[@]}" \
  ./scripts/install/install_unix.sh \
  ./dist/*-linux-*.tar.gz \
  ./dist/*.rpm \
  ./dist/*.deb

bundle artifacts-darwin \
  "${common[@]}" \
  ./scripts/install/install_macos.sh \
  ./dist/*-darwin-*.tar.gz

bundle artifacts-windows \
  "${common[@]}" \
  ./scripts/install/install_windows.ps1 \
  ./observiq-otel-collector.msi \
  ./observiq-otel-collector-arm64.msi \
  ./dist/*-windows-*.zip
