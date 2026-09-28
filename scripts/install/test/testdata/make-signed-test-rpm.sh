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
# throwaway key laid out like the BDOT key, whose signing subkey makes the signatures. gpg 2.0
# cannot make a certify-only primary, so this one can sign too, but gpg signs with the subkey.
# It runs on Amazon Linux 2, whose rpm 4.11 rpmsign writes both the header signature and the
# header and payload signature, as the BDOT release packages carry. Run it from this directory:
#   podman run --rm -v "$PWD:/out:Z" -w /out amazonlinux:2 sh make-signed-test-rpm.sh
set -e
yum install -y -q rpm-build rpm-sign gnupg2 > /dev/null
GNUPGHOME=$(mktemp -d); export GNUPGHOME
gpg --batch --gen-key <<'EOF'
Key-Type: RSA
Key-Length: 2048
Subkey-Type: RSA
Subkey-Length: 2048
Subkey-Usage: sign
Name-Real: BDOT install test
Name-Email: test@example.com
Expire-Date: 0
%commit
EOF
gpg --armor --export > signed-test-key.asc

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
cp "$top/RPMS/noarch/signed-test-1-1.noarch.rpm" signed-test.rpm
cp "$top/RPMS/noarch/signed-test-2-1.noarch.rpm" signed-test-other.rpm
