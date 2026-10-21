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

# Tests that install_unix.sh prerequisites never gate on AIX I/O accounting.
# The collector reads AIX process and disk I/O through libperfstat and libc, so
# neither the sys0 iostat attribute nor Workload Manager (WLM) is needed.
# Usage: test_aix_io.sh [path-to-install_unix.sh]
SCRIPT=${1:-$(dirname "$0")/../install_unix.sh}
WORK=$(mktemp -d)
sed '$d' "$SCRIPT" > "$WORK/lib.sh"   # drop trailing main "$@"
pass=0; fail=0

# run_case <name> <setup shell>
run_case() {
  name=$1 setup=$2
  bin="$WORK/bin-$name"; log="$WORK/log-$name"; mkdir -p "$bin"; : > "$log"
  for p in $(IFS=:; for d in $PATH; do printf '%s\n' "$d"/*; done); do
    [ -e "$p" ] && [ ! -e "$bin/${p##*/}" ] || continue   # first match on PATH wins, like the shell
    ln -s "$p" "$bin/${p##*/}"; done
  rm -f "$bin/uname" "$bin/lsattr" "$bin/wlmcntrl" "$bin/mkitab"
  printf '#!/bin/sh\necho AIX\n' > "$bin/uname"
  # iostat off and WLM stopped: the worst case for the old checks
  printf '#!/bin/sh\necho "lsattr $*" >> "%s"\necho "iostat false Continuously maintain DISK I/O history True"\n' "$log" > "$bin/lsattr"
  printf '#!/bin/sh\necho "wlmcntrl $*" >> "%s"\necho "WLM is stopped"; exit 1\n' "$log" > "$bin/wlmcntrl"
  printf '#!/bin/sh\necho "mkitab $*" >> "%s"\n' "$log" > "$bin/mkitab"
  chmod +x "$bin/uname" "$bin/lsattr" "$bin/wlmcntrl" "$bin/mkitab"
  # Stub the checks that need a real AIX host, so only the I/O gate is under test.
  stubs='root_check(){ :; }; os_check(){ :; }; set_os_arch(){ :; }; os_arch_check(){ :; }; package_type_check(){ :; }; dependencies_check(){ :; }; user_check(){ :; }'
  out=$(printf 'y\n' | PATH="$bin" "$bin/sh" -c ". '$WORK/lib.sh'; $stubs; $setup; check_prereqs; echo REACHED_END" 2>&1); rc=$?
  ok=true
  [ $rc -eq 0 ] || ok=false
  printf '%s' "$out" | grep -q REACHED_END || ok=false
  printf '%s' "$out" | grep -qi 'WARNING' && ok=false
  [ -s "$log" ] && ok=false
  if $ok; then pass=$((pass+1)); echo "PASS $name"; else fail=$((fail+1)); echo "FAIL $name (rc=$rc)"; printf '%s\n' "$out" | tail -4; sed 's/^/  log: /' "$log"; fi
}

run_case interactive ":"
run_case quiet       "non_interactive=true"
rm -rf "$WORK"
echo "pass=$pass fail=$fail"; [ $fail -eq 0 ]
