# The archdoc pipe for Bitbucket Pipelines: the publish step, which checks nothing out, takes the
# record the generate step kept, and sends it with the token Bitbucket made for archdoc
# (BITBUCKET_STEP_OIDC_TOKEN, with "oidc: audiences" naming archdoc's). Built from the engine's
# repository root: docker build -f ci/bitbucket/Dockerfile .
FROM alpine:3.21
RUN apk add --no-cache curl jq
COPY ci/publish.sh /publish.sh
COPY ci/bitbucket/pipe.sh /pipe.sh
ENTRYPOINT ["/bin/sh", "/pipe.sh"]
