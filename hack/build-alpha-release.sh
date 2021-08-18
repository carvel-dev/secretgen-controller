#!/bin/bash

set -e -x -u

mkdir -p tmp/

ytt -f config/ -f config-alpha-release | kbld -f- > ./tmp/release.yml

echo alpha SUCCESS
