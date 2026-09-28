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

# Tests for install_unix.sh signature checks, which must hold under any locale since gpg
# translates its warnings. Needs gpg, ar, rpm, and de_DE.UTF-8, and fails rather than skips
# without them.
# Usage: test_gpg_verify.sh [path-to-install_unix.sh]
SCRIPT=$(cd "$(dirname "${1:-$(dirname "$0")/../install_unix.sh}")" && pwd)/$(basename "${1:-install_unix.sh}")
DATA=$(cd "$(dirname "$0")" && pwd)/testdata
WORK=$(mktemp -d)
# Stop every gpg-agent the tests started, including those whose gpgconf the tests hide
trap 'pkill -u "$(id -u)" -f "gpg-agent --homedir $WORK" 2> /dev/null; rm -rf "$WORK"' EXIT
fail=0; pass=0
for tool in gpg ar rpm; do
  command -v "$tool" > /dev/null 2>&1 || { echo "FAIL setup: $tool is not installed"; exit 1; }
done
if ! locale -a 2>/dev/null | grep -qix 'de_DE.utf8'; then
  echo "FAIL setup: the de_DE.UTF-8 locale is not installed, so the locale cases would test nothing"
  exit 1
fi
# On Ubuntu the translations come from language-pack-de-base, on Debian from gnupg-l10n
if ! LC_ALL=de_DE.UTF-8 gpg --help 2>/dev/null | grep -q '^Befehle:'; then
  echo "FAIL setup: gpg prints no German under de_DE.UTF-8, so the locale cases would test nothing"
  exit 1
fi
sed '/^main "\$@"$/d' "$SCRIPT" > "$WORK/lib.sh"
grep -q '^main "\$@"$' "$SCRIPT" || { echo "FAIL setup: could not find the main call to strip"; exit 1; }

G="$WORK/gnupg"; mkdir -m 700 "$G"
gpgq() { gpg --homedir "$G" --batch --quiet --pinentry-mode loopback --passphrase '' "$@" 2>/dev/null; }
fpr() { gpg --homedir "$G" --with-colons --list-keys "$1" 2>/dev/null | awk -F: '/^fpr/{print $10; exit}'; }
rev() { sed 's/^:-----BEGIN/-----BEGIN/' "$G/openpgp-revocs.d/$1.rev"; }
REAL_RPM=$(command -v rpm)

# Keys: a cert-only primary with a signing subkey (like BDOT's), an unrelated key, revoked by
# certificate and embedded, expired after signing (still valid), expired before signing, and
# one whose signature carries its own expiry.
gpgq --quick-gen-key 'Signer <s@x>' rsa2048 cert never; S=$(fpr s@x); gpgq --quick-add-key "$S" rsa2048 sign 2y
gpgq --quick-gen-key 'Other <o@x>' rsa2048 sign never
gpgq --quick-gen-key 'Revoked <r@x>' rsa2048 sign never; R=$(fpr r@x)
gpgq --quick-gen-key 'Embedded <e@x>' rsa2048 sign never; E=$(fpr e@x)
gpgq --faked-system-time 20200101T000000 --quick-gen-key 'Lapsed <l@x>' rsa2048 sign 1y; L=$(fpr l@x)
gpgq --faked-system-time 20200101T000000 --quick-gen-key 'Late <x@x>' rsa2048 sign never; X=$(fpr x@x)
gpgq --faked-system-time 20191201T000000 --quick-gen-key 'Expsig <p@x>' rsa2048 sign never; P=$(fpr p@x)
# The BDOT flow: a cert-only primary with a signing subkey, and the primary revoked later
gpgq --quick-gen-key 'Primrev <pr@x>' rsa2048 cert never; PR=$(fpr pr@x); gpgq --quick-add-key "$PR" rsa2048 sign never
# Primaries that stay valid while only the signing subkey is revoked, or expired before it signed
gpgq --quick-gen-key 'Subrev <sr@x>' rsa2048 cert never; SR=$(fpr sr@x); gpgq --quick-add-key "$SR" rsa2048 sign never
gpgq --faked-system-time 20200101T000000 --quick-gen-key 'Sublate <sl@x>' rsa2048 cert never; SL=$(fpr sl@x)
gpgq --faked-system-time 20200101T000000 --quick-add-key "$SL" rsa2048 sign never

# A minimal .deb: _gpgorigin signs debian-binary + control.tar.gz + data.tar.gz
cd "$WORK" || exit 1
printf '2.0\n' > debian-binary; printf 'c' | gzip > control.tar.gz; printf 'd' | gzip > data.tar.gz
cat debian-binary control.tar.gz data.tar.gz > payload
sign() { gpgq $4 --local-user "$1" $3 --detach-sign --output "$WORK/$2" "$WORK/payload"; }
sign "$S" good.sig
sign o@x other.sig
sign "$R" revoked.sig
sign "$E" embedded.sig
sign "$L" lapsed.sig "" "--faked-system-time 20200601T000000"
sign "$X" late.sig "" "--faked-system-time 20200601T000000"
sign "$P" expsig.sig "--default-sig-expire 1d" "--faked-system-time 20200101T000000"
sign "$SR" subrev.sig
sign "$PR" primrev.sig
sign "$SL" sublate.sig "" "--faked-system-time 20200601T000000"
for k in good other revoked embedded lapsed late subrev sublate; do
  gpg --homedir "$G" --batch --enarmor < "$WORK/$k.sig" 2>/dev/null | sed 's/ARMORED FILE/SIGNATURE/' > "$WORK/$k.armor"
done
gpgq --faked-system-time 20200102T000000 --quick-set-expire "$X" 1d
subfpr() { gpg --homedir "$G" --with-colons --with-subkey-fingerprints --list-keys "$1" 2>/dev/null | awk -F: '/^sub/{s=1} s && /^fpr/{print $10; exit}'; }
gpgq --faked-system-time 20200102T000000 --quick-set-expire "$SL" 1d "$(subfpr "$SL")"
printf 'key 1\nrevkey\ny\n0\n\ny\nsave\n' | gpgq --command-fd 0 --edit-key "$SR"
rev "$R" > "$WORK/revoked-cert.asc"
cp "$G/openpgp-revocs.d/$R.rev" "$WORK/revoked-raw.rev"
rev "$PR" > "$WORK/primrev-cert.asc"
# A certificate gpg reads but rejects: its signature no longer verifies
rev "$R" | gpg --dearmor 2>/dev/null > "$WORK/corrupt-cert.gpg"
printf 'X' | dd of="$WORK/corrupt-cert.gpg" bs=1 seek=$(($(wc -c < "$WORK/corrupt-cert.gpg") - 2)) conv=notrunc 2>/dev/null
rev "$E" | gpg --homedir "$G" --batch --quiet --import 2>/dev/null
gpgq --quick-gen-key 'Gone <g@x>' rsa2048 sign never; rev "$(fpr g@x)" > "$WORK/gone-cert.asc"

