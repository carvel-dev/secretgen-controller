#!/bin/bash

set -e -x -u

source ./hack/version-util.sh

mkdir -p tmp/

ytt -f config/release -f config-build -f config-deploy -f config-release -v secretgen_controller_version="$(get_sgctrl_ver)" | kbld --imgpkg-lock-output config/.imgpkg/images.yml -f- > ./tmp/release.yml

imgpkg push -b ghcr.io/vmware-tanzu/carvel-secretgen-controller-package-bundle:"$(get_sgctrl_ver)" -f config --lock-output ./bundle-image.yml

ytt -f packaging/package.yml -f packaging/values.yml -v version="$(get_sgctrl_ver)" -v image="$(yq eval '.bundle.image' bundle-image.yml)" > ./tmp/package.yml

cp packaging/metadata.yml ./tmp/metadata.yml

shasum -a 256 ./tmp/release*.yml

echo SUCCESS
