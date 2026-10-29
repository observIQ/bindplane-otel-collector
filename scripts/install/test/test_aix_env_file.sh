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

# Tests that install_unix.sh backs up an existing AIX env file before replacing it.
# Usage: test_aix_env_file.sh [path-to-install_unix.sh]
SCRIPT=${1:-$(dirname "$0")/../install_unix.sh}
WORK=$(mktemp -d)
sed '$d' "$SCRIPT" > "$WORK/lib.sh"   # drop trailing main "$@"
pass=0; fail=0

check() {
  if [ "$2" = true ]; then pass=$((pass+1)); echo "PASS $1"; else fail=$((fail+1)); echo "FAIL $1"; fi
}

# fresh install: no existing file, so no backup
dir="$WORK/fresh"; mkdir -p "$dir"
echo "A=new" > "$dir/pkg.env"
sh -c ". '$WORK/lib.sh'; AIX_ENV_FILE='$dir/collector.env'; install_aix_env_file '$dir/pkg.env'" > /dev/null 2>&1
ok=true
[ "$(cat "$dir/collector.env")" = "A=new" ] || ok=false
for f in "$dir"/*.bak; do [ -e "$f" ] && ok=false; done
check fresh_install_has_no_backup $ok

# upgrade: the existing file is kept as a timestamped backup
dir="$WORK/upgrade"; mkdir -p "$dir"
echo "A=new" > "$dir/pkg.env"
echo "A=edited" > "$dir/collector.env"
sh -c ". '$WORK/lib.sh'; AIX_ENV_FILE='$dir/collector.env'; install_aix_env_file '$dir/pkg.env'" > /dev/null 2>&1
ok=true
[ "$(cat "$dir/collector.env")" = "A=new" ] || ok=false
backups=0
for f in "$dir"/collector.env.[0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9].bak; do
  [ -e "$f" ] || continue
  backups=$((backups+1))
  [ "$(cat "$f")" = "A=edited" ] || ok=false
done
[ "$backups" = 1 ] || ok=false
check upgrade_backs_up_existing_file $ok

rm -rf "$WORK"
echo "pass=$pass fail=$fail"; [ $fail -eq 0 ]
