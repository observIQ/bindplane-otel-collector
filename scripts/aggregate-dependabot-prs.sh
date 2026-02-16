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

# Aggregates all open dependabot PRs into the current branch.
# For each PR, the go.mod changes from the PR diff are applied (skipping
# go.sum to avoid conflicts), `make tidy` is run to regenerate go.sum,
# and the result is committed with the PR title.
#
# Usage: ./scripts/aggregate-dependabot-prs.sh

set -e

# Ensure we're in the repo root (where the Makefile lives)
REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT"

# apply_pr_gomod_changes attempts to apply go.mod changes from a PR diff.
# First tries git apply; if that fails (e.g. context mismatch), falls back
# to parsing the new requirements from the diff and applying them via go mod edit.
apply_pr_gomod_changes() {
    local pr_number="$1"

    DIFF=$(gh pr diff "$pr_number")

    # Try git apply first (fast path)
    if echo "$DIFF" | git apply --include='**/go.mod' --include='go.mod' 2>/dev/null; then
        return 0
    fi

    echo "    git apply failed, falling back to go mod edit..."
    git checkout -- . 2>/dev/null

    # Parse the diff to find go.mod file paths and new direct requirements.
    # For each "+++ b/<path>/go.mod" we track the directory, then look for
    # added lines that look like Go module requirements (skip indirect ones).
    echo "$DIFF" | awk '
        /^\+\+\+ b\/.*go\.mod$/ {
            gomod = substr($0, 7)
            dir = gomod
            sub(/\/go\.mod$/, "", dir)
            if (dir == gomod) dir = "."
        }
        /^diff --git/ { gomod = ""; dir = "" }
        gomod && /^\+[[:space:]]+[a-z]/ && !/\/\/ indirect/ {
            line = $0
            sub(/^\+[[:space:]]+/, "", line)
            split(line, parts, " ")
            if (parts[1] != "" && parts[2] != "") {
                print dir "\t" parts[1] "\t" parts[2]
            }
        }
    ' | while IFS="$(printf '\t')" read -r dir module version; do
        echo "    go mod edit -require ${module}@${version} in ${dir}/"
        (cd "$dir" && go mod edit -require "${module}@${version}")
    done
}

# Ensure working tree is clean before starting
if [ -n "$(git status --porcelain)" ]; then
    echo "Error: working tree is not clean. Please commit or stash changes first."
    exit 1
fi

echo "Fetching open dependabot PRs..."
PRS=$(gh pr list --author "app/dependabot" --state open --json number,title,headRefName --jq '.[] | "\(.number)\t\(.title)\t\(.headRefName)"')

if [ -z "$PRS" ]; then
    echo "No open dependabot PRs found."
    exit 0
fi

PR_COUNT=$(echo "$PRS" | wc -l | tr -d ' ')
echo "Found $PR_COUNT open dependabot PR(s)."
echo ""

CURRENT_BRANCH=$(git branch --show-current)
SUCCEEDED=0
FAILED=0

echo "$PRS" | while IFS="$(printf '\t')" read -r number title branch; do
    echo "============================================"
    echo "PR #${number}: ${title}"
    echo "Branch: ${branch}"
    echo "============================================"

    # Apply go.mod changes from the PR diff (skipping go.sum to avoid conflicts).
    echo "  Applying go.mod changes from PR diff..."
    if ! apply_pr_gomod_changes "$number"; then
        echo "  Warning: could not apply go.mod changes for PR #${number}, skipping."
        git checkout -- . 2>/dev/null
        FAILED=$((FAILED + 1))
        continue
    fi

    # Run make tidy to reconcile go.mod/go.sum across all modules
    echo "  Running make tidy..."
    if ! make tidy; then
        echo "  Warning: make tidy failed for PR #${number}, resetting and skipping."
        git checkout -- .
        FAILED=$((FAILED + 1))
        continue
    fi

    # Stage all changes (including any tidy updates) and commit
    git add -A
    if git diff --cached --quiet; then
        echo "  No changes after merge, skipping commit."
    else
        git commit -m "$title"
        echo "  Committed: $title"
        SUCCEEDED=$((SUCCEEDED + 1))
    fi

    echo ""
done

echo "============================================"
echo "Done. Aggregated dependabot PRs into branch '$CURRENT_BRANCH'."
echo "============================================"
