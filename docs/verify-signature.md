# Verifying Artifact Signatures

Each release artifact has been signed with a ECDSA-P256 key. The public key is available in the repository at [here](../signature/bp_agent_key.pub) and can be used to verify the signature of the artifact.

In order to verify the signature of an artifact, you will need to install [cosign](https://github.com/sigstore/cosign). This can be done by getting the release artifacts from the [cosign releases page](https://github.com/sigstore/cosign/releases/tag/v1.13.1) or by using the following command if you have Go installed:

```bash
go install github.com/sigstore/cosign/cmd/cosign@v1.13.1
```

Once you have cosign installed, you can verify the signature of an artifact by running the following command:

```bash
cosign verify-blob --key ./signature/bp_agent_key.pub --signature <PATH_TO_SIG> <PATH_TO_ARTIFACT>
```

## Example

Heres an example of verifying the signature of an agent binary:

```bash
cosign verify-blob --key ./signature/bp_agent_key.pub --signature bindplane-otel-collector-v1.47.1-darwin-amd64.tar.gz.sig bindplane-otel-collector-v1.47.1-darwin-amd64.tar.gz
```
