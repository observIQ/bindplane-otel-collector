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

# Tests that install_unix.sh writes manager.yaml as the collector user through
# runuser on Linux, and as root without runuser on AIX, which has no runuser.
# Usage: test_manager_yml.sh [path-to-install_unix.sh]
SCRIPT=${1:-$(dirname "$0")/../install_unix.sh}
WORK=$(mktemp -d)
sed '$d' "$SCRIPT" > "$WORK/lib.sh"   # drop trailing main "$@"
pass=0; fail=0

check() {
  if [ "$2" = true ]; then pass=$((pass+1)); echo "PASS $1"; else fail=$((fail+1)); echo "FAIL $1"; fi
}

# run_create <dir> <os> <endpoint> <labels> <secret_key>
# chown and chgrp are stubbed so the test runs unprivileged. The runuser stub
# records its arguments in <dir>/runuser.args and runs the command after --.
run_create() {
  sh -c ". '$WORK/lib.sh'
    chown() { :; }
    chgrp() { :; }
    runuser() {
      printf '%s\n' \"\$*\" > '$1/runuser.args'
      while [ \"\$1\" != -- ]; do shift; done; shift
      \"\$@\"
    }
    OS_TYPE='$2' COLLECTOR_USER=bdot COLLECTOR_GROUP=bdot
    OPAMP_ENDPOINT='$3' OPAMP_LABELS='$4' OPAMP_SECRET_KEY='$5'
    create_manager_yml '$1/manager.yaml'" > /dev/null 2>&1
}

mode() { ls -l "$1" | cut -c1-10; }

expected_full='endpoint: "wss://example.com/v1/opamp"
labels: "env=test"
secret_key: "s3cret"'

# AIX: written by root, never through runuser
dir="$WORK/aix"; mkdir -p "$dir"
run_create "$dir" AIX wss://example.com/v1/opamp env=test s3cret; rc=$?
ok=true
[ $rc -eq 0 ] || ok=false
[ "$(cat "$dir/manager.yaml")" = "$expected_full" ] || ok=false
[ "$(mode "$dir/manager.yaml")" = "-rw-r-----" ] || ok=false
[ ! -e "$dir/runuser.args" ] || ok=false
check aix_writes_without_runuser $ok

# AIX: the optional fields left out, which must not fail under set -e
dir="$WORK/aix-endpoint-only"; mkdir -p "$dir"
run_create "$dir" AIX wss://example.com/v1/opamp "" ""; rc=$?
ok=true
[ $rc -eq 0 ] || ok=false
[ "$(cat "$dir/manager.yaml")" = 'endpoint: "wss://example.com/v1/opamp"' ] || ok=false
check aix_endpoint_only $ok

# Linux: written as the collector user through runuser
dir="$WORK/linux"; mkdir -p "$dir"
run_create "$dir" Linux wss://example.com/v1/opamp env=test s3cret; rc=$?
ok=true
[ $rc -eq 0 ] || ok=false
[ "$(cat "$dir/manager.yaml")" = "$expected_full" ] || ok=false
[ "$(mode "$dir/manager.yaml")" = "-rw-r-----" ] || ok=false
grep -q -- '-u bdot -g bdot --' "$dir/runuser.args" 2>/dev/null || ok=false
# The secret key goes through stdin, not runuser's arguments
! grep -q s3cret "$dir/runuser.args" 2>/dev/null || ok=false
check linux_writes_through_runuser $ok

# An existing manager.yaml is left alone on both
for os in AIX Linux; do
  dir="$WORK/existing-$os"; mkdir -p "$dir"
  echo 'endpoint: "wss://old.example.com"' > "$dir/manager.yaml"
  run_create "$dir" "$os" wss://example.com/v1/opamp env=test s3cret; rc=$?
  ok=true
  [ $rc -eq 0 ] || ok=false
  [ "$(cat "$dir/manager.yaml")" = 'endpoint: "wss://old.example.com"' ] || ok=false
  [ ! -e "$dir/runuser.args" ] || ok=false
  check "existing_file_kept_$os" $ok
done

[ -n "$KEEP" ] || rm -rf "$WORK"
echo "pass=$pass fail=$fail"; [ $fail -eq 0 ]
