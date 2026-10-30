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

# Tests for install_unix.sh AIX package signature verification. Builds throwaway GPG keys,
# then checks both paths: gpg when installed, and base Perl plus openssl when it is not.
# Revoked keys and signatures made after key expiry must fail, since gpg --verify exits 0
# for both and only reports them on its status channel.
# Usage: test_aix_verify.sh [path-to-install_unix.sh]
SCRIPT=${1:-$(dirname "$0")/../install_unix.sh}
WORK=$(mktemp -d)
# Stop every gpg-agent the tests started. ps and awk rather than pkill, which base AIX lacks.
trap 'ps -eo pid,args | awk -v w="gpg-agent --homedir $WORK" "index(\$0, w) { print \$1 }" | xargs kill 2> /dev/null; rm -rf "$WORK"' EXIT
sed '$d' "$SCRIPT" > "$WORK/lib.sh"   # drop trailing main "$@"
pass=0; fail=0
G="$WORK/gnupg"; mkdir -m 700 "$G"
gpgq() { gpg --homedir "$G" --batch --quiet --pinentry-mode loopback --passphrase '' "$@" 2>/dev/null; }
fpr() { gpg --homedir "$G" --with-colons --list-keys "$1" 2>/dev/null | awk -F: '/^fpr/{print $10; exit}'; }
printf 'package-bytes\n' > "$WORK/pkg.tar.gz"

# signer: cert-only primary with a signing subkey, like the BDOT release key
gpgq --quick-gen-key 'Signer <s@x>' rsa2048 cert never; S=$(fpr s@x)
gpgq --quick-add-key "$S" rsa2048 sign 2y
# other: an unrelated key that is not in the bundle
gpgq --quick-gen-key 'Other <o@x>' rsa2048 sign never
# revoked: its revocation certificate ships in deb-revocations/
gpgq --quick-gen-key 'Revoked <r@x>' rsa2048 sign never; R=$(fpr r@x)
# embedded: revoked, and the revocation is part of the exported public key
gpgq --quick-gen-key 'Embedded <e@x>' rsa2048 sign never; E=$(fpr e@x)
# expired: created 2020-01-01, signs 2020-06-01, then expiry is set to 2020-01-03
gpgq --faked-system-time 20200101T000000 --quick-gen-key 'Expired <x@x>' rsa2048 sign never; X=$(fpr x@x)

sign() { gpgq --local-user "$1" $3 --detach-sign --output "$WORK/$2.sig" "$WORK/pkg.tar.gz"; }
sign "$S" subkey
sign "$S" sha512 "--digest-algo SHA512"
sign o@x other
sign "$R" revoked
sign "$E" embedded
gpgq --faked-system-time 20200601T000000 --local-user "$X" --detach-sign --output "$WORK/expired.sig" "$WORK/pkg.tar.gz"
gpgq --faked-system-time 20200102T000000 --quick-set-expire "$X" 1d
sed 's/^:-----BEGIN/-----BEGIN/' "$G/openpgp-revocs.d/$R.rev" > "$WORK/revoked-cert.asc"
gpgq --import < "$(sed 's/^:-----BEGIN/-----BEGIN/' "$G/openpgp-revocs.d/$E.rev" > "$WORK/e.rev" && echo "$WORK/e.rev")"

# run_case <name> <signature> <key fingerprint for the bundle> <with revocation cert: yes|no> <path: gpg|openssl> <expected return code>
# 3 never installs, even interactively; 1 lets an interactive user choose
run_case() {
  name=$1 sig=$2 key=$3 rev=$4 path=$5 expect=$6
  t="$WORK/tmp-$name-$path"; mkdir -p "$t/gpg/deb-revocations"
  gpg --homedir "$G" --armor --export "$key" > "$t/gpg/bdot-public-gpg-key.asc" 2>/dev/null
  [ "$rev" = yes ] && cp "$WORK/revoked-cert.asc" "$t/gpg/deb-revocations/revoked.asc"
  cp "$WORK/pkg.tar.gz" "$t/pkg.tar.gz"; cp "$WORK/$sig.sig" "$t/pkg.tar.gz.gpg.sig"
  [ "$name" = corrupted ] && printf 'x' >> "$t/pkg.tar.gz"
  bin="$WORK/bin-$name-$path"; mkdir -p "$bin"
  for p in $(IFS=:; for d in $PATH; do printf '%s\n' "$d"/*; done); do
    [ -e "$p" ] && [ ! -e "$bin/${p##*/}" ] || continue   # first match on PATH wins, like the shell
    ln -s "$p" "$bin/${p##*/}"; done
  rm -f "$bin/uname"; printf '#!/bin/sh\necho AIX\n' > "$bin/uname"; chmod +x "$bin/uname"
  [ "$path" = gpg ] || rm -f "$bin/gpg" "$bin/gpg2"
  out=$(PATH="$bin" "$bin/sh" -c ". '$WORK/lib.sh'; GPG_DIR='$t/gpg'; package_out_file_path='$t/pkg.tar.gz'; verify_package_aix" 2>&1); rc=$?
  if [ "$rc" -eq "$expect" ]; then
    pass=$((pass+1)); echo "PASS $name ($path)"
  else
    fail=$((fail+1)); echo "FAIL $name ($path, rc=$rc, wanted $expect)"; printf '%s\n' "$out" | tail -3
  fi
}

