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

# Tests that the deb and rpm preinstall refuses install directories that the package
# scripts could damage the system through, and that postremove skips its cleanup
# for one instead of deleting under it.
PKG=$(dirname "$0")/../../package
WORK=$(mktemp -d)
pass=0; fail=0

# The scripts run against a copy whose service paths point into $WORK, with systemctl
# stubbed, so the tests never touch the host's service manager.
mkdir -p "$WORK/bin"
for c in systemctl service chkconfig useradd groupadd usermod groupmod; do
  printf '#!/bin/sh\nexit 0\n' > "$WORK/bin/$c"; chmod +x "$WORK/bin/$c"
done
PATH="$WORK/bin:$PATH"
for f in preinstall postremove; do
  sed -e "s#\"/usr/lib/systemd/#\"$WORK/usr/lib/systemd/#" -e "s#\"/etc/systemd/#\"$WORK/etc/systemd/#" \
    -e "s#\"/etc/init.d/#\"$WORK/etc/init.d/#" "$PKG/$f.sh" > "$WORK/$f.sh"
done
if grep -nE '"/(usr/lib/systemd|etc/systemd|etc/init\.d)/' "$WORK/postremove.sh"; then
  echo "FAIL service paths not redirected"; exit 1
fi
PKG="$WORK"

check() {
  if [ "$2" = true ]; then pass=$((pass+1)); echo "PASS $1"; else fail=$((fail+1)); echo "FAIL $1"; fi
}

# preinstall <dir>: exit status of preinstall.sh, skipping user creation
preinstall() {
  BDOT_CONFIG_HOME="$1" BDOT_SKIP_RUNTIME_USER_CREATION=true BDOT_USER=root BDOT_GROUP=root sh "$PKG/preinstall.sh" > /dev/null 2>&1
}

for bad in / /usr /etc /var/lib /opt relative/dir /srv/../etc /srv/./x '/srv/my dir' /srv/a:b; do
  preinstall "$bad" && ok=false || ok=true
  check "preinstall_rejects $bad" "$ok"
done
for good in "$WORK/opt/collector" "$WORK/srv/collector/" "$WORK/custom"; do
  preinstall "$good" && ok=true || ok=false
  check "preinstall_accepts $good" "$ok"
done

# postremove leaves a rejected directory alone, and still exits 0 so removal finishes
mkdir -p "$WORK/victim/plugins" "$WORK/x"
BDOT_CONFIG_HOME="$WORK/x/../victim" sh "$PKG/postremove.sh" 0 > /dev/null 2>&1 && ok=true || ok=false
[ -d "$WORK/victim/plugins" ] || ok=false
check postremove_skips_rejected_dir "$ok"

# postremove still cleans up a valid directory
mkdir -p "$WORK/home/plugins" "$WORK/home/storage"; : > "$WORK/home/config.yaml"
BDOT_CONFIG_HOME="$WORK/home" sh "$PKG/postremove.sh" 0 > /dev/null 2>&1 && ok=true || ok=false
[ -e "$WORK/home/plugins" ] && ok=false
[ -f "$WORK/home/config.yaml" ] && [ -d "$WORK/home/storage" ] || ok=false
check postremove_cleans_valid_dir "$ok"

rm -rf "$WORK"
echo "pass=$pass fail=$fail"; [ $fail -eq 0 ]
