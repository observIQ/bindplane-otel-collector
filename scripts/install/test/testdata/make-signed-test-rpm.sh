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

# Regenerates signed-test.rpm and signed-test-key.asc with a new throwaway key. Run it from
# this directory in a Fedora container, then update the key ID test_gpg_verify.sh expects:
#   podman run --rm -v "$PWD:/out:Z" -w /out fedora:latest sh make-signed-test-rpm.sh
set -e
dnf install -y -q rpm-build rpm-sign gnupg2 > /dev/null
GNUPGHOME=$(mktemp -d); export GNUPGHOME
gpg --batch --pinentry-mode loopback --passphrase '' --quick-gen-key 'BDOT install test <test@example.com>' rsa2048 sign never
gpg --armor --export > signed-test-key.asc

top=$(mktemp -d)
cat > "$top/signed-test.spec" <<'EOF'
Name: signed-test
Version: 1
Release: 1
Summary: Signed rpm for the install script tests
License: Apache-2.0
BuildArch: noarch
%description
Signed rpm for the install script tests.
%files
EOF
rpmbuild --define "_topdir $top" -bb "$top/signed-test.spec" > /dev/null
rpmsign --define '_gpg_name test@example.com' --addsign "$top"/RPMS/noarch/signed-test-1-1.noarch.rpm > /dev/null
cp "$top"/RPMS/noarch/signed-test-1-1.noarch.rpm signed-test.rpm
gpg --with-colons --list-keys | awk -F: '/^pub/ { print "key ID:", $5; exit }'
