# GPG Key Revocations

Any primary public keys that have been revoked should be placed within the `deb-revocations` folder.

If a primary keypair has been lost or destroyed, its revocation certificate should be placed within the `deb-revocations` folder.

If either of the above have occurred, add the RPM key ID for the primary public key to the install script. Explore the private GPG document for more info on how to find the RPM key ID.

Once one of the above three steps has been taken for the revoked keypair, the release action and install scripts will distribute the revocations to prevent users from installing new software signed using the revoked keypair.