for path in openssl gpg; do
  run_case subkey    subkey   "$S" no  $path 0
  run_case sha512    sha512   "$S" no  $path 0
  run_case corrupted subkey   "$S" no  $path 3
  run_case unknown   other    "$S" no  $path 1
  run_case revoked   revoked  "$R" yes $path 3
  run_case embedded  embedded "$E" no  $path 3
  run_case expired   expired  "$X" no  $path 3
  # a revocation certificate for a key outside the bundle means the bundle is wrong
  run_case stray-rev subkey   "$S" yes $path 3
done
# Full verify_package on AIX, where base tar has no -z and there is no mktemp: the key bundle must still extract
t="$WORK/tmp-bundle"; mkdir -p "$t/keys/deb-revocations" "$t/gpg"
gpg --homedir "$G" --armor --export "$S" > "$t/keys/bdot-public-gpg-key.asc" 2>/dev/null
(cd "$t/keys" && tar -cf - . | gzip -c > "$t/gpg-keys.tar.gz")   # AIX 7.3 base tar has no -z
cp "$WORK/pkg.tar.gz" "$t/pkg.tar.gz"; cp "$WORK/subkey.sig" "$t/pkg.tar.gz.gpg.sig"
bin="$WORK/bin-bundle"; mkdir -p "$bin"
for p in $(IFS=:; for d in $PATH; do printf '%s\n' "$d"/*; done); do
    [ -e "$p" ] && [ ! -e "$bin/${p##*/}" ] || continue   # first match on PATH wins, like the shell
    ln -s "$p" "$bin/${p##*/}"; done
rm -f "$bin/uname" "$bin/tar" "$bin/gpg" "$bin/gpg2" "$bin/mktemp"   # base AIX has no mktemp
printf '#!/bin/sh\necho AIX\n' > "$bin/uname"
printf '#!/bin/sh\ncase "$1" in *z*) echo "tar: illegal option -- z" >&2; exit 1 ;; esac\nexec %s "$@"\n' "$(command -v tar)" > "$bin/tar"
chmod +x "$bin/uname" "$bin/tar"
out=$(PATH="$bin" "$bin/sh" -c ". '$WORK/lib.sh'; TMP_DIR='$t'; package_type=mkssys; package_out_file_path='$t/pkg.tar.gz'; gpg_tar_out_file_path='$t/gpg-keys.tar.gz'; verify_package" 2>&1); rc=$?
if [ $rc -eq 0 ]; then pass=$((pass+1)); echo "PASS bundle-without-tar-z (openssl)"; else fail=$((fail+1)); echo "FAIL bundle-without-tar-z (rc=$rc)"; printf '%s\n' "$out" | tail -3; fi

# Missing verification tools: warn and install unverified by default, abort with --require-gpg-check
rm -f "$bin/openssl"
out=$(PATH="$bin" "$bin/sh" -c ". '$WORK/lib.sh'; TMP_DIR='$t'; package_type=mkssys; package_out_file_path='$t/pkg.tar.gz'; gpg_tar_out_file_path='$t/gpg-keys.tar.gz'; verify_package" 2>&1); rc=$?
case "$rc $out" in "0 "*"Skipping package signature verification: this system lacks [gpg or openssl]"*) pass=$((pass+1)); echo "PASS no-tools-skips" ;; *) fail=$((fail+1)); echo "FAIL no-tools-skips (rc=$rc)"; printf '%s\n' "$out" | tail -3 ;; esac
out=$(PATH="$bin" "$bin/sh" -c ". '$WORK/lib.sh'; TMP_DIR='$t'; package_type=mkssys; require_gpg_check=true; verification_check; echo reached" 2>&1); rc=$?
case "$rc $out" in *reached*) fail=$((fail+1)); echo "FAIL no-tools-required (rc=$rc)"; printf '%s\n' "$out" | tail -3 ;; *"requires: [gpg or openssl]"*) pass=$((pass+1)); echo "PASS no-tools-required" ;; *) fail=$((fail+1)); echo "FAIL no-tools-required (rc=$rc)"; printf '%s\n' "$out" | tail -3 ;; esac

echo "pass=$pass fail=$fail"; [ $fail -eq 0 ]
