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

# Tests for install_unix.sh tail_follow_flag and the logs command it feeds.
# Usage: test_tail_follow.sh [path-to-install_unix.sh]
SCRIPT=${1:-$(dirname "$0")/../install_unix.sh}
WORK=$(mktemp -d)
sed '$d' "$SCRIPT" > "$WORK/lib.sh"   # drop trailing main "$@"
pass=0; fail=0

# run_case <name> <OS_TYPE> <shell snippet> <expected substring>
run_case() {
  name=$1 os=$2 snippet=$3 want=$4
  got=$(sh -c ". '$WORK/lib.sh'; OS_TYPE='$os'; $snippet" 2>/dev/null)
  case "$got" in
    *"$want"*) pass=$((pass+1)); echo "PASS $name" ;;
    *) fail=$((fail+1)); echo "FAIL $name (got '$got', want '$want')" ;;
  esac
}

run_case flag-aix     AIX     tail_follow_flag -f
run_case flag-linux   Linux   tail_follow_flag -F
run_case flag-darwin  Darwin  tail_follow_flag -F
run_case flag-freebsd FreeBSD tail_follow_flag -F
run_case logs-aix     AIX     display_results "tail -f "
run_case logs-linux   Linux   display_results "tail -F "
rm -rf "$WORK"
echo "pass=$pass fail=$fail"; [ $fail -eq 0 ]
