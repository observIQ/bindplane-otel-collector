# GPG Key Revocations

Any primary public keys that have been revoked should be placed within the `revocations` folder.

If a primary keypair has been lost or destroyed, its revocation certificate should be placed within the `revocations` folder.

Once one of the above two steps has been taken for the revoked keypair, the release action and install scripts will distribute the revocations to prevent users from installing new software signed using the revoked keypair.