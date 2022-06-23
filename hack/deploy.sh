#!/bin/bash

set -e

./hack/build.sh && ytt -f config/ -f config-build/ -f config-deploy/ | kbld -f- | kapp deploy -a sg -f- -c -y
