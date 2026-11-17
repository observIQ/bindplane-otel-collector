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

# Tests that the generic install scripts accept only a plain distribution name, so
# the install directory is always a direct child of /opt, and that uninstall removes
# only the files the tarball and the script installed, never the whole directory.
GEN=$(dirname "$0")/../../generic-install
WORK=$(mktemp -d)
pass=0; fail=0

check() {
  if [ "$2" = true ]; then pass=$((pass+1)); echo "PASS $1"; else fail=$((fail+1)); echo "FAIL $1"; fi
}

# Stub the service managers, so no test reaches the host's
mkdir -p "$WORK/bin"
for c in systemctl launchctl dpkg rpm curl; do
  printf '#!/bin/sh\nexit 0\n' > "$WORK/bin/$c"; chmod +x "$WORK/bin/$c"
done
PATH="$WORK/bin:$PATH"

# A distribution tarball with a nested directory
mkdir -p "$WORK/pkg/plugins" "$WORK/pkg/service"
: > "$WORK/pkg/myotel"; : > "$WORK/pkg/plugins/a.yaml"; : > "$WORK/pkg/service/myotel.plist"
(cd "$WORK/pkg" && tar czf "$WORK/myotel.tar.gz" .)

for os in unix macos; do
  sed '$d' "$GEN/install_$os.sh" > "$WORK/lib-$os.sh"   # drop trailing main "$@"
  lib="$WORK/lib-$os.sh"

  for bad in '' . .. ../etc a/b '-x' 'my otel' '*'; do
    sh -c ". '$lib'; DISTRIBUTION='$bad'; verify_distribution" > /dev/null 2>&1 && ok=false || ok=true
    check "${os}_rejects_distribution '$bad'" $ok
  done
  out=$(sh -c ". '$lib'; DISTRIBUTION=my-otel_1.2; verify_distribution; echo \"\$INSTALL_DIR\"" 2>/dev/null)
  [ "$out" = /opt/my-otel_1.2 ] && ok=true || ok=false
  check "${os}_accepts_plain_distribution" $ok

  # install records the tarball's files; uninstall removes them and the script's own,
  # keeps anything else, and removes the directory once it is empty
  d="$WORK/$os-opt/myotel"; mkdir -p "$d"
  sh -c ". '$lib'; DISTRIBUTION=myotel; INSTALL_DIR='$d'; install_tarball '$WORK/myotel.tar.gz'" > /dev/null 2>&1
  [ -f "$d/plugins/a.yaml" ] && [ -f "$d/.install-files" ] && ok=true || ok=false
  check "${os}_install_records_files" $ok
  mkdir -p "$d/supervisor_storage"; : > "$d/supervisor_config.yaml"; : > "$d/supervisor.log"; : > "$d/supervisor_storage/x"
  : > "$d/notes.txt"
  sh -c ". '$lib'; DISTRIBUTION=myotel; INSTALL_DIR='$d'; remove_install_dir" > /dev/null 2>&1
  ok=true
  [ -f "$d/notes.txt" ] || ok=false
  for f in myotel plugins service supervisor_storage supervisor_config.yaml supervisor.log .install-files; do [ -e "$d/$f" ] && ok=false; done
  check "${os}_uninstall_removes_installed_files_keeps_others" $ok
  rm -f "$d/notes.txt"
  sh -c ". '$lib'; DISTRIBUTION=myotel; INSTALL_DIR='$d'; remove_install_dir" > /dev/null 2>&1
  [ -e "$d" ] && ok=false || ok=true
  check "${os}_uninstall_removes_emptied_dir" $ok

  # without a file list, uninstall removes only the script's own files
  d="$WORK/$os-old/myotel"; mkdir -p "$d/plugins" "$d/supervisor_storage"; : > "$d/myotel"; : > "$d/supervisor_config.yaml"
  sh -c ". '$lib'; DISTRIBUTION=myotel; INSTALL_DIR='$d'; remove_install_dir" > /dev/null 2>&1
  [ -f "$d/myotel" ] && [ -d "$d/plugins" ] && [ ! -e "$d/supervisor_config.yaml" ] && [ ! -e "$d/supervisor_storage" ] && ok=true || ok=false
  check "${os}_uninstall_without_file_list_keeps_unknown_files" $ok

  # file list entries can't reach outside the install directory
  d="$WORK/$os-esc/myotel"; mkdir -p "$d"; : > "$WORK/$os-esc/victim"
  printf '%s\n' ../victim /etc/hostname > "$d/.install-files"
  sh -c ". '$lib'; DISTRIBUTION=myotel; INSTALL_DIR='$d'; remove_install_dir" > /dev/null 2>&1
  [ -f "$WORK/$os-esc/victim" ] && ok=true || ok=false
  check "${os}_uninstall_ignores_escaping_entries" $ok

  grep -nE 'rm -[a-z]*r[a-z]* "?\$\{?INSTALL_DIR\}?"?( |$)' "$GEN/install_$os.sh" > /dev/null && ok=false || ok=true
  check "${os}_no_recursive_delete_of_install_dir" $ok
done

rm -rf "$WORK"
echo "pass=$pass fail=$fail"; [ $fail -eq 0 ]
