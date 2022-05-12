// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"reflect"
	"strings"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/vmware-tanzu/carvel-secretgen-controller/pkg/apis/secretgen2/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func TestSecretTemplate_Single_Secret(t *testing.T) {
	env := BuildEnv(t)
	logger := Logger{}
	kapp := Kapp{t, env.Namespace, logger}
	kubectl := Kubectl{t, env.Namespace, logger}

	testYaml := `
apiVersion: v1
kind: Namespace
metadata:
  name: sg-template-test1
---
apiVersion: v1
kind: Secret
metadata:
  name: secret1
  namespace: sg-template-test1
type: Opaque
stringData:
  key1: val1
  key2: val2
---
apiVersion: v1
kind: Secret
metadata:
  name: secret2
  namespace: sg-template-test1
type: Opaque
stringData:
  key3: val3
  key4: val4
---
apiVersion: secretgen.carvel.dev/v1alpha1
kind: SecretTemplate
metadata:
  name: combined-secret
  namespace: sg-template-test1
spec:
  inputResources:
  - name: secret1 
    ref:
      apiVersion: v1
      kind: Secret
      name: secret1
  - name: secret2
    ref:
      apiVersion: v1
      kind: Secret
      name: secret2
  template:
    data: 
      key1: "$(.secret1.data.key1)"
      key2: "$(.secret1.data.key2)"
      key3: "$(.secret2.data.key3)"
      key4: "$(.secret2.data.key4)"
`

	name := "test-secrettemplate-successful"
	cleanUp := func() {
		kapp.RunWithOpts([]string{"delete", "-a", name}, RunOpts{AllowError: true})
	}

	cleanUp()
	defer cleanUp()

	logger.Section("Deploy", func() {
		kapp.RunWithOpts([]string{"deploy", "-f", "-", "-a", name},
			RunOpts{StdinReader: strings.NewReader(testYaml)})
	})

	logger.Section("Check secret was created", func() {
		out := waitForSecretInNs(t, kubectl, "sg-template-test1", "combined-secret")

		var secret corev1.Secret

		err := yaml.Unmarshal([]byte(out), &secret)
		if err != nil {
			t.Fatalf("Failed to unmarshal: %s", err)
		}

		expectedData := map[string][]byte{
			"key1": []byte("val1"),
			"key2": []byte("val2"),
			"key3": []byte("val3"),
			"key4": []byte("val4"),
		}
		if !reflect.DeepEqual(secret.Data, expectedData) {
			t.Fatalf("Expected secret data to match, but was: %#v vs %s", secret.Data, out)
		}
	})

	logger.Section("Check SecretTemplate .status.secret.name was updated", func() {
		args := []string{"get", "secrettemplate", "combined-secret", "-oyaml", "-n", "sg-template-test1"}

		out, err := kubectl.RunWithOpts(args, RunOpts{AllowError: true, NoNamespace: true})
		if err != nil {
			t.Fatalf("Failed to get secrettemplate: %s", err)
		}

		var template v1alpha1.SecretTemplate
		err = yaml.Unmarshal([]byte(out), &template)
		if err != nil {
			t.Fatalf("Failed to unmarshal: %s", err)
		}

		if !reflect.DeepEqual(template.Status.Secret.Name, "combined-secret") {
			t.Fatalf("Expected secrettemplate .status.secret.name to match, but was: %#v vs %s", "combined-secret", out)
		}
	})
}

func TestSecretTemplate_With_Service_Account(t *testing.T) {
	env := BuildEnv(t)
	logger := Logger{}
	kapp := Kapp{t, env.Namespace, logger}
	kubectl := Kubectl{t, env.Namespace, logger}

	test_yaml := `
apiVersion: v1
kind: Namespace
metadata:
  name: sg-template-test1
---
apiVersion: v1
kind: Secret
metadata:
  name: secret1
  namespace: sg-template-test1
type: Opaque
stringData:
  key1: val1
  key2: val2
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: configmap1
  namespace: sg-template-test1
data:
  key3: val3
  key4: val4
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: serviceaccount
  namespace: sg-template-test1
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: secret-template-reader
  namespace: sg-template-test1
rules:
- apiGroups:
  - ""
  resources:
  - configmaps
  - secrets
  verbs:
  - get
  - list
  - watch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: sa-rb
  namespace: sg-template-test1
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: secret-template-reader
subjects:
- kind: ServiceAccount
  name: serviceaccount
  namespace: sg-template-test1
---
apiVersion: secretgen.carvel.dev/v1alpha1
kind: SecretTemplate
metadata:
  name: combined-secret-sa
  namespace: sg-template-test1
spec:
  serviceAccountName: serviceaccount
  inputResources:
  - name: secret1 
    ref:
      apiVersion: v1
      kind: Secret
      name: secret1
  - name: configmap1
    ref:
      apiVersion: v1
      kind: ConfigMap
      name: configmap1
  template:
    data: 
      key1: "$(.secret1.data.key1)"
      key2: "$(.secret1.data.key2)"
    stringData:
      key3: "$(.configmap1.data.key3)"
      key4: "$(.configmap1.data.key4)"
`

	name := "test-secrettemplate-service-account-successful"
	cleanUp := func() {
		kapp.RunWithOpts([]string{"delete", "-a", name}, RunOpts{AllowError: true})
	}

	cleanUp()
	defer cleanUp()

	logger.Section("Deploy", func() {
		kapp.RunWithOpts([]string{"deploy", "-f", "-", "-a", name},
			RunOpts{StdinReader: strings.NewReader(test_yaml)})
	})

	logger.Section("Check secret was created", func() {
		out := waitForSecretInNs(t, kubectl, "sg-template-test1", "combined-secret-sa")

		var secret corev1.Secret

		err := yaml.Unmarshal([]byte(out), &secret)
		if err != nil {
			t.Fatalf("Failed to unmarshal: %s", err)
		}

		expectedData := map[string][]byte{
			"key1": []byte("val1"),
			"key2": []byte("val2"),
			"key3": []byte("val3"),
			"key4": []byte("val4"),
		}
		if !reflect.DeepEqual(secret.Data, expectedData) {
			t.Fatalf("Expected secret data to match, but was: %#v vs %s", secret.Data, out)
		}
	})
}
