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

# Tests for install_unix.sh AIX user creation in install_aix.
# A user install_aix creates is locked, and removed again if the install fails.
# Usage: test_aixuser.sh [path-to-install_unix.sh]
SCRIPT=${1:-$(dirname "$0")/../install_unix.sh}
WORK=$(mktemp -d)
sed '$d' "$SCRIPT" > "$WORK/lib.sh"   # drop trailing main "$@"
pass=0; fail=0

# run_case <name> <user exists: yes|no> <tar exit code> <after install_aix> <expectations...>
# expectations: created-locked | not-created | removed | kept
run_case() {
  name=$1 exists=$2 tar_rc=$3 after=$4; shift 4
  bin="$WORK/bin-$name"; log="$WORK/log-$name"; mkdir -p "$bin"; : > "$log"
  for p in $(IFS=:; for d in $PATH; do printf '%s\n' "$d"/*; done); do
    [ -e "$p" ] && [ ! -e "$bin/${p##*/}" ] || continue   # first match on PATH wins, like the shell
    ln -s "$p" "$bin/${p##*/}"; done
  stubs="uname tar gunzip mkdir mv cp chmod chown lsgroup lssrc lsuser mkgroup mkuser rmuser useradd userdel chuser"
  for t in $stubs; do rm -f "$bin/$t"; done
  printf '#!/bin/sh\necho AIX\n' > "$bin/uname"
  printf '#!/bin/sh\nexit %s\n' "$tar_rc" > "$bin/tar"
  # The install dir is /opt, which a test can't write, and set -e would end the run there
  for t in gunzip mkdir mv cp chmod chown; do printf '#!/bin/sh\nexit 0\n' > "$bin/$t"; done
  printf '#!/bin/sh\nexit 0\n' > "$bin/lsgroup"
  printf '#!/bin/sh\n' > "$bin/lssrc"
  if [ "$exists" = yes ]; then u=0; else u=1; fi
  printf '#!/bin/sh\nexit %s\n' "$u" > "$bin/lsuser"
  for t in mkgroup mkuser rmuser useradd userdel chuser; do printf '#!/bin/sh\necho "%s $*" >> "%s"\n' "$t" "$log" > "$bin/$t"; done
  for t in $stubs; do chmod +x "$bin/$t"; done
  PATH="$bin" "$bin/sh" -c ". '$WORK/lib.sh'
    COLLECTOR_USER=bdot; COLLECTOR_GROUP=bdot; package_out_file_path=$WORK/pkg.tar.gz
    install_aix; $after" > "$WORK/out-$name" 2>&1
  ok=true
  for e in "$@"; do
    case $e in
      created-locked) grep -q '^mkuser .*account_locked=true login=false rlogin=false bdot$' "$log" || ok=false ;;
      not-created)    grep -q '^mkuser' "$log" && ok=false ;;
      removed)        grep -qx 'rmuser -p bdot' "$log" || ok=false ;;
      kept)           grep -q '^rmuser' "$log" && ok=false ;;
    esac
  done
  if $ok; then pass=$((pass+1)); echo "PASS $name"; else fail=$((fail+1)); echo "FAIL $name"; sed 's/^/  log: /' "$log"; tail -3 "$WORK/out-$name"; fi
}

run_case new-user-success       no  0 "created_aix_user=false" created-locked kept
run_case new-user-extract-fails no  1 ":"                      created-locked removed
run_case new-user-later-failure no  0 "false"                  created-locked removed
run_case existing-user-fails    yes 1 ":"                      not-created kept
rm -rf "$WORK"
echo "pass=$pass fail=$fail"; [ $fail -eq 0 ]
