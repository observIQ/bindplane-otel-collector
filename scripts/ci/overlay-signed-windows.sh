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

# Replaces the locally built Windows binaries in <outdir> with the
# Authenticode-signed copies produced by the build-signed-windows workflow.
#
# Goreleaser runs on a Linux runner and has no access to DigiCert KeyLocker, so
# the binaries it packages into the Windows zip are unsigned. The Windows job
# builds and signs them, uploads them as artifacts, and the release job points
# SIGNED_WINDOWS_DIR at the download. Without this overlay the zip that the
# console upgrade path consumes would ship unsigned executables even though the
# MSI is signed (BPOP-5780).
#
# Usage: overlay-signed-windows.sh <outdir>
#
# SIGNED_WINDOWS_DIR unset -> no-op, so local snapshot builds
#   (make release-test) keep working without signing credentials.
# SIGNED_WINDOWS_DIR set   -> every expected binary must be present, or the
#   script fails and aborts the release before anything is published.

set -euo pipefail

outdir="${1:-}"
if [ -z "$outdir" ]; then
  echo "usage: $(basename "$0") <outdir>" >&2
  exit 2
fi

# Branch on whether the variable is present, not on whether it is non-empty. A
# caller that sets it from an expression that evaluates to empty must fail, not
# silently publish unsigned binaries.
if [ -z "${SIGNED_WINDOWS_DIR+set}" ]; then
  echo "overlay-signed-windows: SIGNED_WINDOWS_DIR is not set;" \
    "keeping the locally built (unsigned) Windows binaries in ${outdir}."
  echo "overlay-signed-windows: this is expected for local snapshot builds," \
    "never for a real release."
  exit 0
fi

if [ -z "$SIGNED_WINDOWS_DIR" ]; then
  echo "overlay-signed-windows: SIGNED_WINDOWS_DIR is set but empty." \
    "Refusing to guess; unset it for an unsigned local build, or point it at" \
    "the signed-windows-* artifact download." >&2
  exit 1
fi

if [ ! -d "$outdir" ]; then
  echo "overlay-signed-windows: output directory ${outdir} does not exist" >&2
  exit 1
fi

if [ ! -d "$SIGNED_WINDOWS_DIR" ]; then
  echo "overlay-signed-windows: SIGNED_WINDOWS_DIR=${SIGNED_WINDOWS_DIR}" \
    "is not a directory. Did the signed-windows-* artifact download fail?" >&2
  exit 1
fi

# Must match the windows targets goreleaser packages. See the collector and
# updater builds in .goreleaser.yml: windows is built for amd64 and arm64, and
# arm/ppc64/ppc64le are ignored.
binaries=(
  "collector_windows_amd64.exe"
  "collector_windows_arm64.exe"
  "updater_windows_amd64.exe"
  "updater_windows_arm64.exe"
)

missing=0
for name in "${binaries[@]}"; do
  if [ ! -s "${SIGNED_WINDOWS_DIR}/${name}" ]; then
    echo "overlay-signed-windows: missing or empty ${SIGNED_WINDOWS_DIR}/${name}" >&2
    missing=1
  fi
done

if [ "$missing" -ne 0 ]; then
  echo "overlay-signed-windows: refusing to release unsigned Windows binaries." >&2
  echo "overlay-signed-windows: contents of ${SIGNED_WINDOWS_DIR}:" >&2
  ls -la "$SIGNED_WINDOWS_DIR" >&2 || true
  exit 1
fi

for name in "${binaries[@]}"; do
  cp -f "${SIGNED_WINDOWS_DIR}/${name}" "${outdir}/${name}"
  echo "overlay-signed-windows: installed signed ${name}" \
    "($(wc -c <"${outdir}/${name}" | tr -d ' ') bytes)"
done

echo "overlay-signed-windows: ${#binaries[@]} signed Windows binaries staged in ${outdir}."
