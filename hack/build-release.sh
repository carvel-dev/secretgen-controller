#!/bin/bash

set -e -x -u

source ./hack/version-util.sh

mkdir -p tmp/

ytt -f config/ -f config-release -v secretgen_controller_version="$(get_sgctrl_ver)" | kbld -f- > ./tmp/release.yml

echo SUCCESS
