#!/bin/bash

set -e

# Force the legacy (non-Buildx) builder. Modern Docker CLI routes plain
# `docker build` through Buildx by default whenever the buildx plugin is
# present, regardless of kbld's build/buildx config choice, and Buildx's
# builder bootstrap fails to inspect its builder image under minikube's
# docker-env when the node's container runtime is containerd.
export DOCKER_BUILDKIT=0

./hack/build.sh && ytt -f config/package-bundle/config -f config/dev | kbld -f- | kapp deploy -a sg -f- -c -y
