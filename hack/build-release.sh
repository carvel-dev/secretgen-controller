#!/bin/bash

set -e -x -u

source ./hack/version-util.sh

mkdir -p tmp/
mkdir -p config/package-bundle/.imgpkg/

export version="$(get_sgctrl_ver)"

yq eval '.metadata.annotations."secretgen-controller.carvel.dev/version" = env(version)' -i "config/package-bundle/contents/deployment.yml"

ytt -f config/package-bundle/contents -f config/release -v dev.version="$version" | kbld --imgpkg-lock-output config/package-bundle/.imgpkg/images.yml -f- > ./tmp/release.yml

imgpkg push -b ghcr.io/vmware-tanzu/carvel-secretgen-controller-package-bundle:"$version" -f config/package-bundle --lock-output ./bundle-image.yml

# generate openapi schema for package
ytt -f config/package-bundle/contents --data-values-schema-inspect -o openapi-v3 > ./schema-openapi.yml

ytt -f config/package/package.yml -f config/package/values.yml --data-value-file openapi=./schema-openapi.yml -v version="$version" -v image="$(yq eval '.bundle.image' bundle-image.yml)" > ./tmp/package.yml

cp config/package/metadata.yml ./tmp/metadata.yml
rm ./bundle-image.yml
rm ./schema-openapi.yml

shasum -a 256 ./tmp/release.yml
shasum -a 256 ./tmp/package.yml
shasum -a 256 ./tmp/metadata.yml

echo SUCCESS
