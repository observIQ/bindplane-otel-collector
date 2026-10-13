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

# Tests for install_unix.sh dependencies_check.
# Usage: test_prereqs.sh [path-to-install_unix.sh]
# Sources the script with main stripped, on a PATH of symlinks that omits chosen tools.
SCRIPT=${1:-$(dirname "$0")/../install_unix.sh}
WORK=$(mktemp -d)
sed '$d' "$SCRIPT" > "$WORK/lib.sh"   # drop trailing main "$@"
pass=0; fail=0

# run_case <name> <os> <omit-list> <expect: ok|fail> <must-mention...>
run_case() {
  name=$1 os=$2 omit=$3 expect=$4; shift 4
  if [ "$os" = Linux ] && [ "$(uname)" = AIX ]; then echo "SKIP $name (needs a Linux host: a faked uname can't swap AIX's own tools)"; return; fi
  bin="$WORK/bin-$name"; mkdir -p "$bin"
  # Mirror PATH, minus the omitted tools.
  for p in $(IFS=:; for d in $PATH; do printf '%s\n' "$d"/*; done); do
    [ -e "$p" ] && [ ! -e "$bin/${p##*/}" ] || continue   # first match on PATH wins, like the shell
    t=${p##*/}
    case " $omit " in *" $t "*) continue ;; esac
    ln -s "$p" "$bin/$t"
  done
  rm -f "$bin/uname"
  printf '#!/bin/sh\necho %s\n' "$os" > "$bin/uname"; chmod +x "$bin/uname"
  # No TERM, as under CI, cron, or a pipe, where tput fails
  out=$(unset TERM; PATH="$bin" "$bin/sh" -c ". '$WORK/lib.sh'; dependencies_check" 2>&1); rc=$?
  ok=true
  if [ "$expect" = ok ] && [ $rc -ne 0 ]; then ok=false; fi
  if [ "$expect" = fail ]; then
    [ $rc -ne 0 ] || ok=false
    req=$(printf '%s\n' "$out" | grep 'required by this script')
    for m in "$@"; do printf '%s' "$req" | grep -q "$m" || ok=false; done
  fi
  if $ok; then pass=$((pass+1)); echo "PASS $name"; else fail=$((fail+1)); echo "FAIL $name (rc=$rc)"; printf "%s\n" "$out" | tail -5; fi
}

run_case linux-all-present Linux ""          ok
run_case linux-no-awk      Linux "awk"       fail awk
run_case aix-no-gunzip     AIX   "gunzip"    fail gunzip
run_case aix-no-both       AIX   "gunzip awk" fail gunzip awk
run_case aix-no-iostat      AIX   "iostat"    ok
[ -n "$KEEP" ] || rm -rf "$WORK"
echo "pass=$pass fail=$fail"; [ $fail -eq 0 ]
