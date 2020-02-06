#!/bin/bash

set -e

git clone bosh-utils bumped-bosh-utils

mkdir -p workspace/src/github.com/cloudfoundry/
ln -s $PWD/bumped-bosh-utils workspace/src/github.com/cloudfoundry/bosh-utils

export GOPATH=$PWD/workspace

cd workspace/src/github.com/cloudfoundry/bosh-utils

dep ensure -v -update

if [ "$(git status --porcelain)" != "" ]; then
  git status
  git add vendor Gopkg.lock
  git config user.name "CI Bot"
  git config user.email "cf-bosh-eng@pivotal.io"
  git commit -m "Update vendored dependencies"
fi

