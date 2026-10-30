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

# Redaction must give the same output everywhere, including with AIX's sed,
# which has no -E, -i or I flag. The golden file is what the GNU sed
# implementation produced before the move to awk.
# Usage: test_redact.sh [path-to-create-support-bundle.sh]
SCRIPT=${1:-$(dirname "$0")/../create-support-bundle.sh}
DATA=$(dirname "$0")/testdata
WORK=$(mktemp -d 2>/dev/null) || { WORK=/tmp/sbtest.$$; mkdir "$WORK"; }   # AIX has no mktemp
sed '$d' "$SCRIPT" > "$WORK/lib.sh"   # drop trailing main "$@"
pass=0; fail=0

# run_case <name> <PATH>
run_case() {
  cp "$DATA/redact-input.txt" "$WORK/in.txt"
  PATH="$2" sh -c ". '$WORK/lib.sh'; redact_in_place '$WORK/in.txt'" > "$WORK/out" 2>&1
  if cmp -s "$WORK/in.txt" "$DATA/redact-golden.txt"; then
    pass=$((pass+1)); echo "PASS $1"
  else
    fail=$((fail+1)); echo "FAIL $1"; diff "$DATA/redact-golden.txt" "$WORK/in.txt"; cat "$WORK/out"
  fi
}

run_case system-tools "$PATH"

# Stand in for AIX sed: basic sed only.
mkdir -p "$WORK/aixbin"
real_sed=$(command -v sed)
cat > "$WORK/aixbin/sed" <<STUB
#!/bin/sh
for a in "\$@"; do
  case "\$a" in -E|-r|-i|--*) echo "sed: illegal option -- \${a#-}" >&2; exit 2 ;; esac
done
exec "$real_sed" "\$@"
STUB
chmod +x "$WORK/aixbin/sed"
run_case aix-sed "$WORK/aixbin:$PATH"

rm -rf "$WORK"
echo "pass=$pass fail=$fail"; [ $fail -eq 0 ]