# bundle <dir> <key fingerprint> <revocation cert file or ""> writes gpg-keys.tar.gz
# BAD_KEY=1 ships a key file gpg cannot read, and BAD_TAR=1 a bundle tar cannot extract.
bundle() {
  mkdir -p "$1/keys/deb-revocations"
  # shellcheck disable=SC2086 # $2 may name several keys
  gpg --homedir "$G" --armor --export $2 > "$1/keys/bdot-public-gpg-key.asc" 2>/dev/null
  [ -n "$BAD_KEY" ] && echo garbage > "$1/keys/bdot-public-gpg-key.asc"
  [ -n "$3" ] && cp "$3" "$1/keys/deb-revocations/"
  [ -n "$RPM_REVOKE" ] && echo "$RPM_REVOKE" > "$1/keys/rpm-revocations.txt"
  (cd "$1/keys" && tar -czf "$1/gpg-keys.tar.gz" .)
  [ -n "$BAD_TAR" ] && echo garbage > "$1/gpg-keys.tar.gz"
  return 0
}
# report <name> <rc> <expect> <message a failure must contain> <output>. expect is ok (0),
# fail (1, the user may override), hard (3, never installs), or abort (error_exit's SIGPIPE).
report() {
  case "$3" in ok) want=0 ;; fail) want=1 ;; hard) want=3 ;; abort) want=141 ;; esac
  if [ "$2" -eq "$want" ] && { [ "$want" -eq 0 ] || printf '%s' "$5" | grep -qF -- "$4"; }; then pass=$((pass+1)); echo "PASS $1"; return; fi
  fail=$((fail+1)); echo "FAIL $1 (rc=$2, wanted $3${4:+: $4})"; printf '%s\n' "$5" | tail -3
}
# run_verify <dir> <package file> <package type> <locale> <extra shell> runs verify_package
run_verify() {
  (cd "$1" && LC_ALL=$4 sh -c ". '$WORK/lib.sh'; TMP_DIR='$1'; package_type=$3; package_out_file_path='$2'; gpg_tar_out_file_path='$1/gpg-keys.tar.gz'; $5 verify_package" 2>&1)
}

