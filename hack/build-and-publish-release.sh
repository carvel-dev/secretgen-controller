#!/bin/bash

set -e -x -u

source ./hack/version-util.sh

mkdir -p tmp/
mkdir -p config/package-bundle/.imgpkg/

export version="$(get_sgctrl_ver)"
export version_without_v_prefix="$(get_sgctrl_ver_without_v)"

yq eval '.metadata.annotations."secretgen-controller.carvel.dev/version" = env(version)' -i "config/package-bundle/config/deployment.yml"

ytt -f config/package-bundle/config -f config/release -v dev.version="$version_without_v_prefix" | kbld --imgpkg-lock-output config/package-bundle/.imgpkg/images.yml -f- > ./tmp/release.yml

imgpkg push -b ghcr.io/vmware-tanzu/carvel-secretgen-controller-package-bundle:"$version" -f config/package-bundle --lock-output ./tmp/bundle-image.yml

# generate openapi schema for package
ytt -f config/package-bundle/config --data-values-schema-inspect -o openapi-v3 > ./tmp/schema-openapi.yml

ytt -f config/package/package.yml -f config/package/values.yml --data-value-file openapi=./tmp/schema-openapi.yml -v version="$version_without_v_prefix" -v image="$(yq eval '.bundle.image' ./tmp/bundle-image.yml)" > ./tmp/package.yml

cp config/package/package-metadata.yml ./tmp/package-metadata.yml
rm ./tmp/bundle-image.yml
rm ./tmp/schema-openapi.yml

shasum -a 256 ./tmp/release.yml | tee ./tmp/checksums.txt
shasum -a 256 ./tmp/package.yml | tee -a ./tmp/checksums.txt
shasum -a 256 ./tmp/package-metadata.yml | tee -a ./tmp/checksums.txt

echo SUCCESS
