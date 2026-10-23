#!/usr/bin/env bash
# The archdoc GitHub Action's work (action.yml), each part a workflow opts into: what a change does
# to the architecture, whether the committed documentation is current, and the site. Runs anywhere
# with git, jq and a built archdoc — set ARCHDOC and the INPUT_ variables — so it is tried locally
# as it runs in CI.
set -euo pipefail

archdoc="${ARCHDOC:?the archdoc binary}"
root="${INPUT_PATH:-.}"
summary="${GITHUB_STEP_SUMMARY:-/dev/stdout}"
output="${GITHUB_OUTPUT:-/dev/null}"
marker="<!-- archdoc: what this pull request changes in the architecture -->"

# 1. What the change does to the architecture: base against head, read from git at each commit.
if [ -n "${INPUT_BASE:-}" ]; then
  head="${INPUT_HEAD:-HEAD}"
  for c in "$INPUT_BASE" "$head"; do
    # A shallow checkout has the commit it checked out, not the pull request's base.
    git -C "$root" cat-file -e "$c^{commit}" 2>/dev/null || git -C "$root" fetch --quiet --no-tags --depth=1 origin "$c"
  done
  from=$(git -C "$root" rev-parse --short "$INPUT_BASE")
  to=$(git -C "$root" rev-parse --short "$head")
  said=$("$archdoc" diff "$root" "$from" "$to")
  echo "$said"

  # What diff says, between its first line (from → to) and its last (the count); nothing when it
  # says there is no change.
  changes=$(printf '%s\n' "$said" | sed -n 's/^\([0-9][0-9]*\) change(s) to the structure.*/\1/p')
  changes="${changes:-0}"
  lines=$(printf '%s\n' "$said" | sed '1,2d' | sed '/change(s) to the structure/d;/^no change to the architecture$/d' | sed '/^$/d')
  echo "changes=$changes" >>"$output"

  if [ -z "$lines" ]; then
    body="$marker
### Architecture

This changes nothing in the architecture — no element, relationship or way in added, removed or renamed.

<sub>Compared by <code>archdoc diff</code>, $from → $to, read from the code.</sub>"
  else
    body="$marker
### Architecture

What this changes in the architecture, $from → $to:

\`\`\`diff
$lines
\`\`\`

<sub>Read from the code by <code>archdoc diff</code>: <code>archdoc show</code> follows any of these, every line cited.</sub>"
  fi
  printf '%s\n' "$body" >>"$summary"

  # One comment per pull request, edited as it changes: the first once there is something to say,
  # then kept true — including when a later push undoes the change.
  if [ "${INPUT_COMMENT:-false}" = "true" ]; then
    pr=$(jq -r '.pull_request.number // empty' "${GITHUB_EVENT_PATH:-/dev/null}" 2>/dev/null || true)
    if [ -z "$pr" ]; then
      echo "::warning::comment is on, but this run is not for a pull request — nothing posted"
    else
      repo="${GITHUB_REPOSITORY:?}"
      id=$(gh api "repos/$repo/issues/$pr/comments" --paginate --jq ".[] | select(.body | startswith(\"$marker\")) | .id" | head -n 1)
      if [ -n "$id" ]; then
        gh api --silent -X PATCH "repos/$repo/issues/comments/$id" -f body="$body"
      elif [ -n "$lines" ]; then
        gh api --silent -X POST "repos/$repo/issues/$pr/comments" -f body="$body"
      fi
    fi
  fi
fi

# 2. Is the committed documentation current? Before the site, which generates over it.
if [ "${INPUT_CHECK:-false}" = "true" ]; then
  status=$("$archdoc" status "$root" --json)
  if [ "$(jq -r .documented <<<"$status")" != "true" ]; then
    echo "::error::$root is not documented: run 'archdoc generate $root' and commit .archdoc/model.json and docs/architecture/"
    exit 1
  fi
  moved=$(jq -r .changes <<<"$status")
  if [ "$moved" -lt 0 ]; then
    echo "::error::archdoc could not read the code to compare it: $(jq -r .code_error <<<"$status")"
    exit 1
  fi
  if [ "$moved" -gt 0 ]; then
    "$archdoc" status "$root"
    echo "::error::The architecture documentation is out of date: the code has $moved change(s) to the structure it does not have. Run 'archdoc generate $root' and commit what it writes."
    exit 1
  fi
  echo "The architecture documentation matches the code."
fi

# 3. The site: generated here, so a repository need not commit its documentation to publish it.
if [ "${INPUT_SITE:-false}" = "true" ]; then
  "$archdoc" generate "$root"
  "$archdoc" export --site "$root"
  echo "site=$root/.archdoc/site" >>"$output"
fi
