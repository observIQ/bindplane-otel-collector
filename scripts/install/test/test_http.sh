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

# Tests for install_unix.sh HTTP helpers, which fall back from curl to base Perl
# so AIX installs work without the AIX Toolbox.
# Usage: test_http.sh [path-to-install_unix.sh]
SCRIPT=${1:-$(dirname "$0")/../install_unix.sh}
WORK=$(mktemp -d)
sed '$d' "$SCRIPT" > "$WORK/lib.sh"   # drop trailing main "$@"
pass=0; fail=0

# Serve a small file over plain HTTP on localhost for the download cases.
mkdir -p "$WORK/www" && printf 'payload-123\n' > "$WORK/www/pkg.bin"
PORT=$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1",0)); print(s.getsockname()[1]); s.close()')
python3 -m http.server "$PORT" --bind 127.0.0.1 --directory "$WORK/www" > /dev/null 2>&1 &
SERVER=$!
trap 'kill $SERVER 2>/dev/null; rm -rf "$WORK"' EXIT
i=0; until python3 -c "import urllib.request; urllib.request.urlopen('http://127.0.0.1:$PORT/pkg.bin')" 2>/dev/null; do i=$((i+1)); [ $i -gt 50 ] && break; sleep 0.1; done

# run_case <name> <os> <omit tools> <shell to run after sourcing> <expect: ok|fail> [must-mention]
run_case() {
  name=$1 os=$2 omit=$3 body=$4 expect=$5 mention=$6
  if [ "$os" = Linux ] && [ "$(uname)" = AIX ]; then echo "SKIP $name (needs a Linux host: a faked uname can't swap AIX's own tools)"; return; fi
  bin="$WORK/bin-$name"; mkdir -p "$bin"
  for p in $(IFS=:; for d in $PATH; do printf '%s\n' "$d"/*; done); do
    [ -e "$p" ] && [ ! -e "$bin/${p##*/}" ] || continue   # first match on PATH wins, like the shell
    t=${p##*/}
    case " $omit " in *" $t "*) continue ;; esac
    ln -s "$p" "$bin/$t"
  done
  rm -f "$bin/uname"; printf '#!/bin/sh\necho %s\n' "$os" > "$bin/uname"; chmod +x "$bin/uname"
  out=$(PATH="$bin" "$bin/sh" -c ". '$WORK/lib.sh'; $body" 2>&1); rc=$?
  ok=true
  if [ "$expect" = ok ]; then [ $rc -eq 0 ] || ok=false; else [ $rc -ne 0 ] || ok=false; fi
  [ -z "$mention" ] || printf '%s' "$out" | grep -q -- "$mention" || ok=false
  if $ok; then pass=$((pass+1)); echo "PASS $name"; else fail=$((fail+1)); echo "FAIL $name (rc=$rc)"; printf '%s\n' "$out" | tail -4; fi
}

URL=http://127.0.0.1:$PORT/pkg.bin
OUT=$WORK/out.bin
run_case prereqs-no-curl-no-sudo AIX   "curl sudo"      "dependencies_check" ok
run_case prereqs-no-http-client  AIX   "curl sudo perl" "dependencies_check" fail "curl or perl"
run_case download-with-perl      AIX   "curl sudo"      "dependencies_check >/dev/null; http_download $URL $OUT && grep -qx payload-123 $OUT" ok
run_case download-with-curl      Linux ""               "dependencies_check >/dev/null; http_download $URL $OUT.c && grep -qx payload-123 $OUT.c" ok
run_case download-404-with-perl  AIX   "curl sudo"      "dependencies_check >/dev/null; http_download $URL.missing $OUT.m" fail
run_case get-with-perl           AIX   "curl sudo"      "dependencies_check >/dev/null; [ \"\$(http_get $URL)\" = payload-123 ]" ok
run_case check-with-perl         AIX   "curl sudo"      "dependencies_check >/dev/null; http_check $URL" ok
run_case check-refused-with-perl AIX   "curl sudo"      "dependencies_check >/dev/null; http_check http://127.0.0.1:1/" fail
echo "pass=$pass fail=$fail"; [ $fail -eq 0 ]
