#!/bin/sh
# The pipe's work: Bitbucket's variables as ci/publish.sh takes them.
set -eu
export ARCHDOC_URL="${ARCHDOC_URL:-https://app.archdoc.dev}"
export ARCHDOC_PROJECT="${ARCHDOC_PROJECT:?ARCHDOC_PROJECT: the project id from its setup page, prj_…}"
export ARCHDOC_RECORD="${ARCHDOC_RECORD:-.archdoc/record.json}"
if [ -z "${BITBUCKET_STEP_OIDC_TOKEN:-}" ]; then
  echo "archdoc: Bitbucket gave this step no token — give the step 'oidc' with archdoc's audience, as the setup page writes" >&2
  exit 1
fi
export ARCHDOC_TOKEN="$BITBUCKET_STEP_OIDC_TOKEN"
exec sh /publish.sh
