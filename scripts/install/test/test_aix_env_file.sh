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

# Tests that install_unix.sh installs the AIX env file: backs up an existing one,
# and points it at a custom install directory.
# Usage: test_aix_env_file.sh [path-to-install_unix.sh]
SCRIPT=${1:-$(dirname "$0")/../install_unix.sh}
WORK=$(mktemp -d)
sed '$d' "$SCRIPT" > "$WORK/lib.sh"   # drop trailing main "$@"
pass=0; fail=0
# A quiet install makes the script's printf wrapper print nothing, and an interactive
# one indents. Neither may leak into the env file or the install directory checks.
QUIET="non_interactive=true; indent='    '"

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

# custom install directory: the packaged default is replaced with BDOT_CONFIG_HOME
dir="$WORK/custom"; mkdir -p "$dir"
echo "H=/opt/observiq-otel-collector S=/opt/observiq-otel-collector/storage" > "$dir/pkg.env"
sh -c ". '$WORK/lib.sh'; $QUIET; AIX_ENV_FILE='$dir/collector.env'; BDOT_CONFIG_HOME=/srv/bdot; install_aix_env_file '$dir/pkg.env'" > /dev/null 2>&1
ok=true
[ "$(cat "$dir/collector.env")" = "H=/srv/bdot S=/srv/bdot/storage" ] || ok=false
check custom_home_rewrites_env_file $ok

# only values under the default directory move, and shell or awk metacharacters in the path stay literal
dir="$WORK/custom-meta"; mkdir -p "$dir"
echo "H=/opt/observiq-otel-collector O=/opt/observiq-otel-collector-other X=keep" > "$dir/pkg.env"
sh -c ". '$WORK/lib.sh'; $QUIET; AIX_ENV_FILE='$dir/collector.env'; BDOT_CONFIG_HOME='/srv/a&b\\c'; install_aix_env_file '$dir/pkg.env'" > /dev/null 2>&1
ok=true
[ "$(cat "$dir/collector.env")" = 'H=/srv/a&b\c O=/opt/observiq-otel-collector-other X=keep' ] || ok=false
check custom_home_replaces_only_default_prefix $ok

# AIX can't use an install directory with whitespace or a double quote
for bad in '/srv/my bdot' '/srv/a"b' '/srv/a	b'; do
  if sh -c ". '$WORK/lib.sh'; $QUIET; BDOT_CONFIG_HOME='$bad'; check_aix_config_home" > /dev/null 2>&1; then
    check "rejects_unsafe_home $bad" false
  else
    check "rejects_unsafe_home $bad" true
  fi
done
sh -c ". '$WORK/lib.sh'; $QUIET; BDOT_CONFIG_HOME='/srv/bdot-custom'; check_aix_config_home" > /dev/null 2>&1 && ok=true || ok=false
check accepts_plain_home $ok

# uninstall: removes the env file and every backup, and nothing else in its directory
dir="$WORK/uninstall"; mkdir -p "$dir/home"
echo "A=new" > "$dir/collector.env"
echo "A=old" > "$dir/collector.env.20261028120000.bak"
echo "A=older" > "$dir/collector.env.20261027120000.bak"
echo "keep" > "$dir/other.env"
sh -c ". '$WORK/lib.sh'; AIX_ENV_FILE='$dir/collector.env'; BDOT_CONFIG_HOME='$dir/home'; package_type=mkssys; uninstall_package" > /dev/null 2>&1
ok=true
for f in "$dir"/collector.env*; do [ -e "$f" ] && ok=false; done
[ -f "$dir/other.env" ] || ok=false
[ -e "$dir/home" ] && ok=false
check uninstall_removes_env_file_and_backups $ok

rm -rf "$WORK"
echo "pass=$pass fail=$fail"; [ $fail -eq 0 ]
