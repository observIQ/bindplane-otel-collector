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

# Tests that install_unix.sh help expands its variables and prints no escapes,
# which AIX's /bin/sh gets wrong for a here-document directly in $(...).
# Usage: test_usage.sh [path-to-install_unix.sh]
SCRIPT=${1:-$(dirname "$0")/../install_unix.sh}
WORK=$(mktemp -d 2>/dev/null) || { WORK=/tmp/usagetest.$$; mkdir "$WORK"; }   # AIX has no mktemp
sed '$d' "$SCRIPT" > "$WORK/lib.sh"   # drop trailing main "$@"
pass=0; fail=0

# run_case <os>
run_case() {
  if [ "$1" = Linux ] && [ "$(uname)" = AIX ]; then echo "SKIP $1 (needs a Linux host: a faked uname can't swap AIX's own tools)"; return; fi
  out=$(sh -c ". '$WORK/lib.sh'; OS_TYPE=$1; usage" 2>&1)
  # The default download URL must be expanded, and a backslash before a quote
  # means the shell printed an escape literally
  if printf '%s' "$out" | grep -q "default to 'https://" && ! printf '%s' "$out" | grep -q "\\\\'"; then
    pass=$((pass+1)); echo "PASS $1"
  else
    fail=$((fail+1)); echo "FAIL $1"
  fi
}

run_case AIX
run_case Linux

rm -rf "$WORK"
echo "pass=$pass fail=$fail"; [ $fail -eq 0 ]
