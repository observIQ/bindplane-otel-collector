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

# Tests that install_unix.sh refuses install directories its install and uninstall
# steps could damage the system through, and that it changes and removes only the
# files it installed.
# Usage: test_install_dir.sh [path-to-install_unix.sh]
SCRIPT=${1:-$(dirname "$0")/../install_unix.sh}
WORK=$(mktemp -d)
sed '$d' "$SCRIPT" > "$WORK/lib.sh"   # drop trailing main "$@"
pass=0; fail=0
QUIET="non_interactive=true; indent=''"

# Stub the service and package managers, so no test reaches the host's
mkdir -p "$WORK/bin"
for c in systemctl service chkconfig rpm dpkg curl wget startsrc stopsrc mkssys rmssys mkitab rmitab; do
  printf '#!/bin/sh\nexit 0\n' > "$WORK/bin/$c"; chmod +x "$WORK/bin/$c"
done
PATH="$WORK/bin:$PATH"

check() {
  if [ "$2" = true ]; then pass=$((pass+1)); echo "PASS $1"; else fail=$((fail+1)); echo "FAIL $1"; fi
}

# guard <os> <dir>: exit status of check_install_dir, and the normalized dir on stdout
guard() {
  sh -c ". '$WORK/lib.sh'; $QUIET; OS_TYPE='$1'; BDOT_CONFIG_HOME='$2'; check_install_dir && command printf '%s' \"\$BDOT_CONFIG_HOME\"" 2>/dev/null
}

for bad in relative/dir ./x /srv/../etc /srv/./x /srv/x/.. '/srv/my dir' '/srv/a"b' /srv/a:b \
           / /opt /opt/ //opt /usr /usr/local /var /var/lib /etc /tmp /home /root /bin /lib64 /boot /proc /sys /dev /run; do
  guard Linux "$bad" > /dev/null && ok=false || ok=true
  check "linux_rejects $bad" $ok
done
for bad in /usr/lpp /admin /lpp /usr/sys; do
  guard AIX "$bad" > /dev/null && ok=false || ok=true
  check "aix_rejects $bad" $ok
done
[ "$(guard Linux /srv/dbdot/)" = /srv/dbdot ] && ok=true || ok=false
check accepts_and_normalizes_custom_dir $ok
[ "$(sh -c ". '$WORK/lib.sh'; $QUIET; OS_TYPE=AIX; BDOT_CONFIG_HOME=/opt/observiq-otel-collector; check_install_path && command printf '%s' \"\$BDOT_CONFIG_HOME\"" 2>/dev/null)" = /opt/observiq-otel-collector ] && ok=true || ok=false
check accepts_default_dir $ok

# an existing directory must be empty or hold a file a collector install writes
d="$WORK/notempty"; mkdir -p "$d"; echo x > "$d/notes.txt"
guard Linux "$d" > /dev/null && ok=false || ok=true
check rejects_unrelated_nonempty_dir $ok
d="$WORK/install"; mkdir -p "$d"; : > "$d/observiq-otel-collector"; : > "$d/notes.txt"
guard Linux "$d" > /dev/null && ok=true || ok=false
check accepts_existing_install $ok
d="$WORK/leftovers"; mkdir -p "$d/log" "$d/storage"; : > "$d/config.yaml"; : > "$d/manager.yaml"; : > "$d/logging.yaml"
guard Linux "$d" > /dev/null && ok=true || ok=false
check accepts_uninstall_leftovers $ok
d="$WORK/empty"; mkdir -p "$d"
guard Linux "$d" > /dev/null && ok=true || ok=false
check accepts_empty_dir $ok
d="$WORK/old-leftovers"; mkdir -p "$d"; : > "$d/gpg-keys.tar.gz"; : > "$d/manager.yaml"; : > "$d/notes.txt"
guard Linux "$d" > /dev/null && ok=true || ok=false
check accepts_leftovers_with_other_files $ok

# install and uninstall refuse a system directory on every OS before doing anything
for os in Linux AIX; do
  for fn in install_package uninstall; do
    out=$(sh -c ". '$WORK/lib.sh'; $QUIET; OS_TYPE='$os'; BDOT_CONFIG_HOME=/usr; $fn" 2>&1)
    case "$out" in *"system directory /usr"*) ok=true ;; *) ok=false ;; esac
    check "${fn}_guards_dir_$os" $ok
  done
