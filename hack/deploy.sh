#!/bin/bash

set -e

./hack/build.sh && ytt -f config/package-bundle/contents -f config/dev | kbld -f- | kapp deploy -a sg -f- -c -y
