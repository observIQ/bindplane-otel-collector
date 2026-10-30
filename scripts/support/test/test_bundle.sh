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

# End-to-end runs of create-support-bundle.sh against stubbed AIX and Linux
# tools and fake v1 and v2 installs. Checks the bundle's contents and that no
# seeded secret survives.
# Usage: test_bundle.sh [path-to-create-support-bundle.sh]
SCRIPT=${1:-$(dirname "$0")/../create-support-bundle.sh}
case "$SCRIPT" in /*) ;; *) SCRIPT="$(pwd)/$SCRIPT" ;; esac
. "$(dirname "$0")/stubs.sh"
WORK=$(mktemp -d 2>/dev/null) || { WORK=/tmp/sbtest.$$; mkdir "$WORK"; }   # AIX has no mktemp
pass=0; fail=0
n=0

# fake_install <dir> <version>
fake_install() {
  mkdir -p "$1"
  if [ "$2" = v1 ]; then
    mkdir -p "$1/log"
    echo "log line password: logsecret" > "$1/log/collector.log"
    echo "panic: boom" > "$1/log/observiq_collector.err"
    # The collector recreates its error log on every start, so it is usually
    # the newest file. The recent log must still be collector.log.
    touch -t 202601010000 "$1/log/collector.log"
    echo "token: configsecret" > "$1/config.yaml"
    echo "secret_key: managersecret" > "$1/manager.yaml"
  else
    mkdir -p "$1/supervisor_storage"
    echo "supervisor line token=logsecret" > "$1/supervisor.log"
    echo "agent line password: logsecret" > "$1/supervisor_storage/agent.log"
    printf 'server:\n  headers:\n    Authorization: Secret-Key managersecret\n' > "$1/supervisor.yaml"
    echo "api_key: configsecret" > "$1/supervisor_storage/effective.yaml"
  fi
}

# run_case <name> <os> <version> <expected files in the bundle>
run_case() {
  name=$1 os=$2 version=$3 want=$4
  n=$((n+1)); bin="$WORK/bin$n"; state="$WORK/state$n"; out="$WORK/out$n"; dir="$WORK/install$n"
  make_stubs "$bin" "$state" "$os"
  if [ "$version" = v1 ]; then svc=observiq-otel-collector; exe=observiq-otel-collector; else svc=bindplane-otel-collector; exe=opampsupervisor; fi
  fake_install "$dir" "$version"
  echo "$dir/$exe 4242" > "$state/$svc"
  mkdir -p "$out" "$WORK/proc$n/4242/fd"
  echo "Max open files 2000 2000 files" > "$WORK/proc$n/4242/limits"
  # recent logs, include configs, no pprof, collect handles and limits
  (cd "$out" && printf 'y\ny\nn\ny\n' | PROC="$WORK/proc$n" PATH="$bin:$PATH" sh "$SCRIPT" > "$out.log" 2>&1)
  bundle=$(ls "$out"/support_bundle_*.tar.gz 2>/dev/null)
  if [ -z "$bundle" ]; then fail=$((fail+1)); echo "FAIL $name (no bundle)"; tail -5 "$out.log"; return; fi
  mkdir "$out/x" && gzip -dc "$bundle" | (cd "$out/x" && tar -xf -)
  missing=""
  for f in $want; do [ -f "$out/x/$f" ] || missing="$missing $f"; done
  leaked=$(grep -rlE 'logsecret|configsecret|managersecret|journalsecret|envsecret' "$out/x")
  if [ -z "$missing" ] && [ -z "$leaked" ]; then
    pass=$((pass+1)); echo "PASS $name"
  else
    fail=$((fail+1)); echo "FAIL $name (missing:${missing:- none}; leaked: ${leaked:-none})"
  fi
}

run_case aix-v1 AIX v1 \
  "collector.log observiq_collector.err config.yaml manager.yaml src_status.log oslevel.txt user_limits.txt procfiles.txt system_stats.txt"
run_case aix-v2 AIX v2 \
  "supervisor.log agent.log supervisor.yaml effective.yaml src_status.log oslevel.txt user_limits.txt procfiles.txt system_stats.txt"
# Simulated Linux needs the host's Linux tools (GNU sed), so not on AIX.
if [ "$(uname -s)" != AIX ]; then
run_case linux-v1 Linux v1 \
  "collector.log observiq_collector.err config.yaml manager.yaml journalctl.log systemd_limits.txt proc_limits.txt system_stats.txt"
run_case linux-v2 Linux v2 \
  "supervisor.log agent.log supervisor.yaml effective.yaml journalctl.log systemd_limits.txt proc_limits.txt system_stats.txt"
fi

rm -rf "$WORK"
echo "pass=$pass fail=$fail"; [ $fail -eq 0 ]
