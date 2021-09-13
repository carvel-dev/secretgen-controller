#!/bin/bash

set -e -x -u

./hack/build.sh
./hack/deploy.sh
./hack/test.sh
./hack/test-e2e.sh
./hack/test-examples.sh

echo ALL SUCCESS
