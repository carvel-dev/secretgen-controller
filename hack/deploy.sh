#!/bin/bash

set -e

./hack/build.sh && ytt -f config/ | kbld -f- | kapp deploy -a sg -f- -c -y
