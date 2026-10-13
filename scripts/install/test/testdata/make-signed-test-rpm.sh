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

# Regenerates signed-test.rpm, signed-test-other.rpm, and signed-test-key.asc with a new
# throwaway key laid out like the BDOT key: a certify-only primary with a signing subkey,
# certified with SHA-512. gpg 2.1 or newer on this host makes the key, since gpg 2.0 cannot
# make a certify-only primary. Amazon Linux 2 then signs the packages, since its rpm 4.11
# rpmsign writes both the header signature and the header and payload signature, as the
# BDOT release packages carry. Run it from this directory on a host with podman:
#   sh make-signed-test-rpm.sh
set -e
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
GNUPGHOME="$work/gnupg"; export GNUPGHOME
mkdir -m 700 "$GNUPGHOME"
gpg --batch --pinentry-mode loopback --passphrase '' --cert-digest-algo SHA512 \
  --quick-gen-key 'BDOT install test <test@example.com>' rsa2048 cert never
fpr=$(gpg --with-colons --list-keys test@example.com | awk -F: '/^fpr/ { print $10; exit }')
gpg --batch --pinentry-mode loopback --passphrase '' --cert-digest-algo SHA512 \
  --quick-add-key "$fpr" rsa2048 sign never
gpg --armor --export > signed-test-key.asc
gpg --batch --pinentry-mode loopback --passphrase '' --armor --export-secret-keys > "$work/secret.asc"

cat > "$work/sign.sh" <<'SIGN'
set -e
yum install -y -q rpm-build rpm-sign gnupg2 > /dev/null
GNUPGHOME=$(mktemp -d); export GNUPGHOME
gpg --batch --import /work/secret.asc
# Two packages with different payloads, so the tests can graft one's payload onto the other
top=$(mktemp -d)
for v in 1 2; do
  cat > "$top/signed-test.spec" <<EOF
Name: signed-test
Version: $v
Release: 1
Summary: Signed rpm for the install script tests
License: Apache-2.0
BuildArch: noarch
%description
Signed rpm for the install script tests.
%install
mkdir -p %{buildroot}/opt
echo $v > %{buildroot}/opt/signed-test
%files
/opt/signed-test
EOF
  rpmbuild --define "_topdir $top" -bb "$top/signed-test.spec" > /dev/null
  # rpm 4.11 reads the empty passphrase from stdin when there is no terminal
  echo | rpmsign --define '_gpg_name test@example.com' --define '_gpg_digest_algo sha256' --addsign "$top/RPMS/noarch/signed-test-$v-1.noarch.rpm" > /dev/null
done
cp "$top/RPMS/noarch/signed-test-1-1.noarch.rpm" /out/signed-test.rpm
cp "$top/RPMS/noarch/signed-test-2-1.noarch.rpm" /out/signed-test-other.rpm
SIGN
podman run --rm -v "$PWD:/out:z" -v "$work:/work:ro,z" amazonlinux:2 sh /work/sign.sh
