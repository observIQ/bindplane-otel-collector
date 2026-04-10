Dismiss all open Dependabot security alerts that affect dependencies in directories listed in `migrated-modules.txt`. These modules have been migrated out and their vulnerabilities are not exploitable from this repo.

Run the following bash script to identify and dismiss the matching alerts:

```bash
set -euo pipefail

# Normalize migrated module prefixes: strip trailing slashes and blank lines
PREFIXES=$(sed 's|/$||' migrated-modules.txt | grep -v '^[[:space:]]*$')

# Fetch all open dependabot alerts (paginated)
# Note: use --jq inline to avoid jq parse errors from control chars in advisory text
echo "Fetching open Dependabot alerts..."
ALERT_LINES=$(gh api --paginate --jq '.[] | "\(.number)\t\(.dependency.manifest_path)\t\(.dependency.package.name)"' \
  'repos/observIQ/bindplane-otel-collector/dependabot/alerts?state=open&per_page=100')

# Find alert numbers where manifest_path matches a migrated module prefix
TO_DISMISS=$(echo "$ALERT_LINES" | \
  while IFS=$'\t' read -r num manifest pkg; do
    while IFS= read -r prefix; do
      if [[ "$manifest" == "$prefix/"* || "$manifest" == "$prefix/go.mod" ]]; then
        echo "$num $manifest $pkg"
        break
      fi
    done <<< "$PREFIXES"
  done | sort -u)

if [[ -z "$TO_DISMISS" ]]; then
  echo "No matching Dependabot alerts found."
  exit 0
fi

COUNT=$(echo "$TO_DISMISS" | wc -l | tr -d ' ')
echo "Found $COUNT alerts to dismiss:"
echo "$TO_DISMISS"
echo ""

echo "$TO_DISMISS" | while read -r num manifest pkg; do
  echo "Dismissing alert #$num ($pkg in $manifest)..."
  gh api --method PATCH "repos/observIQ/bindplane-otel-collector/dependabot/alerts/$num" \
    -f state=dismissed \
    -f dismissed_reason=not_used \
    -f dismissed_comment="Dismissing: this alert is for a module listed in migrated-modules.txt. Dependencies for migrated modules are managed independently in their own repositories." \
    > /dev/null
done

echo "Done."
```

After running, verify by checking the Dependabot alerts page — dismissed alerts will no longer appear in the open list. If any alerts were missed, check whether the `manifest_path` format differs from the expected `prefix/module/go.mod` pattern.
