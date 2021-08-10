#!/bin/bash

set -e -x -u

go clean -testcache

echo "Note: if you have not deployed a recent version of SecretGen Controller your e2e tests may fail! run hack/deploy.sh or hack/test-all.sh to deploy"

# create ns if not exists because the `apply -f -` won't complain on a no-op if the ns already exists.
kubectl create ns $SECRETGEN_E2E_NAMESPACE --dry-run=client -o yaml | kubectl apply -f -
go test ./test/e2e/ -timeout 60m -test.v $@

echo E2E SUCCESS
