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

# Fails when the AIX supervisor fork branch isn't at the supervisor version the
# other platforms build, by reading the fork branch's contrib-base version.
set -euo pipefail

deps="${1:-buildscripts/download-dependencies.sh}"

want=$(sed -n 's/^ *SUPERVISOR_VERSION="\(v[^"]*\)".*/\1/p' "$deps")
repo=$(sed -n 's/^ *AIX_SUPERVISOR_REPO="\([^"]*\)".*/\1/p' "$deps")
branch=$(sed -n 's/^ *AIX_SUPERVISOR_BRANCH="\([^"]*\)".*/\1/p' "$deps")
if [ -z "$want" ] || [ -z "$repo" ] || [ -z "$branch" ]; then
  echo "could not read the supervisor versions from $deps" >&2
  exit 1
fi

raw="${repo#https://github.com/}"
raw="https://raw.githubusercontent.com/${raw%.git}/$branch/versions.yaml"
if ! versions=$(curl -fsSL "$raw"); then
  echo "could not fetch $raw" >&2
  exit 1
fi
have=$(awk '/^  contrib-base:/{f=1; next} f && /version:/{print $2; exit}' <<< "$versions")

if [ "$have" != "$want" ]; then
  echo "AIX supervisor branch $branch is at ${have:-unknown}, but the other platforms build $want" >&2
  exit 1
fi
echo "AIX supervisor branch $branch is at $want"
