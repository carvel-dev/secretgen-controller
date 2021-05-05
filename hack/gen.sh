#!/bin/bash

set -e

rm -rf pkg/client

./vendor/k8s.io/code-generator/generate-groups.sh \
	all github.com/vmware-tanzu/carvel-secretgen-controller/pkg/client github.com/vmware-tanzu/carvel-secretgen-controller/pkg/apis secretgen:v1alpha1 \
	--go-header-file ./hack/gen-boilerplate.txt
