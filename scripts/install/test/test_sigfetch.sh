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

# Tests for install_unix.sh network signature fetch.
# Stubs curl to log each URL and output path, and stubs verify_package to stop right after downloads.
# Usage: test_sigfetch.sh [path-to-install_unix.sh]
SCRIPT=${1:-$(dirname "$0")/../install_unix.sh}
WORK=$(mktemp -d)
sed '$d' "$SCRIPT" > "$WORK/lib.sh"   # drop trailing main "$@"
pass=0; fail=0
URL=https://bdot.example/v1.110.1/observiq-otel-collector-v1.110.1-aix-ppc64.tar.gz

# run_case <name> <os> <package_type> <setup shell> <expectations...>
# expectations: sig-fetched | no-sig-fetch | proxy-on-sig
run_case() {
  name=$1 os=$2 ptype=$3 setup=$4; shift 4
  bin="$WORK/bin-$name"; log="$WORK/log-$name"; mkdir -p "$bin"; : > "$log"
  for p in $(IFS=:; for d in $PATH; do printf '%s\n' "$d"/*; done); do
    [ -e "$p" ] && [ ! -e "$bin/${p##*/}" ] || continue   # first match on PATH wins, like the shell
    ln -s "$p" "$bin/${p##*/}"; done
  rm -f "$bin/uname" "$bin/curl"
  printf '#!/bin/sh\necho %s\n' "$os" > "$bin/uname"
  cat > "$bin/curl" <<EOF
#!/bin/sh
echo "curl \$*" >> "$log"
while [ \$# -gt 0 ]; do [ "\$1" = "-o" ] && : > "\$2"; shift; done
exit 0
EOF
  chmod +x "$bin/uname" "$bin/curl"
  PATH="$bin" "$bin/sh" -c ". '$WORK/lib.sh'
    verify_package() { echo VERIFY_REACHED; return 1; }
    non_interactive=true; offline_installation=false; package_type=$ptype
    collector_download_url=$URL; package_out_file_path=$WORK/pkg-$name.tar.gz
    gpg_tar_download_url=https://bdot.example/v2.0.1/gpg-keys.tar.gz; gpg_tar_out_file_path=$WORK/keys-$name.tar.gz
    $setup
    install_package" > "$WORK/out-$name" 2>&1
  ok=true
  for e in "$@"; do
    case $e in
      sig-fetched)  grep -qF "$URL.gpg.sig -o $WORK/pkg-$name.tar.gz.gpg.sig" "$log" || ok=false ;;
      no-sig-fetch) grep -q 'gpg.sig' "$log" && ok=false ;;
      proxy-on-sig) grep 'gpg.sig' "$log" | grep -q -- '-x http://proxy:3128' || ok=false ;;
    esac
  done
  if $ok; then pass=$((pass+1)); echo "PASS $name"; else fail=$((fail+1)); echo "FAIL $name"; sed 's/^/  log: /' "$log"; fi
}

run_case aix-network        AIX   mkssys ":"                                 sig-fetched
run_case aix-network-proxy  AIX   mkssys "proxy=http://proxy:3128"           sig-fetched proxy-on-sig
run_case aix-no-gpg-check   AIX   mkssys "skip_gpg_check=true"               no-sig-fetch
run_case aix-offline        AIX   mkssys "offline_installation=true"         no-sig-fetch
run_case linux-rpm-network  Linux rpm    ":"                                 no-sig-fetch
rm -rf "$WORK"
echo "pass=$pass fail=$fail"; [ $fail -eq 0 ]
