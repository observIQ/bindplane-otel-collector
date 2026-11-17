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

# Tests for create-support-bundle.sh install detection on AIX and Linux.
# Usage: test_detect.sh [path-to-create-support-bundle.sh]
SCRIPT=${1:-$(dirname "$0")/../create-support-bundle.sh}
. "$(dirname "$0")/stubs.sh"
WORK=$(mktemp -d 2>/dev/null) || { WORK=/tmp/sbtest.$$; mkdir "$WORK"; }   # AIX has no mktemp
sed '$d' "$SCRIPT" > "$WORK/lib.sh"   # drop trailing main "$@"
pass=0; fail=0
n=0

# run_case <name> <os> <"svc=path pid" ...> -- <stdin> <expected "version service dir pid">
run_case() {
  name=$1 os=$2; shift 2
  n=$((n+1)); bin="$WORK/bin$n"; state="$WORK/state$n"
  make_stubs "$bin" "$state" "$os"
  while [ "$1" != -- ]; do echo "${1#*=}" > "$state/${1%%=*}"; shift; done
  input=$2 want=$3
  got=$(printf '%s\n' "$input" | PATH="$bin:$PATH" sh -c ". '$WORK/lib.sh'; detect_install; echo \"\$collector_version \$collector_service \$collector_dir \$(service_pid \"\$collector_service\")\"" 2>/dev/null)
  if [ "$got" = "$want" ]; then pass=$((pass+1)); echo "PASS $name"; else fail=$((fail+1)); echo "FAIL $name (got '$got', want '$want')"; fi
}

run_case aix-v1        AIX   "observiq-otel-collector=/opt/oiq/observiq-otel-collector 123" -- "" \
  "v1 observiq-otel-collector /opt/oiq 123"
run_case aix-v1-stopped AIX  "observiq-otel-collector=/opt/oiq/observiq-otel-collector" -- "" \
  "v1 observiq-otel-collector /opt/oiq "
run_case aix-v2        AIX   "bindplane-otel-collector=/opt/bp/opampsupervisor 456" -- "" \
  "v2 bindplane-otel-collector /opt/bp 456"
run_case aix-both-pick-2 AIX "observiq-otel-collector=/opt/oiq/observiq-otel-collector 1" \
  "bindplane-otel-collector=/opt/bp/opampsupervisor 2" -- "2" \
  "v2 bindplane-otel-collector /opt/bp 2"
run_case linux-v1      Linux "observiq-otel-collector=/opt/oiq/observiq-otel-collector 789" -- "" \
  "v1 observiq-otel-collector.service /opt/oiq 789"
run_case linux-v2      Linux "bindplane-otel-collector=/opt/bp/opampsupervisor 321" -- "" \
  "v2 bindplane-otel-collector.service /opt/bp 321"

rm -rf "$WORK"
echo "pass=$pass fail=$fail"; [ $fail -eq 0 ]
