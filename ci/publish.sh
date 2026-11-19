#!/bin/sh
# Sends a record to archdoc: the publish job of every CI's integration — the GitHub Action, the GitLab
# template, the Bitbucket pipe — runs this, after its CI gave it a token made for archdoc. It needs
# nothing but sh, gzip and curl (jq, when there, for reading the answer), and checks nothing out.
#
#   ARCHDOC_URL      where archdoc is (https://app.archdoc.dev)
#   ARCHDOC_PROJECT  the project's id, prj_…, from its setup page
#   ARCHDOC_TOKEN    the OIDC token the CI issued this run, made for archdoc
#   ARCHDOC_RECORD   the record the generate job kept (.archdoc/record.json)
#
# The record is exactly what is sent: the same file stays on the run, for anyone to read what archdoc
# received. Never retried: a token is taken once, so a second try would only be refused as used.
# Exits 0 when archdoc took it — or had it already — and 1, saying why and what to do, when it was
# refused.
set -eu

url="${ARCHDOC_URL:-https://app.archdoc.dev}"
project="${ARCHDOC_PROJECT:?ARCHDOC_PROJECT: the project id from its setup page, prj_…}"
token="${ARCHDOC_TOKEN:?ARCHDOC_TOKEN: the CI gave no token — the publish job needs one made for archdoc}"
record="${ARCHDOC_RECORD:-.archdoc/record.json}"

if [ ! -s "$record" ]; then
  echo "archdoc: no record at $record — the generate job keeps it as an artifact; this job must download it" >&2
  exit 1
fi

answer=$(mktemp)
trap 'rm -f "$answer"' EXIT
echo "archdoc: sending $record ($(wc -c <"$record" | tr -d ' ') bytes) to $url for $project"
code=$(gzip -c "$record" | curl -sS -X POST "$url/api/publish?project=$project" \
  -H "Authorization: Bearer $token" -H "Content-Type: application/gzip" \
  --data-binary @- -o "$answer" -w '%{http_code}') || {
  echo "archdoc: $url didn't answer" >&2
  exit 1
}

if command -v jq >/dev/null 2>&1; then
  outcome=$(jq -r '.outcome // ""' "$answer" 2>/dev/null || true)
  message=$(jq -r '.message // ""' "$answer" 2>/dev/null || true)
  todo=$(jq -r '.what_to_do // ""' "$answer" 2>/dev/null || true)
else
  outcome=$(sed -n 's/.*"outcome":"\([a-z]*\)".*/\1/p' "$answer")
  message=$(cat "$answer")
  todo=""
fi

case "$outcome" in
  accepted | same)
    echo "archdoc: $message"
    exit 0
    ;;
esac
echo "archdoc: refused (HTTP $code). ${message:-$(cat "$answer")}" >&2
[ -n "$todo" ] && echo "archdoc: what to do: $todo" >&2
exit 1
