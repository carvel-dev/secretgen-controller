#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

rm -rf tmp/crds

go run ./vendor/sigs.k8s.io/controller-tools/cmd/controller-gen/main.go \
  crd \
  output:dir=./tmp/crds \
  paths=./pkg/apis/...

ytt -f tmp/crds -f ./hack/crd-overlay.yml -f ./config/package-bundle/config/values-schema.yml > config/package-bundle/config/crds.yml

rm -rf tmp/crds
