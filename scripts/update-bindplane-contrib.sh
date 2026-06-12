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

TARGET_VERSION=$1
if [ -z "$TARGET_VERSION" ]; then
    echo "Usage: $0 <target-version>"
    echo "Example: $0 v1.1.0"
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
    # Temporarily disable 'set -e' for this command so we can check its exit status
    set +e
    (
        # Exit subshell on any error
        set -e

        echo "Updating deps in $local_mod"
        cd "$local_mod"
        # go list will not work if module is not tidy, so we tidy first
        go mod tidy -compat=1.26.4

        echo "  Tidied $local_mod"

        GO_LIST_OUT=$(go list -m -f '{{if not (or .Indirect .Main)}}{{.Path}}{{end}}' all) || {
            echo "Error: go list failed in $local_mod" >&2
            exit 1
        }
        # Temporarily disable 'set -e' in case there are no bindplane-otel-contrib modules
        set +e
        CONTRIB_MODULES=$(printf '%s\n' "$GO_LIST_OUT" | grep -E -e '^github.com/observiq/bindplane-otel-contrib')
        set -e

        for mod in $CONTRIB_MODULES; do
            echo "  Updating $local_mod: $mod@$TARGET_VERSION"
            go mod edit -require "$mod@$TARGET_VERSION"
        done
    )
    # Get the exit status of the subshell and re-enable 'set -e'
    SUBSHELL_EXIT=$?
    set -e
    # Check if subshell failed and exit if it did
    if [ $SUBSHELL_EXIT -ne 0 ]; then
        echo "Error: Failed to update $local_mod" >&2
        exit 1
    fi
done

# Update the bindplane-otel-contrib module versions pinned in the ocb manifest.
# This is the source of truth for `make agent`, so it must track the same
# versions as the local go.mods updated above. Only bindplane-otel-contrib
# modules are touched; internal bindplane-otel-collector modules (v0.0.0) and
# all other modules are left untouched.
MANIFEST="$REPO_ROOT/manifests/observIQ/manifest.yaml"
echo "Updating manifest $MANIFEST"

MANIFEST_TMP="$(mktemp)"
while IFS= read -r line; do
    case "$line" in
    *"gomod:"*)
        # Split "  - gomod: <module-path> <version>" into its parts.
        prefix="${line%%gomod:*}gomod: "
        rest="${line#*gomod: }"
        mod="${rest%% *}"
        ver="${rest#"$mod" }"

        case "$mod" in
        github.com/observiq/bindplane-otel-contrib/*)
            newver="$TARGET_VERSION"
            ;;
        *)
            # Not a bindplane-otel-contrib module; leave as-is.
            newver="$ver"
            ;;
        esac

        printf '%s%s %s\n' "$prefix" "$mod" "$newver"
        ;;
    *)
        printf '%s\n' "$line"
        ;;
    esac
done < "$MANIFEST" > "$MANIFEST_TMP"
mv "$MANIFEST_TMP" "$MANIFEST"
