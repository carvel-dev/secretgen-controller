#!/bin/bash

set -e -x -u

source ./hack/version-util.sh

mkdir -p tmp/
mkdir -p .imgpkg/

export version="$(get_sgctrl_ver)"
export version_without_v_prefix="$(get_sgctrl_ver_without_v)"

yq eval '.metadata.annotations."secretgen-controller.carvel.dev/version" = env(version)' -i "config/package-bundle/config/deployment.yml"

ytt -f config/package-bundle/config -f config/release -v dev.version="$version_without_v_prefix" | kbld --imgpkg-lock-output .imgpkg/images.yml -f- > ./tmp/release.yml

echo SUCCESS
