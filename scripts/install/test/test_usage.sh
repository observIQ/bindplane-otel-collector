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

# Tests that install_unix.sh help shows the package names for the OS it runs on.
# Usage: test_usage.sh [path-to-install_unix.sh]
SCRIPT=${1:-$(dirname "$0")/../install_unix.sh}
WORK=$(mktemp -d)
sed '$d' "$SCRIPT" > "$WORK/lib.sh"   # drop trailing main "$@"
pass=0; fail=0

# run_case <os> <expected> <unexpected>
run_case() {
  if [ "$1" = Linux ] && [ "$(uname)" = AIX ]; then echo "SKIP $1 (needs a Linux host: a faked uname can't swap AIX's own tools)"; return; fi
  out=$(sh -c ". '$WORK/lib.sh'; OS_TYPE=$1; usage" 2>&1)
  if printf '%s' "$out" | grep -q -- "$2" && ! printf '%s' "$out" | grep -q -- "$3"; then
    pass=$((pass+1)); echo "PASS $1"
  else
    fail=$((fail+1)); echo "FAIL $1"
  fi
}

run_case AIX   "aix-ppc64.tar.gz" "_linux_"
run_case Linux "_linux_"          "aix-ppc64"

rm -rf "$WORK"
echo "pass=$pass fail=$fail"; [ $fail -eq 0 ]
