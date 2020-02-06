#!/bin/bash

set -e

./hack/build.sh && ytt -f config/ --data-value-yaml=push_images=true -v image_repo=docker.io/dkalinin/test | kbld -f- | kapp deploy -a sg -f- -c -y
