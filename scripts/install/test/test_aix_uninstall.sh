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

# Tests that the AIX uninstall finishes whether the collector is stopped or
# running, removes the subsystem only once it has stopped, and fails without
# removing anything when the collector never stops.
# Usage: test_aix_uninstall.sh [path-to-install_unix.sh]
SCRIPT=${1:-$(dirname "$0")/../install_unix.sh}
WORK=$(mktemp -d)
sed '$d' "$SCRIPT" > "$WORK/lib.sh"   # drop trailing main "$@"
pass=0; fail=0

# run_case <name> <stopsrc exit status> <lssrc polls reporting active before inoperative>
# The SRC commands are shell functions, so nothing reaches the host's services, and
# uninstall_aix is replaced so nothing is deleted.
run_case() {
  name=$1 stop_rc=$2 active_polls=$3
  log="$WORK/log-$name"; : > "$log"; echo "$active_polls" > "$WORK/polls-$name"
  sh -c ". '$WORK/lib.sh'
    OS_TYPE=AIX; SVC_PRE=mkssys; AIX_SRC_GROUP=bpcollector; AIX_SRC_SUBSYSTEM=bindplane-otel-collector; AIX_SRC_WAIT_SECONDS=120
    id() { echo root; }
    sleep() { :; }
    stopsrc() { echo stopsrc >> '$log'; return $stop_rc; }
    lssrc() {
      n=\$(cat '$WORK/polls-$name')
      if [ \"\$n\" -gt 0 ]; then echo \$((n - 1)) > '$WORK/polls-$name'; state=active; else state=inoperative; fi
      echo 'Subsystem Group PID Status'; echo \" bindplane-otel-collector bpcollector \$state\"
    }
    rmssys() { echo \"rmssys polls_left=\$(cat '$WORK/polls-$name')\" >> '$log'; }
    lsitab() { return 0; }
    rmitab() { echo rmitab >> '$log'; }
    uninstall_aix() { echo uninstall_aix >> '$log'; }
    uninstall" > "$WORK/out-$name" 2>&1
  rc=$?
  if [ "$name" = never-stops ]; then
    # Removes nothing and fails once the wait runs out
    if [ $rc -ne 0 ] && grep -q 'did not stop within 120 seconds' "$WORK/out-$name" && ! grep -qE 'rmssys|rmitab|uninstall_aix' "$log"; then
      pass=$((pass+1)); echo "PASS $name"
    else
      fail=$((fail+1)); echo "FAIL $name rc=$rc"; sed 's/^/  log: /' "$log"; tail -3 "$WORK/out-$name"
    fi
  elif grep -qx uninstall_aix "$log" && grep -qx 'rmssys polls_left=0' "$log" && grep -qx rmitab "$log"; then
    pass=$((pass+1)); echo "PASS $name"
  else
    fail=$((fail+1)); echo "FAIL $name"; sed 's/^/  log: /' "$log"; tail -3 "$WORK/out-$name"
  fi
}

run_case already-stopped 1 0
run_case running         0 0
run_case slow-to-stop    0 3
run_case never-stops     0 1000

rm -rf "$WORK"
echo "pass=$pass fail=$fail"; [ $fail -eq 0 ]