# deb_case <name> <signature> <key> <revocation cert> <locale> <expect> <message> [layout] [extra shell]
deb_case() {
  t="$WORK/deb-$1"; mkdir -p "$t/pkg"; bundle "$t" "$3" "$4"
  cp "$WORK/debian-binary" "$WORK/control.tar.gz" "$WORK/data.tar.gz" "$t/pkg/"; cp "$WORK/$2" "$t/pkg/_gpgorigin"
  [ "$1" = corrupted ] && printf 'x' | gzip > "$t/pkg/data.tar.gz"
  printf 'evil' | gzip > "$t/pkg/control.tar.xz"; printf 'evil' | gzip > "$t/pkg/data.tar.xz"
  printf 'evil' > "$t/pkg/x%sy"; printf 'evil' > "$t/pkg/debian-binary control.tar.gz"; printf 'evil' > "$t/pkg/$(printf 'x\033y')"
  # A + in the layout stands for a space inside a member name
  layout=${8:-debian-binary control.tar.gz data.tar.gz _gpgorigin}
  (cd "$t/pkg" && set -- && for m in $layout; do set -- "$@" "$(printf '%s' "$m" | tr '+' ' ')"; done && ar rc ../pkg.deb "$@")
  out=$(run_verify "$t" "$t/pkg.deb" deb "$5" "$9"); rc=$?
  report "deb-$1 ($5)" $rc "$6" "$7" "$out"
}
deb_case good        good.sig     "$S" ""                     C           ok
set -- "$WORK/deb-good"/bdot-gpg.*; if [ -e "$1" ]; then report deb-good-cleanup 1 ok "" "left $1"; else report deb-good-cleanup 0 ok "" ""; fi
# After a revocation the bundle still carries the revoked key and its certificate
deb_case good-after-revocation good.sig "$S $R" "$WORK/revoked-cert.asc" C ok
deb_case good-after-revocation-de good.sig "$S $R" "$WORK/revoked-cert.asc" de_DE.UTF-8 ok
BAD_KEY=1 deb_case unreadable-key good.sig "$S" ""            C           fail "Failed to import public key"
BAD_TAR=1 deb_case unreadable-bundle good.sig "$S" ""         C           fail "Failed to extract GPG key tar file"
deb_case good-de     good.sig     "$S" ""                     de_DE.UTF-8 ok
deb_case good-quiet  good.sig     "$S" ""                     C           ok   "" "" "non_interactive=true;"
deb_case corrupted   good.sig     "$S" ""                     C           hard "does not match its contents"
deb_case unknown     other.sig    "$S" ""                     C           fail "signature is invalid"
deb_case revoked     revoked.sig  "$R" "$WORK/revoked-cert.asc" C         hard "signing key is revoked"
deb_case revoked-de  revoked.sig  "$R" "$WORK/revoked-cert.asc" de_DE.UTF-8 hard "signing key is revoked"
deb_case embedded    embedded.sig "$E" ""                     C           hard "signing key is revoked"
deb_case subkey-revoked subrev.sig "$SR" ""                   C           hard "is revoked"
deb_case primary-revoked primrev.sig "$PR" "$WORK/primrev-cert.asc" C     hard "is revoked"
deb_case lapsed      lapsed.sig   "$L" ""                     C           ok
deb_case late        late.sig     "$X" ""                     C           hard "had expired when it signed"
deb_case late-de     late.sig     "$X" ""                     de_DE.UTF-8 hard "had expired when it signed"
deb_case subkey-late sublate.sig  "$SL" ""                    C           hard "had expired when it signed"
deb_case expsig      expsig.sig   "$P" ""                     C           hard "signature has expired"
deb_case extra-members good.sig   "$S" ""                     C           hard "unexpected contents" "debian-binary control.tar.xz data.tar.xz control.tar.gz data.tar.gz _gpgorigin"
deb_case reordered   good.sig     "$S" ""                     C           hard "unexpected contents" "debian-binary data.tar.gz control.tar.gz _gpgorigin"
# The signed members without a signature is an unsigned package, which the user may override
deb_case unsigned    good.sig     "$S" ""                     C           fail "Package is not signed" "debian-binary control.tar.gz data.tar.gz"
# A download that is not an ar archive at all, such as a proxy's HTML page, may be retried
t="$WORK/deb-html"; mkdir -p "$t"; bundle "$t" "$S" ""; echo '<html>login</html>' > "$t/pkg.deb"
out=$(run_verify "$t" "$t/pkg.deb" deb C ""); rc=$?
report deb-not-a-deb $rc fail "not a valid Debian package" "$out"
# Joined with spaces, these two members read like the expected layout
deb_case spaced-name good.sig     "$S" ""                     C           hard "unexpected contents" "debian-binary+control.tar.gz data.tar.gz _gpgorigin"
# Member names reach the error message, which must print them literally
deb_case format-name good.sig     "$S" ""                     C           hard "[x%sy]" "x%sy"
# Control characters in member names are printed as ?, so a name cannot rewrite the terminal
deb_case control-name good.sig    "$S" ""                     C           hard "[x?y]" "$(printf 'x\033y')"
# Every shipped revocation certificate must revoke a key in the bundle, or the bundle is broken
deb_case stale-revocation good.sig "$S" "$WORK/gone-cert.asc" C           hard "does not revoke a key in the BDOT key bundle"
deb_case raw-revocation revoked.sig "$R" "$WORK/revoked-raw.rev" C        hard "does not revoke a key in the BDOT key bundle"
deb_case corrupt-revocation revoked.sig "$R" "$WORK/corrupt-cert.gpg" C   hard "does not revoke a key in the BDOT key bundle"
# gnupg2-minimal exits 2 after a good import because it cannot start an agent
mkdir -p "$WORK/gpg-agentless"
printf '#!/bin/sh\n%s "$@"; rc=$?\ncase " $* " in *" --import "*) exit 2 ;; esac\nexit $rc\n' "$(command -v gpg)" > "$WORK/gpg-agentless/gpg"
chmod +x "$WORK/gpg-agentless/gpg"
deb_case import-exit-2 good.sig   "$S" ""                     C           ok   "" "" "PATH='$WORK/gpg-agentless':\$PATH;"
# gpg exiting 0 without a VALIDSIG line is not a verified signature
mkdir -p "$WORK/gpg-silent"
printf '#!/bin/sh\ncase " $* " in *" --verify "*) exit 0 ;; esac\nexec %s "$@"\n' "$(command -v gpg)" > "$WORK/gpg-silent/gpg"
chmod +x "$WORK/gpg-silent/gpg"
deb_case no-validsig   good.sig   "$S" ""                     C           fail "signature is invalid" "" "PATH='$WORK/gpg-silent':\$PATH;"
# gpg printing VALIDSIG but exiting nonzero, as with a second signature by an unknown key
mkdir -p "$WORK/gpg-exit-2"
printf '#!/bin/sh\ncase " $* " in *" --verify "*) %s "$@"; exit 2 ;; esac\nexec %s "$@"\n' "$(command -v gpg)" "$(command -v gpg)" > "$WORK/gpg-exit-2/gpg"
chmod +x "$WORK/gpg-exit-2/gpg"
deb_case verify-exit-2 good.sig   "$S" ""                     C           fail "signature is invalid" "" "PATH='$WORK/gpg-exit-2':\$PATH;"
# A failed bundle listing is a reported failure, not a silent return code
mkdir -p "$WORK/gpg-list-fails"
printf '#!/bin/sh\ncase " $* " in *" --fixed-list-mode "*) exit 2 ;; esac\nexec %s "$@"\n' "$(command -v gpg)" > "$WORK/gpg-list-fails/gpg"
chmod +x "$WORK/gpg-list-fails/gpg"
deb_case list-fails    good.sig   "$S" ""                     C           fail "Failed to list the bundle keys" "" "PATH='$WORK/gpg-list-fails':\$PATH;"
# A host gpg.conf with auto-key-import lets --verify add the signer's key from the signature
# itself; only keys the bundle carried may count
mkdir -p "$WORK/gpg-autoimport"
printf '#!/bin/sh\ncase " $* " in *" --verify "*) exec %s --auto-key-import "$@" ;; esac\nexec %s "$@"\n' "$(command -v gpg)" "$(command -v gpg)" > "$WORK/gpg-autoimport/gpg"
chmod +x "$WORK/gpg-autoimport/gpg"
gpgq --local-user o@x --include-key-block --detach-sign --output "$WORK/embedded-key.sig" "$WORK/payload"
deb_case auto-key-import embedded-key.sig "$S" ""             C           fail "not signed by a key in the BDOT key bundle" "" "PATH='$WORK/gpg-autoimport':\$PATH;"
# ar failing to extract the signature member
mkdir -p "$WORK/ar-no-sig"
printf '#!/bin/sh\nif [ "$1" = p ] && [ "$3" = _gpgorigin ]; then exit 1; fi\nexec %s "$@"\n' "$(command -v ar)" > "$WORK/ar-no-sig/ar"
chmod +x "$WORK/ar-no-sig/ar"
deb_case ar-no-sig     good.sig   "$S" ""                     C           fail "Failed to extract package signature" "" "PATH='$WORK/ar-no-sig':\$PATH;"
# ar failing partway through the signed members is an extraction failure, not tampering
mkdir -p "$WORK/ar-truncates"
printf '#!/bin/sh\nif [ "$1" = p ] && [ $# -gt 3 ]; then %s p "$2" debian-binary; exit 1; fi\nexec %s "$@"\n' "$(command -v ar)" "$(command -v ar)" > "$WORK/ar-truncates/ar"
chmod +x "$WORK/ar-truncates/ar"
deb_case ar-truncates  good.sig   "$S" ""                     C           fail "Failed to extract the signed package contents" "" "PATH='$WORK/ar-truncates':\$PATH;"

# gpg_key_status on canned listings like gpg 2.0's, whose subkey rows omit the primary's state
# status_case <name> <pub validity> <pub expiry> <signature time> <want>
status_case() {
  printf 'pub:%s:4096:1:6D1F39C113D127C8:1700000000:%s::-:::cC::::::23::0:\nfpr:::::::::7A0E3514903C1907DFED7DF16D1F39C113D127C8:\nsub:-:4096:1:3247193063B114B3:1700000000:1810399470:::::s::::::23:\nfpr:::::::::8A4C16208246418A8821E5F83247193063B114B3:\n' "$2" "$3" > "$WORK/canned-$1"
  out=$(sh -c ". '$WORK/lib.sh'; gpg_key_status '$WORK/canned-$1' 3247193063B114B3 $4" 2>&1)
  if [ "$out" = "$5" ]; then report "status-$1" 0 ok "" ""; else report "status-$1" 1 ok "" "got $out, wanted $5"; fi
}
status_case primary-revoked r ""         1750000000 revoked
status_case primary-expired e 1740000000 1750000000 expired
status_case before-primary-expiry - 1760000000 1750000000 ok
status_case at-subkey-expiry - ""        1810399470 expired
status_case before-subkey-expiry - ""    1810399469 ok
# A fingerprint whose last 16 hex match the subkey ID but whose rest differs is not that key
out=$(sh -c ". '$WORK/lib.sh'; gpg_key_status '$WORK/canned-primary-revoked' 0000000000000000000000003247193063B114B3 1750000000" 2>&1)
if [ "$out" = unknown ]; then report status-fingerprint-mismatch 0 ok "" ""; else report status-fingerprint-mismatch 1 ok "" "got $out, wanted unknown"; fi