done

# uninstall checks only the path, so a partial install can still be removed
d="$WORK/partial"; mkdir -p "$d/plugins"; : > "$d/plugins/a.yaml"
sh -c ". '$WORK/lib.sh'; $QUIET; BDOT_CONFIG_HOME='$d'; check_install_path" > /dev/null 2>&1 && ok=true || ok=false
check uninstall_path_check_accepts_partial_install $ok

# make_install <dir>: an installed collector with runtime files and a user file
make_install() {
  mkdir -p "$1/plugins" "$1/install" "$1/log" "$1/storage" "$1/.staging"
  for f in observiq-otel-collector updater LICENSE VERSION.txt gpg-keys.tar.gz package_statuses.json config.yaml logging.yaml manager.yaml plugins/a.yaml install/x.env log/collector.log storage/queue notes.txt; do : > "$1/$f"; done
}

# uninstall removes the named collector files and keeps configs, storage and anything unknown
d="$WORK/uninstall"; make_install "$d"
sh -c ". '$WORK/lib.sh'; $QUIET; BDOT_CONFIG_HOME='$d'; AIX_ENV_FILE='$WORK/none.env'; package_type=mkssys; uninstall_package" > /dev/null 2>&1
ok=true
for f in config.yaml logging.yaml manager.yaml storage/queue notes.txt; do [ -f "$d/$f" ] || ok=false; done
for f in observiq-otel-collector updater LICENSE VERSION.txt gpg-keys.tar.gz package_statuses.json plugins log install .staging; do [ -e "$d/$f" ] && ok=false; done
check uninstall_removes_named_files_keeps_configs_and_storage $ok

# uninstall of a directory holding only collector files removes the directory
d="$WORK/uninstall-clean"; make_install "$d"; rm -rf "$d/storage" "$d/notes.txt" "$d"/*.yaml
sh -c ". '$WORK/lib.sh'; $QUIET; BDOT_CONFIG_HOME='$d'; AIX_ENV_FILE='$WORK/none.env'; package_type=mkssys; uninstall_package" > /dev/null 2>&1
[ -e "$d" ] && ok=false || ok=true
check uninstall_removes_emptied_dir $ok

# ownership of the install directory is set item by item, never recursively
d="$WORK/chown"; mkdir -p "$d/log" "$d/storage"; : > "$d/config.yaml"
bin="$WORK/bin"; mkdir -p "$bin"; printf '#!/bin/sh\necho "chown $*" >> "%s"\n' "$WORK/chown.log" > "$bin/chown"; chmod +x "$bin/chown"
PATH="$bin:$PATH" sh -c ". '$WORK/lib.sh'; $QUIET; BDOT_CONFIG_HOME='$d'; COLLECTOR_USER=dbdot; COLLECTOR_GROUP=dbdot; aix_chown_install" > /dev/null 2>&1
ok=true
grep -q -- '-R' "$WORK/chown.log" && ok=false
for p in "$d" "$d/log" "$d/storage" "$d/config.yaml"; do grep -qx "chown dbdot:dbdot $p" "$WORK/chown.log" || ok=false; done
check chown_install_without_recursion $ok

# nothing in the script deletes or re-owns the install directory itself recursively;
# named subdirectories under it are fine
no_recursive_home() { ! grep -nE '(rm -[a-z]*r|chown -R|chmod -R)[^#]*\$\{?BDOT_CONFIG_HOME[:?}]*/?(\*|"|$| )' "$1" > /dev/null; }
no_recursive_home "$SCRIPT" && ok=true || ok=false
check no_recursive_delete_or_chown_of_install_dir $ok
printf 'x() {\n  rm -rf "$BDOT_CONFIG_HOME"\n}\n' > "$WORK/bad.sh"
no_recursive_home "$WORK/bad.sh" && ok=false || ok=true
check recursive_check_catches_home_delete $ok

rm -rf "$WORK"
echo "pass=$pass fail=$fail"; [ $fail -eq 0 ]
