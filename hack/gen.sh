#!/bin/bash

set -e

gen_groups_path=./vendor/k8s.io/code-generator/generate-groups.sh

chmod +x $gen_groups_path

rm -rf pkg/client

./vendor/k8s.io/code-generator/generate-groups.sh \
	all github.com/vmware-tanzu/carvel-secretgen-controller/pkg/client github.com/vmware-tanzu/carvel-secretgen-controller/pkg/apis secretgen:v1alpha1 \
	--go-header-file ./hack/gen-boilerplate.txt

./vendor/k8s.io/code-generator/generate-groups.sh \
	all github.com/vmware-tanzu/carvel-secretgen-controller/pkg/client2 github.com/vmware-tanzu/carvel-secretgen-controller/pkg/apis secretgen2:v1alpha1 \
	--go-header-file ./hack/gen-boilerplate.txt

chmod -x $gen_groups_path
