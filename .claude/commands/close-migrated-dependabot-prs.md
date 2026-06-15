Close all open dependabot PRs that target directories listed in `migrated-modules.txt`. These modules have been migrated out and their dependencies are managed independently, so dependabot PRs for them are noise.

Run the following bash script to identify and close the matching PRs:

```bash
set -euo pipefail

# Normalize migrated module prefixes: strip trailing slashes and blank lines
PREFIXES=$(sed 's|/$||' migrated-modules.txt | grep -v '^[[:space:]]*$')

# Fetch all open dependabot PRs (up to 200) including branch info
PRS=$(gh pr list --author "app/dependabot" --limit 200 --json number,title,headRefName)

# Find PR numbers where the directory suffix matches a migrated module prefix
TO_CLOSE=$(echo "$PRS" | jq -r '.[] | "\(.number)\t\(.headRefName)\t\(.title)"' | \
  while IFS=$'\t' read -r num branch title; do
    # Dependabot titles end with "in /path/to/module"
    dir=$(echo "$title" | sed 's/.* in \///')
    [[ -z "$dir" ]] && continue
    while IFS= read -r prefix; do
      if [[ "$dir" == "$prefix" || "$dir" == "$prefix/"* ]]; then
        echo "$num $branch"
        break
      fi
    done <<< "$PREFIXES"
  done | sort -u)

if [[ -z "$TO_CLOSE" ]]; then
  echo "No matching dependabot PRs found."
  exit 0
fi

COUNT=$(echo "$TO_CLOSE" | wc -l | tr -d ' ')
echo "Found $COUNT PRs to close:"
echo "$TO_CLOSE"
echo ""

echo "$TO_CLOSE" | while read -r num branch; do
  echo "Closing PR #$num (branch: $branch)..."
  gh pr close "$num" --comment "Closing: this PR targets a directory listed in \`migrated-modules.txt\`. Dependencies for migrated modules are managed independently and should not be updated here." --delete-branch
done

echo "Done."
```

After running, confirm the count of closed PRs matches expectations. If any PRs were missed, check whether the title format differs from the expected `... in /path/to/module` pattern.
