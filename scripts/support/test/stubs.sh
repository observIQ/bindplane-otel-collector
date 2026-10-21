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

# Shared stubs for the create-support-bundle.sh tests. Each installed collector
# is a file $STATE/<service> holding "<binary path> <pid>"; an empty pid means stopped.
# Usage: . stubs.sh; make_stubs <bin dir> <state dir> <uname -s>
make_stubs() {
  bin=$1 state=$2 os=$3
  mkdir -p "$bin" "$state"
  stub() { printf '#!/bin/sh\nSTATE=%s\n%s\n' "$state" "$2" > "$bin/$1"; chmod +x "$bin/$1"; }
  stub uname "case \"\$1\" in -p) echo powerpc ;; -m) echo x86_64 ;; *) echo $os ;; esac"
  stub bootinfo 'echo 64'
  stub id 'case "$1" in -un) echo root ;; -u) echo 0 ;; esac'
  stub oslevel 'echo 7300-04-00-2546'
  stub lsuser 'echo "root nofiles=-1 fsize=-1"'
  stub procfiles 'echo "$2 : collector"; echo "  Current rlimit: 2000 file descriptors"'
  stub journalctl 'echo "journal line password=journalsecret"'
  stub lssrc '
eval "svc=\${$#}"
if [ ! -f "$STATE/$svc" ]; then echo "0513-085 The $svc Subsystem is not on file." >&2; exit 1; fi
read -r path pid < "$STATE/$svc"
if [ "$1" = -S ]; then
  echo "#subsysname:synonym:cmdargs:path:uid:auditid:standin:standout:standerr:"
  echo "$svc::--config x:$path:0:0:/dev/console:/dev/console:/dev/console:"
  exit 0
fi
echo "Subsystem         Group            PID          Status"
if [ -n "$pid" ]; then echo " $svc    grp    $pid    active"; else echo " $svc    grp        inoperative"; fi'
  stub systemctl '
case "$1" in
  list-unit-files) svc=${3%.service}; [ -f "$STATE/$svc" ] && echo "$svc.service enabled disabled" ;;
  show)
    svc=${2%.service}
    [ -f "$STATE/$svc" ] || exit 0
    read -r path pid < "$STATE/$svc"
    case "$4" in
      ExecStart) echo "{ path=$path ; argv[]=$path --config x ; }" ;;
      MainPID) echo "${pid:-0}" ;;
      *) echo "LimitNOFILE=65536"; echo "Environment=OPAMP_SECRET_KEY=envsecret" ;;
    esac ;;
esac'
}