# rpm: a stub answers verify_package_rpm's calls, since the real rpm database needs root. It
# returns real signature packets, and keeps installed keys as "name fingerprints" lines. Like
# rpm 4.11, it imports a multi-key file as one entry named after the file's last key. Given a
# testdata .rpm instead of an armor file, it passes queries to the real rpm.
# PRESEED lists "name fingerprint" entries installed before the run; "name @file" serves that
# file as the entry's key.
# rpm_case <name> <armor file or rpm> <key> <revocation cert> <checksig text> <checksig rc> <rpmdb key> <expect> <message> [rpm version]
rpm_case() {
  t="$WORK/rpm-$1"; mkdir -p "$t/bin"; bundle "$t" "$3" "$4"
  printf '%s\n' "$5" > "$t/checksig.txt"
  echo "gpg-pubkey-8483c65d-5ccc5b19 0000000000000000000000000000000000000000" > "$t/rpmdb.state"
  [ -n "$PRESEED" ] && printf '%s\n' "$PRESEED" >> "$t/rpmdb.state"
  # An rpmdb key outside the bundle stands in for an import that did not take
  f7=$(fpr "$7"); case " $3 " in *" $f7 "*) ;; *) echo "$f7" > "$t/rpmdb-override" ;; esac
  legacy=""; case "${10:-4.16.1.3}" in 4.[0-9].* | 4.1[01].*) legacy=1 ;; esac
  pkg="$t/pkg.rpm"; query="cat '$WORK/$2' 2>/dev/null"
  case "$2" in
    /*) pkg=$2; query="exec '$REAL_RPM' \"\$@\"" ;;
    *.rpm) pkg="$DATA/$2"; query="exec '$REAL_RPM' \"\$@\"" ;;
  esac
  [ -n "$NO_SIGPGP" ] && query="case \"\$*\" in *SIGPGP*) echo '(none)' ;; *) $query ;; esac"
  cat > "$t/bin/rpm" <<STUB
#!/bin/sh
case "\$*" in
  *"--qf "*) $query ;;
  *--checksig*) cat "$t/checksig.txt"; exit $6 ;;
  --version) echo "RPM version ${10:-4.16.1.3}" ;;
  "--import "*)
    if [ -f "$t/import-fails" ]; then
      # The first import fails; a later one (the restore) is recorded
      if [ -e "$t/import-failed" ]; then cat "\$2" > "$t/restored"; exit 0; fi
      touch "$t/import-failed"; echo "import refused"; exit 1
    fi
    echo imported >> "$t/rpm.log"
    if [ -f "$t/rpmdb-override" ]; then fprs=\$(cat "$t/rpmdb-override")
    else fprs=\$(gpg --with-colons --show-keys "\$2" 2>/dev/null | awk -F: '\$1 == "pub" { p = 1 } \$1 == "fpr" && p { print \$10; p = 0 }'); fi
    # rpm 4 skips a key whose entry is already installed; rpm 6 updates it
    case "${10:-4.16.1.3}" in 6.*) ;; *)
      new=""; for f in \$fprs; do n="gpg-pubkey-\$(printf '%s' "\$f" | cut -c33-40 | tr '[:upper:]' '[:lower:]')-5f000000"; awk -v n="\$n" '\$1 == n { f = 1 } END { exit !f }' "$t/rpmdb.state" || new="\$new \$f"; done
      fprs=\$new ;;
    esac
    [ -n "\$fprs" ] || exit 0
    if [ -n "$legacy" ]; then
      last=\$(printf '%s\n' \$fprs | tail -n 1)
      echo "gpg-pubkey-\$(printf '%s' "\$last" | cut -c33-40 | tr '[:upper:]' '[:lower:]')-5f000000" \$fprs >> "$t/rpmdb.state"
    else
      for f in \$fprs; do echo "gpg-pubkey-\$(printf '%s' "\$f" | cut -c33-40 | tr '[:upper:]' '[:lower:]')-5f000000 \$f"; done >> "$t/rpmdb.state"
    fi ;;
  # Like real rpm, -qa matches its pattern against the package name only
  "-qa gpg-pubkey*") while read -r n rest; do echo "\$n"; done < "$t/rpmdb.state" ;;
  "-qi "*)
    fprs=\$(awk -v n="\$2" '\$1 == n { \$1 = ""; print }' "$t/rpmdb.state")
    case "\$fprs" in *@*) cat \${fprs#*@} ;; "") ;; *) gpg --homedir "$G" --armor --export \$fprs 2>/dev/null ;; esac ;;
  "-q "*) awk -v n="\$2" '\$1 == n { f = 1 } END { exit !f }' "$t/rpmdb.state" ;;
  "-e "*)
    [ -f "$t/erase-fails" ] && grep -qx "\$2" "$t/erase-fails" && exit 1
    echo "erased \$2" >> "$t/rpm.log"
    awk -v n="\$2" '\$1 != n' "$t/rpmdb.state" > "$t/rpmdb.new"; mv "$t/rpmdb.new" "$t/rpmdb.state" ;;
esac
STUB
  chmod +x "$t/bin/rpm"
  [ -n "$ERASE_FAILS" ] && echo "$ERASE_FAILS" > "$t/erase-fails"
  [ -n "$IMPORT_FAILS" ] && touch "$t/import-fails"
  [ -n "$TOUCH" ] && touch "$t/$TOUCH"
  out=$(run_verify "$t" "$pkg" rpm de_DE.UTF-8 "PATH='$t/bin':\$PATH;"); rc=$?
  report "rpm-$1" $rc "$8" "$9" "$out"
}
# installed <case> <name>: the case's rpm keyring still has that entry
installed() { awk -v n="$2" '$1 == n { f = 1 } END { exit !f }' "$WORK/rpm-$1/rpmdb.state"; }
SUBID=$(gpg --homedir "$G" --with-colons --list-keys "$S" 2>/dev/null | awk -F: '/^sub/{print tolower($5); exit}')
SUB8=$(printf '%s' "$SUBID" | cut -c9-16)
# rpm 4 names the key by its 8 or 16 hex ID, and rpm 6 by the primary key's fingerprint
OK="Header V4 RSA/SHA256 Signature, key ID $SUBID: OK"
OK8="Header V4 RSA/SHA256 Signature, key ID $SUB8: OK"
OK6="Header OpenPGP V4 RSA/SHA256 signature, key fingerprint: $(printf '%s' "$S" | tr '[:upper:]' '[:lower:]'): OK"
OTHER_OK="Header V4 RSA/SHA256 Signature, key ID $(gpg --homedir "$G" --with-colons --list-keys o@x 2>/dev/null | awk -F: '/^pub/{print tolower($5); exit}'): OK"
NOKEY="Header V4 RSA/SHA256 Signature, key ID $SUBID: NOKEY"
rpm_case good       good.armor     "$S" ""                       "$OK" 0 "$S" ok
case "$out" in *rpm-revocations*) report rpm-good-quiet-stderr 1 ok "" "$out" ;; *) report rpm-good-quiet-stderr 0 ok "" "" ;; esac
rpm_case good-8     good.armor     "$S" ""                       "$OK8" 0 "$S" ok
rpm_case good-rpm6  good.armor     "$S" ""                       "$OK6" 0 "$S" ok
rpm_case unknown    other.armor    "$S" ""                       "$OK" 0 "$S" fail "not signed by a key in the BDOT key bundle"
rpm_case revoked    revoked.armor  "$R" "$WORK/revoked-cert.asc" "$OK" 0 "$R" hard "is revoked"
rpm_case embedded   embedded.armor "$E" ""                       "$OK" 0 "$E" hard "is revoked"
rpm_case subkey-revoked subrev.armor "$SR" ""                    "$OK" 0 "$SR" hard "is revoked"
rpm_case lapsed     lapsed.armor   "$L" ""                       "Header V4 RSA/SHA256 Signature, key ID $(printf %s "$L" | cut -c25-40 | tr "[:upper:]" "[:lower:]"): OK" 0 "$L" ok
rpm_case late       late.armor     "$X" ""                       "$OK" 0 "$X" hard "had expired when it signed"
rpm_case subkey-late sublate.armor "$SL" ""                      "$OK" 0 "$SL" hard "had expired when it signed"
rpm_case no-header  missing.armor  "$S" ""                       "$OK" 0 "$S" fail "Could not read the RPM signature"
rpm_case bad        good.armor     "$S" ""                       "$(printf '%s\nHeader SHA256 digest: BAD' "$OK")" 1 "$S" hard "RPM signature is BAD"
rpm_case nokey-ok   good.armor     "$S" ""                       "$(printf '%s\nMD5 digest: NOKEY' "$OK")" 1 "$S" ok
rpm_case nokey-only good.armor     "$S" ""                       "$NOKEY" 1 "$S" fail "could not be checked against the BDOT key"
rpm_case other-key-ok good.armor   "$S" ""                       "$OTHER_OK" 0 "$S" fail "could not be checked against the BDOT key"
rpm_case not-in-rpmdb good.armor   "$S" ""                       "$OK" 0 o@x fail "is not in the rpm keyring"
# Revoked rpm keys an earlier install left behind come out of the rpm keyring, and are never
# imported again. The bundle still carries each revoked key, as revocations.md asks.
R8=$(printf '%s' "$R" | cut -c33-40 | tr '[:upper:]' '[:lower:]')
L8=$(printf '%s' "$L" | cut -c33-40 | tr '[:upper:]' '[:lower:]')
S8=$(printf '%s' "$S" | cut -c33-40 | tr '[:upper:]' '[:lower:]')
RREV="$WORK/revoked-cert.asc"
PRESEED="gpg-pubkey-$R8-5f000000 $R" RPM_REVOKE=gpg-pubkey-$R8-5f000000 rpm_case removes-revoked good.armor "$S $R" "$RREV" "$OK" 0 "$S" ok
if installed removes-revoked gpg-pubkey-$R8-5f000000; then report rpm-removes-revoked-state 1 ok "" "the revoked key is still installed"; else report rpm-removes-revoked-state 0 ok "" ""; fi
# The listed release may differ from the host's; the fingerprint decides
PRESEED="gpg-pubkey-$R8-6abd1642 $R" RPM_REVOKE=gpg-pubkey-$R8-5f000000 rpm_case removes-by-fingerprint good.armor "$S $R" "$RREV" "$OK" 0 "$S" ok
if installed removes-by-fingerprint gpg-pubkey-$R8-6abd1642; then report removes-by-fingerprint-state 1 ok "" "the revoked key is still installed"; else report removes-by-fingerprint-state 0 ok "" ""; fi
# rpm 6 names entries by fingerprint; the 8-hex list form still finds them
RFPR=$(printf '%s' "$R" | tr '[:upper:]' '[:lower:]')
PRESEED="gpg-pubkey-$RFPR-6abd1642 $R" RPM_REVOKE=gpg-pubkey-$R8-5f000000 rpm_case removes-rpm6-name good.armor "$S $R" "$RREV" "$OK" 0 "$S" ok
if installed removes-rpm6-name gpg-pubkey-$RFPR-6abd1642; then report removes-rpm6-name-state 1 ok "" "the revoked key is still installed"; else report removes-rpm6-name-state 0 ok "" ""; fi
# A host key that only shares the revoked key's 8-hex ID stays
PRESEED="gpg-pubkey-$R8-11111111 $L" RPM_REVOKE=gpg-pubkey-$R8-5f000000 rpm_case keeps-id-collision good.armor "$S $R" "$RREV" "$OK" 0 "$S" ok
if installed keeps-id-collision gpg-pubkey-$R8-11111111; then report keeps-id-collision-state 0 ok "" ""; else report keeps-id-collision-state 1 ok "" "a colliding host key was removed"; fi
# rpm 4.11 would merge a multi-key file into one entry named after its last key, the revoked
# one, so removing it would take the signing key along; each key is imported on its own
RPM_REVOKE=gpg-pubkey-$R8-5f000000 rpm_case legacy-import-split good.armor "$S $R" "$RREV" "$OK" 0 "$S" ok "" 4.11.3
if installed legacy-import-split gpg-pubkey-$S8-5f000000 && ! installed legacy-import-split gpg-pubkey-$R8-5f000000; then report legacy-import-split-state 0 ok "" ""; else report legacy-import-split-state 1 ok "" "$(cat "$WORK/rpm-legacy-import-split/rpmdb.state")"; fi
# rpm -e runs as root, so only rpm key names may reach it, and never as a glob
RPM_REVOKE=sudo rpm_case remove-not-a-key good.armor "$S" "" "$OK" 0 "$S" hard "not an rpm key name"
# A glob would expand to file names in the working directory that look like key names
RPM_REVOKE='gpg-pubkey-*' TOUCH=gpg-pubkey-$R8-5f000000 rpm_case remove-glob good.armor "$S $R" "$RREV" "$OK" 0 "$S" hard "not an rpm key name"
# Only keys the bundle carries may be removed, so a bad bundle cannot strip the host's own keys
RPM_REVOKE=gpg-pubkey-8483c65d-5ccc5b19 rpm_case remove-host-key good.armor "$S" "" "$OK" 0 "$S" hard "not a key in the BDOT key bundle"
if installed remove-host-key gpg-pubkey-8483c65d-5ccc5b19; then report remove-host-key-state 0 ok "" ""; else report remove-host-key-state 1 ok "" "the host key was removed"; fi
# A revoked-key list saved with CRLF line endings still names the key
PRESEED="gpg-pubkey-$R8-5f000000 $R" RPM_REVOKE="$(printf 'gpg-pubkey-%s-5f000000\r' "$R8")" rpm_case removes-revoked-crlf good.armor "$S $R" "$RREV" "$OK" 0 "$S" ok
if installed removes-revoked-crlf gpg-pubkey-$R8-5f000000; then report removes-revoked-crlf-state 1 ok "" "the revoked key is still installed"; else report removes-revoked-crlf-state 0 ok "" ""; fi
# A signing key revoked only through the rpm list never installs, on any rpm version
SFPR=$(printf '%s' "$S" | tr '[:upper:]' '[:lower:]')
RPM_REVOKE="gpg-pubkey-$S8-5f000000" rpm_case list-revoked-signer good.armor "$S" "" "$OK" 0 "$S" hard "RPM signing key $(printf '%s' "$SUBID" | tr '[:lower:]' '[:upper:]') is revoked"
RPM_REVOKE="gpg-pubkey-$SFPR-5f000000" rpm_case list-revoked-signer-rpm6 good.armor "$S" "" "$OK" 0 "$S" hard "is revoked"
# A hard failure leaves no revoked key behind and imports nothing, and a malformed list fails
# before rpm is touched
PRESEED="gpg-pubkey-$R8-5f000000 $R" RPM_REVOKE="gpg-pubkey-$S8-5f000000 gpg-pubkey-$R8-5f000000" rpm_case list-revoked-signer-cleans good.armor "$S $R" "$RREV" "$OK" 0 "$S" hard "is revoked"
if installed list-revoked-signer-cleans gpg-pubkey-$R8-5f000000 || grep -q '^imported' "$WORK/rpm-list-revoked-signer-cleans/rpm.log" 2>/dev/null; then report list-revoked-signer-cleans-state 1 ok "" "$(cat "$WORK/rpm-list-revoked-signer-cleans/rpmdb.state")"; else report list-revoked-signer-cleans-state 0 ok "" ""; fi
RPM_REVOKE="bogus gpg-pubkey-$R8-5f000000" rpm_case malformed-list-before-import good.armor "$S $R" "$RREV" "$OK" 0 "$S" hard "not an rpm key name"
if grep -q '^imported' "$WORK/rpm-malformed-list-before-import/rpm.log" 2>/dev/null; then report malformed-list-before-import-state 1 ok "" "rpm --import ran"; else report malformed-list-before-import-state 0 ok "" ""; fi
RPM_REVOKE="gpg-pubkey-$S8-5f000000" rpm_case list-revoked-signer-legacy good.armor "$S" "" "$NOKEY" 1 "$S" hard "is revoked" 4.11.3
IMPORT_FAILS=1 rpm_case rpm-import-fails good.armor "$S" "" "$OK" 0 "$S" fail "Failed to import public key: import refused"
rpm_case good-after-revocation good.armor "$S $R" "$RREV" "$OK" 0 "$S" ok
# A key the bundle itself revokes is neither imported nor left installed, even when the rpm
# list omits it
PRESEED="gpg-pubkey-$R8-5f000000 $R" rpm_case bundle-revoked-not-imported good.armor "$S $R" "$RREV" "$OK" 0 "$S" ok
if installed bundle-revoked-not-imported gpg-pubkey-$R8-5f000000; then report bundle-revoked-not-imported-state 1 ok "" "the revoked key is installed"; else report bundle-revoked-not-imported-state 0 ok "" ""; fi
# An installed copy of the signing primary without the current signing subkey, left by an
# earlier install before a subkey rotation, is replaced
gpg --homedir "$G" --armor --export "$S!" > "$WORK/s-primary-only.asc" 2>/dev/null
PRESEED="gpg-pubkey-$S8-5f000000 @$WORK/s-primary-only.asc" rpm_case refreshes-stale-key good.armor "$S" "" "$OK" 0 "$S" ok
if awk -v n="gpg-pubkey-$S8-5f000000" '$1 == n && $2 !~ /^@/ { f = 1 } END { exit !f }' "$WORK/rpm-refreshes-stale-key/rpmdb.state"; then report refreshes-stale-key-state 0 ok "" ""; else report refreshes-stale-key-state 1 ok "" "$(cat "$WORK/rpm-refreshes-stale-key/rpmdb.state")"; fi
# A current copy is left alone
PRESEED="gpg-pubkey-$S8-5f000000 $S" rpm_case keeps-current-key good.armor "$S" "" "$OK" 0 "$S" ok
if grep -q '^erased' "$WORK/rpm-keeps-current-key/rpm.log" 2>/dev/null; then report keeps-current-key-state 1 ok "" "the current key was removed"; else report keeps-current-key-state 0 ok "" ""; fi
# The old copy comes back when the refreshed import fails
PRESEED="gpg-pubkey-$S8-5f000000 @$WORK/s-primary-only.asc" IMPORT_FAILS=1 rpm_case refresh-restores good.armor "$S" "" "$OK" 0 "$S" fail "Failed to import public key"
if [ -s "$WORK/rpm-refresh-restores/restored" ]; then report refresh-restores-state 0 ok "" ""; else report refresh-restores-state 1 ok "" "the old key was not re-imported"; fi
PRESEED="gpg-pubkey-$S8-5f000000 @$WORK/s-primary-only.asc" ERASE_FAILS=gpg-pubkey-$S8-5f000000 rpm_case refresh-erase-fails good.armor "$S" "" "$OK" 0 "$S" fail "Failed to remove outdated key"
# rpm 6 updates an installed key on import
PRESEED="gpg-pubkey-$S8-5f000000 @$WORK/s-primary-only.asc" rpm_case rpm6-import good.armor "$S" "" "$OK" 0 "$S" ok "" 6.0.2
# An rpm version that cannot be parsed counts as modern, so NOKEY stays a failure
rpm_case unparsable-version good.armor "$S" "" "$NOKEY" 1 "$S" fail "could not be checked against the BDOT key" "garbage"
# One key that will not come out does not keep the rest in
PRESEED="$(printf 'gpg-pubkey-%s-5f000000 %s\ngpg-pubkey-%s-5f000000 %s' "$R8" "$R" "$L8" "$L")" RPM_REVOKE="gpg-pubkey-$R8-5f000000 gpg-pubkey-$L8-5f000000" ERASE_FAILS=gpg-pubkey-$R8-5f000000 rpm_case remove-fails good.armor "$S $R $L" "$RREV" "$OK" 0 "$S" fail "Failed to remove revoked key gpg-pubkey-$R8-5f000000"
if installed remove-fails gpg-pubkey-$L8-5f000000; then report remove-fails-continues 1 ok "" "gpg-pubkey-$L8-5f000000 is still installed"; else report remove-fails-continues 0 ok "" ""; fi
# rpm before 4.12 cannot check subkey signatures and reports NOKEY, so gpg checks the header
# and payload signature over the same bytes rpm installs. The fixtures are signed like the
# BDOT packages: by a signing subkey, with both a header and a header and payload signature.
gpg --homedir "$G" --batch --quiet --import "$DATA/signed-test-key.asc" 2>/dev/null
FIX=$(fpr test@example.com)
FIXSUB=$(gpg --homedir "$G" --with-colons --list-keys "$FIX" 2>/dev/null | awk -F: '/^sub/{print $5; exit}')
FIXNOKEY="Header V4 RSA/SHA256 Signature, key ID $(printf '%s' "$FIXSUB" | cut -c9-16 | tr '[:upper:]' '[:lower:]'): NOKEY"
# grafted.rpm: signed-test.rpm's signed header in front of signed-test-other.rpm's payload
# shellcheck disable=SC2046 # split od's bytes into the positional parameters
header_end() { f=$1; set -- $(od -An -v -tu1 -j 96 -N 16 "$f"); s=$((16 + 16 * (($9 << 24) + (${10} << 16) + (${11} << 8) + ${12}) + ((${13} << 24) + (${14} << 16) + (${15} << 8) + ${16}))); s=$((96 + (s + 7) / 8 * 8)); set -- $(od -An -v -tu1 -j "$((s + 8))" -N 8 "$f"); echo $((s + 16 + 16 * (($1 << 24) + ($2 << 16) + ($3 << 8) + $4) + (($5 << 24) + ($6 << 16) + ($7 << 8) + $8))); }
{ head -c "$(header_end "$DATA/signed-test.rpm")" "$DATA/signed-test.rpm"; tail -c +"$(($(header_end "$DATA/signed-test-other.rpm") + 1))" "$DATA/signed-test-other.rpm"; } > "$WORK/grafted.rpm"
rpm_case legacy-verified  signed-test.rpm "$FIX" ""                   "$FIXNOKEY" 1 "$FIX" ok   "" 4.11.3
case "$out" in *"did not verify"*) report legacy-verified-by-gpg 1 ok "" "$out" ;; *) report legacy-verified-by-gpg 0 ok "" "" ;; esac
rpm_case legacy-verified-el6 signed-test.rpm "$FIX" ""                "$FIXNOKEY" 1 "$FIX" ok   "" 4.8.0
rpm_case legacy-grafted   "$WORK/grafted.rpm" "$FIX" ""               "$FIXNOKEY" 1 "$FIX" hard "does not match its contents" 4.11.3
NO_SIGPGP=1 rpm_case legacy-no-payload-sig signed-test.rpm "$FIX" ""  "$FIXNOKEY" 1 "$FIX" fail "has no header and payload signature" 4.11.3
# The main header starts where rpm_header_offset says, and a non-rpm has no offset
out=$(sh -c ". '$WORK/lib.sh'; o=\$(rpm_header_offset '$DATA/signed-test.rpm') && od -An -tx1 -j \"\$o\" -N 4 '$DATA/signed-test.rpm'" 2>&1); rc=$?
case "$out" in *"8e ad e8 01"*) report rpm-header-offset $rc ok "" "$out" ;; *) report rpm-header-offset 1 ok "" "$out" ;; esac
out=$(sh -c ". '$WORK/lib.sh'; rpm_header_offset '$WORK/payload' || { echo no-offset; exit 1; }" 2>&1); rc=$?
report rpm-header-offset-not-rpm $rc fail no-offset "$out"
# A signature header of 16 + 16 + 4 bytes pads to 40, so the main header starts at 96 + 40
{ head -c 96 /dev/zero; printf '\216\255\350\001\000\000\000\000\000\000\000\001\000\000\000\004'; } > "$WORK/padded.rpm"
out=$(sh -c ". '$WORK/lib.sh'; rpm_header_offset '$WORK/padded.rpm'" 2>&1); rc=$?
[ "$out" = 136 ] || rc=1
report rpm-header-offset-padded $rc ok "" "$out"
rpm_case legacy-other-nokey good.armor  "$S" ""                       "$(printf '%s' "$OTHER_OK" | sed 's/OK$/NOKEY/')" 1 "$S" fail "could not be checked against the BDOT key" 4.11.3
rpm_case legacy-bad       good.armor    "$S" ""                       "$(printf '%s\nMD5 digest: BAD (Expected 1 != 2)' "$NOKEY")" 1 "$S" hard "RPM signature is BAD" 4.11.3
rpm_case legacy-revoked   revoked.armor "$R" "$WORK/revoked-cert.asc" "$NOKEY" 1 "$R" hard "is revoked" 4.11.3
rpm_case modern-nokey     good.armor    "$S" ""                       "$NOKEY" 1 "$S" fail "could not be checked against the BDOT key" 4.12.0

# A key ID or time that is not a number is a malformed signature, not an unknown key
out=$(cd "$WORK" && sh -c ". '$WORK/lib.sh'; GPG_DIR='$WORK/rpm-good'; gpg_key_verdict NOTHEX 5 RPM" 2>&1); rc=$?
report malformed-signature $rc fail "has no usable key ID or signing time" "$out"

# verify_package's own setup failures
out=$(sh -c ". '$WORK/lib.sh'; TMP_DIR='$WORK/missing-dir'; package_type=deb; verify_package" 2>&1); rc=$?
report no-temp-dir $rc fail "Failed to create a temporary GPG directory" "$out"
out=$(cd "$WORK/deb-good" && sh -c ". '$WORK/lib.sh'; TMP_DIR='$WORK/deb-good'; package_type=snap; gpg_tar_out_file_path='$WORK/deb-good/gpg-keys.tar.gz'; verify_package" 2>&1); rc=$?
report unknown-package-type $rc fail "Unrecognized package type" "$out"

# The header signature parser against a real signed rpm (no root needed to read it)
t="$WORK/rpm-real"; mkdir -p "$t/keys/deb-revocations"; cp "$DATA/signed-test-key.asc" "$t/keys/bdot-public-gpg-key.asc"
(cd "$t/keys" && tar -czf "$t/gpg-keys.tar.gz" .)
out=$(cd "$t" && LC_ALL=de_DE.UTF-8 sh -c ". '$WORK/lib.sh'; TMP_DIR='$t'; package_out_file_path='$DATA/signed-test.rpm'; GPG_DIR=\$(mktemp -d '$t/gpg.XXXXXX'); tar -xzf '$t/gpg-keys.tar.gz' -C \"\$GPG_DIR\"; rpm_signing_key_check && echo \"KEYID=\$SIGNING_KEYID\"; gpg_cleanup" 2>&1); rc=$?
case "$out" in *KEYID=$FIXSUB*) report rpm-real-header $rc ok "" "$out" ;; *) report rpm-real-header 1 ok "" "$out" ;; esac
bundle "$t" "$S" ""
out=$(cd "$t" && sh -c ". '$WORK/lib.sh'; TMP_DIR='$t'; package_out_file_path='$DATA/signed-test.rpm'; GPG_DIR=\$(mktemp -d '$t/gpg.XXXXXX'); tar -xzf '$t/gpg-keys.tar.gz' -C \"\$GPG_DIR\"; rpm_signing_key_check; r=\$?; gpg_cleanup; exit \$r" 2>&1); rc=$?
report rpm-real-header-wrong-bundle $rc fail "not signed by a key in the BDOT key bundle" "$out"

# Verification must fail closed when gpg, tar, gzip, or ar is missing, unless --no-gpg-check was given.
# nopath <dir> <tools to omit>: a PATH directory mirroring the system bin dirs without those tools
nopath() {
  mkdir -p "$1"
  for d in /usr/local/bin /usr/bin /bin /usr/sbin /sbin; do
    for p in "$d"/*; do
      n=${p##*/}
      [ -e "$1/$n" ] || [ -L "$1/$n" ] && continue
      case " $2 " in *" $n "*) continue ;; esac
      ln -s "$p" "$1/$n"
    done
  done
}
# tool_case <name> <omit> <shell to run after sourcing> <expect> <message>
tool_case() {
  t="$WORK/tool-$1"; mkdir -p "$t"; bundle "$t" "$S" ""; nopath "$t/bin" "$2"
  cp "$WORK/deb-good/pkg.deb" "$t/pkg.deb"
  out=$(cd "$t" && PATH="$t/bin" sh -c ". '$WORK/lib.sh'; TMP_DIR='$t'; package_out_file_path='$t/pkg.deb'; gpg_tar_out_file_path='$t/gpg-keys.tar.gz'; $3" 2>&1); rc=$?
  report "$1" $rc "$4" "$5" "$out"
}
tool_case deb-verify-no-gpg    "gpg gpg2" "package_type=deb; verify_package"                           fail "requires: [gpg]"
tool_case deb-verify-no-ar     "ar"       "package_type=deb; verify_package"                           fail "requires: [ar]"
tool_case check-no-gpg         "gpg gpg2" "package_type=rpm; verification_check"                       abort "requires: [gpg]"
tool_case check-no-tar-gzip    "tar gzip" "package_type=rpm; verification_check"                       abort "requires: [tar, gzip]"
tool_case check-no-gpg-skipped "gpg gpg2" "package_type=rpm; skip_gpg_check=true; verification_check"  ok
tool_case check-deb-no-ar      "ar"       "package_type=deb; verification_check"                       abort "requires: [ar]"
tool_case check-rpm-no-ar      "ar"       "package_type=rpm; verification_check"                       ok
tool_case check-no-text-tools  "awk sed grep tr cut" "package_type=rpm; verification_check"             abort "requires: [awk, sed, grep, tr, cut]"
tool_case check-all-present    ""         "package_type=deb; verification_check"                       ok
# gnupg2-minimal has no gpgconf, and gpg 2.0's cannot stop daemons; cleanup must still succeed
tool_case verify-no-gpgconf    "gpgconf"  "package_type=deb; verify_package"                           ok
set -- "$WORK/tool-verify-no-gpgconf"/bdot-gpg.*; if [ -e "$1" ]; then report verify-no-gpgconf-cleanup 1 ok "" "left $1"; else report verify-no-gpgconf-cleanup 0 ok "" ""; fi

