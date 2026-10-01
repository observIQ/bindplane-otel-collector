# GPG Key Revocations

These files live in `signature/gpg/`, and the release action ships them in `gpg-keys.tar.gz`. The Linux install script (`scripts/install/install_unix.sh`) enforces them; a test builds this bundle and fails when a certificate would stop installs.

To revoke a primary keypair, keep its public key in `bdot-public-gpg-key.asc` and add its revocation certificate to the `deb-revocations` folder. Each file there holds one armored certificate that the key issued for itself; remove the `:` that gpg prefixes to the `-----BEGIN` line of a certificate from `openpgp-revocs.d`. The install script refuses to install when a certificate is unreadable, rejected by gpg, or names a key missing from the bundle.

To revoke only a signing subkey, revoke it in the keyring and ship the updated public key. The revocation travels inside the key, so it needs no file in `deb-revocations`.

To remove the revoked key from rpm keyrings, add its rpm key name to `RPM_GPG_KEYS_TO_REMOVE` in the install script. Find the name by importing the public key with `rpm --import`, then running `rpm -q gpg-pubkey --info`.
