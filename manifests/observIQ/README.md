# observIQ manifests

Manifests contain the list of components for a specific distribution of the OpenTelemetry Collector.

## Editing a manifest

- **Add a component**: append a `gomod:` entry under the right section. Format: `<module path> <version>`.
- **Bump a dependency**: change the version in `gomod:`. Re-run the build. Commit the manifest change; ocb output dirs are gitignored.
- **Pin a transitive dep or redirect a fork**: add to `replaces:`. These apply to release builds too.