# verification_check needs package_type, so main must run it after setup_installation and
# before install_package, which downloads.
order=$(awk '/^main\(\)/{m=1} m && /^}/{m=0} m && /^  (setup_installation|verification_check|install_package)$/{printf "%s ", $1}' "$SCRIPT")
if [ "$order" = "setup_installation verification_check install_package " ]; then report main-order 0 ok "" ""; else report main-order 1 ok "" "main() order: $order"; fi

# Return code 3 never installs, even on yes; other failures can be overridden
# override_case <name> <verify_package return code> <expect> <message>
override_case() {
  out=$(cd "$WORK" && printf 'y\n' | sh -c ". '$WORK/lib.sh'; verify_package() { return $2; }; unpack_package() { echo UNPACKED; exit 0; }; non_interactive=false; offline_installation=true; install_package" 2>&1); rc=$?
  case "$3:$out" in ok:*UNPACKED*) rc=0 ;; ok:*) rc=1 ;; esac
  report "$1" $rc "$3" "$4" "$out"
}
override_case override-hard-failure 3 abort "Refusing to install"
override_case override-soft-failure 1 ok

# Quiet mode hides ordinary output, but a verification failure must still say why
# quiet_case <name> <verify_package return code>
quiet_case() {
  out=$(cd "$WORK" && sh -c ". '$WORK/lib.sh'; verify_package() { error 'the real reason'; return $2; }; unpack_package() { exit 0; }; non_interactive=true; offline_installation=true; install_package" 2>&1); rc=$?
  report "$1" $rc abort "the real reason" "$out"
}
quiet_case quiet-hard-failure 3
quiet_case quiet-soft-failure 1

