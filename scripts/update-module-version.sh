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

# Directories migrated to the contrib repo — must not be modified here.
# Single source of truth: migrated-modules.txt in the repo root.
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
MIGRATED_PREFIXES=$(tr '\n' ' ' < "$REPO_ROOT/migrated-modules.txt")

is_migrated() {
    mod="$1"
    # Strip leading "./"
    mod="${mod#./}"
    for prefix in $MIGRATED_PREFIXES; do
        case "$mod" in
            ${prefix}*|${prefix%/}) return 0 ;;
        esac
    done
    return 1
}

LOCAL_MODULES=$(find . -type f -name "go.mod" -exec dirname {} \; | sort)
for local_mod in $LOCAL_MODULES; do
    if is_migrated "$local_mod"; then
        echo "Skipping migrated module $local_mod"
        continue
    fi
    # Run in a subshell so that the CD doesn't change this shell's current directory
    (
        echo "Updating version in $local_mod"
        cd "$local_mod" || exit 1
        go mod tidy
        OTEL_MODULES=$(go list -m -f '{{if not (or .Indirect .Main)}}{{.Path}}{{end}}' all |
            grep -E -e '^github.com/observiq/bindplane-otel-collector')

        for mod in $OTEL_MODULES; do
            echo "$local_mod: $mod@$TARGET_VERSION"
            go mod edit -require "$mod@$TARGET_VERSION"
        done
    )
done
