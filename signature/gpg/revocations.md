# GPG Key Revocations

These files live in `signature/gpg/`, and the release action ships them in `gpg-keys.tar.gz`. The Linux install script (`scripts/install/install_unix.sh`) enforces them; a test builds this bundle and fails when a certificate or revoked-key entry would stop installs.

To revoke a primary keypair, keep its public key in `bdot-public-gpg-key.asc` and add its revocation certificate to the `deb-revocations` folder. Each file there holds one armored certificate that the key issued for itself; remove the `:` that gpg prefixes to the `-----BEGIN` line of a certificate from `openpgp-revocs.d`. The install script refuses to install when a certificate is unreadable, rejected by gpg, or names a key missing from the bundle.

To revoke only a signing subkey, revoke it in the keyring and ship the updated public key. The revocation travels inside the key, so it needs no file in `deb-revocations`.

The install script treats every revoked bundle key as revoked for rpm too: it never imports one into the rpm keyring, and removes any copy an earlier install left there. `rpm-revocations.txt` can also name revoked keys, one per line, as `gpg-pubkey-<8-hex key ID>-<release>` (rpm 4) or `gpg-pubkey-<fingerprint>-<date>` (rpm 6). Either form finds the key on every rpm version, since the script matches the installed key by fingerprint and ignores the release. A name that is not a key in the bundle makes every rpm install fail. rpm 6 warns that `rpm -e` of a key is deprecated in favor of `rpmkeys --delete`, but still removes it.
