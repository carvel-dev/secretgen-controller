#!/usr/bin/env bash

fly -t director set-pipeline -p config-server -c ci/pipeline.yml --load-vars-from <(lpass show -G "config-server concourse secrets" --notes)
