module github.com/vmware-tanzu/carvel-secretgen-controller

go 1.16

require (
	github.com/cloudfoundry/bosh-utils v0.0.0-20191216173634-505d7f919144 // indirect
	github.com/cloudfoundry/config-server v0.1.20
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v0.4.0
	github.com/stretchr/testify v1.7.0
	golang.org/x/tools v0.1.5 // indirect
	k8s.io/api v0.19.2
	k8s.io/apimachinery v0.19.2
	k8s.io/client-go v0.19.2
	k8s.io/code-generator v0.19.2
	sigs.k8s.io/controller-runtime v0.7.0
	sigs.k8s.io/controller-tools v0.4.1
)
