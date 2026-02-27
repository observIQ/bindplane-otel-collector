# Releasing
Releases are managed through GitHub releases. The steps to create a release are as follows:

1. Run `make version={VERSION} release` where `{VERSION}` is the version to release. This will create a tag and push it to GitHub.

2. The `release` CD workflow will trigger automatically on the tag push. Goreleaser will handle the following without user intervention:
  - Build the binaries for all platforms
  - Create GitHub release with automatic changelog content (based on commit message)
  - Attach binaries, archives, and packages to the GitHub release
  - Upload artifacts to GCS and Google Artifact Registry
  - Mark the release as a full release once it is finished.

3. Once the `release` workflow has completed successfully, manually trigger the `release-docker` workflow in GitHub Actions, providing the version tag (e.g. `v1.2.3`). This will:
  - Download the pre-built Linux binaries from GCS
  - Build and push Docker images for all supported architectures (amd64, arm64, ppc64le)
  - Push multi-arch manifests to DockerHub, GHCR, and Google Artifact Registry

> **Note:** The `release-docker` workflow is currently triggered manually to allow for confidence-building. In the future it is expected to trigger automatically upon successful completion of the `release` workflow.

4. While the images are built, verify the release.

5. Once the `release-docker` action completes, and verification is complete, alter the GitHub release to no longer be a pre-release.

6. Done!  The agent is released.

# Testing Release locally

In order to run the `make release-test` you need to setup the following in your environment:

1. Run `make install-tools`
2. Setup a github token per [goreleaser instructions](https://goreleaser.com/scm/github/#api-token)
3. Run `make release-test`

To test the Docker release workflow, trigger the `release-docker-test` workflow manually in GitHub Actions with an existing release version. This runs the full docker build pipeline with `--skip=publish`, so no images will be pushed to any registry.