# Interactive mode says why once, whether the user continues or declines
# once_case <name> <answer>
once_case() {
  out=$(cd "$WORK" && printf '%s\n' "$2" | sh -c ". '$WORK/lib.sh'; verify_package() { error 'the real reason'; return 1; }; unpack_package() { exit 0; }; non_interactive=false; offline_installation=true; install_package" 2>&1)
  n=$(printf '%s\n' "$out" | grep -c 'the real reason')
  if [ "$n" -eq 1 ]; then report "$1" 0 ok "" ""; else report "$1" 1 ok "" "printed $n times"; fi
}
once_case reason-once-continue y
once_case reason-once-decline n

# The bundle in signature/gpg, packed as release-prep-gpg does, must pass the checks that
# stop every install when a certificate or revoked-key entry is wrong
REPO=$(cd "$DATA/../../../.." && pwd)
t="$WORK/shipped-bundle"; mkdir -p "$t/keys"
cp -r "$REPO/signature/gpg/." "$t/keys/"; rm -f "$t/keys/revocations.md" "$t/keys/deb-revocations/.keep"
(cd "$t/keys" && tar -czf "$t/gpg-keys.tar.gz" .)
out=$(cd "$t" && sh -c ". '$WORK/lib.sh'; GPG_DIR=\$(mktemp -d '$t/gpg.XXXXXX'); tar -xzf '$t/gpg-keys.tar.gz' -C \"\$GPG_DIR\"; gpg_import_bundle && rpm_read_revoked_list; r=\$?; gpg_cleanup; exit \$r" 2>&1); rc=$?
report shipped-bundle $rc ok "" "$out"

echo "pass=$pass fail=$fail"; [ $fail -eq 0 ]
