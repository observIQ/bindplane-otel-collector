#!/bin/sh
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

# Fails if the AIX manifest contains a component that the main manifest does not.
# Compares module paths (versions ignored) in the component sections; replaces: and dist: are not components.
set -eu

MAIN=${1:-manifests/observIQ/manifest.yaml}
AIX=${2:-manifests/observIQ/manifest-aix.yaml}

components() {
	awk '/^[a-z_]+:/ { s = $1 }
		s ~ /^(extensions|exporters|processors|receivers|connectors|providers):$/ && $1 == "-" && $2 == "gomod:" { print $3 }' "$1" | sort -u
}

tmp=$(mktemp)
trap 'rm -f "$tmp"' EXIT
components "$MAIN" >"$tmp"
extra=$(components "$AIX" | comm -23 - "$tmp")

if [ -n "$extra" ]; then
	echo "$AIX has $(echo "$extra" | wc -l | tr -d ' ') component(s) not in $MAIN:" >&2
	echo "$extra" >&2
	exit 1
fi
echo "$AIX is a subset of $MAIN ($(components "$AIX" | wc -l | tr -d ' ') components)"
