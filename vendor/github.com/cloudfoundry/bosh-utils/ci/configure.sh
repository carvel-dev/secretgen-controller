#!/bin/bash

set -e

fly -t production set-pipeline -p bosh-utils -c ci/pipeline.yml \
    --load-vars-from <(lpass show -G "bosh-utils concourse secrets" --notes) \
    --load-vars-from <(lpass show --note "bosh:docker-images concourse secrets")
