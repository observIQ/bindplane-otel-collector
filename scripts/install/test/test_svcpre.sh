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

# Tests for install_unix.sh SVC_PRE service-manager detection.
# Usage: test_svcpre.sh [path-to-install_unix.sh]
SCRIPT=${1:-$(dirname "$0")/../install_unix.sh}
WORK=$(mktemp -d)
sed '$d' "$SCRIPT" > "$WORK/lib.sh"   # drop trailing main "$@"
pass=0; fail=0

# run_case <name> <os> <tools on PATH> <expected SVC_PRE>
run_case() {
  name=$1 os=$2 tools=$3 want=$4
  bin="$WORK/bin-$name"; mkdir -p "$bin"
  for p in $(IFS=:; for d in $PATH; do printf '%s\n' "$d"/*; done); do
    [ -e "$p" ] && [ ! -e "$bin/${p##*/}" ] || continue   # first match on PATH wins, like the shell
    ln -s "$p" "$bin/${p##*/}"; done
  rm -f "$bin/uname" "$bin/systemctl" "$bin/service" "$bin/mkssys"
  printf '#!/bin/sh\necho %s\n' "$os" > "$bin/uname"; chmod +x "$bin/uname"
  for t in $tools; do printf '#!/bin/sh\n' > "$bin/$t"; chmod +x "$bin/$t"; done
  got=$(PATH="$bin" "$bin/sh" -c ". '$WORK/lib.sh'; echo \"\$SVC_PRE\"" 2>/dev/null | tail -1)
  if [ "$got" = "$want" ]; then pass=$((pass+1)); echo "PASS $name"; else fail=$((fail+1)); echo "FAIL $name (got '$got', want '$want')"; fi
}

run_case aix-with-service-shim AIX   "service mkssys"           mkssys
run_case aix-with-systemctl    AIX   "systemctl service mkssys" mkssys
run_case aix-plain             AIX   "mkssys"                   mkssys
run_case linux-systemd         Linux "systemctl service"        systemctl
run_case linux-sysv            Linux "service"                  service
rm -rf "$WORK"
echo "pass=$pass fail=$fail"; [ $fail -eq 0 ]
