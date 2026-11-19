# archdoc for Bitbucket Pipelines

A pipe for the publish step: it checks nothing out, takes the record the generate step kept as an
artifact (`.archdoc/record.json`), and sends it to archdoc with the token Bitbucket made for this step
(`BITBUCKET_STEP_OIDC_TOKEN`). The step names archdoc's audience under `oidc: audiences`, so the token
is good for archdoc alone. archdoc's setup page writes the whole `bitbucket-pipelines.yml`.

The image is built from the repository's root: `docker build -f ci/bitbucket/Dockerfile .` It runs
`ci/publish.sh`, the script every integration's publish runs. It is not published yet.
