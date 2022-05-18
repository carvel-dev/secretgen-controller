// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/ghodss/yaml"
	sgv1alpha1 "github.com/vmware-tanzu/carvel-secretgen-controller/pkg/apis/secretgen/v1alpha1"
	sg2v1alpha1 "github.com/vmware-tanzu/carvel-secretgen-controller/pkg/apis/secretgen2/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func TestSecretTemplate_Full_Lifecycle(t *testing.T) {
	env := BuildEnv(t)
	logger := Logger{}
	kapp := Kapp{t, env.Namespace, logger}
	kubectl := Kubectl{t, env.Namespace, logger}

	testSecretTemplateYaml := `
apiVersion: v1
kind: Namespace
metadata:
  name: sg-template-test1
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
    type: secret-type
    data:
      key1: "$(.secret1.data.key1)"
      key2: "$(.secret1.data.key2)"
      key3: "$(.secret2.data.key3)"
      key4: "$(.secret2.data.key4)"
`

	testInputResourcesYaml := `
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
`

	name := "test-secrettemplate-full-lifecycle"
	cleanUp := func() {
		kapp.RunWithOpts([]string{"delete", "-a", name + "-template"}, RunOpts{AllowError: true})
		kapp.RunWithOpts([]string{"delete", "-a", name + "-inputs"}, RunOpts{AllowError: true})
	}

	cleanUp()
	defer cleanUp()

	logger.Section("Create Template", func() {
		kapp.RunWithOpts([]string{"deploy", "-f", "-", "-a", name + "-template"},
			RunOpts{StdinReader: strings.NewReader(testSecretTemplateYaml)})
	})

	logger.Section("Check secret wasn't created and template has ReconcileFailed", func() {
		out := waitForSecretTemplate(t, kubectl, "sg-template-test1", "combined-secret", sgv1alpha1.Condition{
			Type:    "ReconcileFailed",
			Status:  corev1.ConditionTrue,
			Reason:  "",
			Message: "cannot fetch input resource secret1: secrets \"secret1\" not found",
		})

		var secretTemplate sg2v1alpha1.SecretTemplate
		err := yaml.Unmarshal([]byte(out), &secretTemplate)
		if err != nil {
			t.Fatalf("Failed to unmarshal: %s", err)
		}

		if !reflect.DeepEqual(secretTemplate.Status.Secret.Name, "") {
			t.Fatalf("Expected .status.secret.name reference to match, but was: %#v vs %s", secretTemplate.Status.Secret.Name, "")
		}
	})

	logger.Section("Create Input Resources", func() {
		kapp.RunWithOpts([]string{"deploy", "-f", "-", "-a", name + "-inputs"},
			RunOpts{StdinReader: strings.NewReader(testInputResourcesYaml)})
	})

	logger.Section("Check secret was created and template has ReconcileSucceeded", func() {
		out := waitForSecretTemplate(t, kubectl, "sg-template-test1", "combined-secret", sgv1alpha1.Condition{
			Type:   "ReconcileSucceeded",
			Status: corev1.ConditionTrue,
		})

		var secretTemplate sg2v1alpha1.SecretTemplate
		err := yaml.Unmarshal([]byte(out), &secretTemplate)
		if err != nil {
			t.Fatalf("Failed to unmarshal: %s", err)
		}

		if !reflect.DeepEqual(secretTemplate.Status.Secret.Name, "combined-secret") {
			t.Fatalf("Expected .status.secret.name reference to match, but was: %#v vs %s", secretTemplate.Status.Secret.Name, "combined-secret")
		}
	})

	logger.Section("Delete Input Resources", func() {
		kapp.RunWithOpts([]string{"delete", "-a", name + "-inputs"}, RunOpts{AllowError: true})
	})

	logger.Section("Check secret was deleted and template has ReconcileFailed", func() {
		out := waitForSecretTemplate(t, kubectl, "sg-template-test1", "combined-secret", sgv1alpha1.Condition{
			Type:   "ReconcileFailed",
			Status: corev1.ConditionTrue,
		})

		var secretTemplate sg2v1alpha1.SecretTemplate
		err := yaml.Unmarshal([]byte(out), &secretTemplate)
		if err != nil {
			t.Fatalf("Failed to unmarshal: %s", err)
		}

		if !reflect.DeepEqual(secretTemplate.Status.Secret.Name, "") {
			t.Fatalf("Expected .status.secret.name reference to match, but was: %#v vs %s", secretTemplate.Status.Secret.Name, "")
		}

		_, lastErr := kubectl.RunWithOpts([]string{"get", "secret", "combined-secret", "-o", "yaml"}, RunOpts{AllowError: true, NoNamespace: true})
		if lastErr == nil {
			t.Fatalf("Expected secret to not be present")
		}
	})

	logger.Section("Recreate Input Resources", func() {
		kapp.RunWithOpts([]string{"deploy", "-f", "-", "-a", name + "-inputs"},
			RunOpts{StdinReader: strings.NewReader(testInputResourcesYaml)})
	})

	logger.Section("Delete SecretTemplate", func() {
		kapp.RunWithOpts([]string{"delete", "-a", name + "-template"}, RunOpts{AllowError: true})
	})

	logger.Section("Check secret was deleted", func() {
		_, lastErr := kubectl.RunWithOpts([]string{"get", "secret", "combined-secret", "-o", "yaml"}, RunOpts{AllowError: true, NoNamespace: true})
		if lastErr == nil {
			t.Fatalf("Expected secret to not be present")
		}
	})
}

func TestSecretTemplate_With_Service_Account(t *testing.T) {
	env := BuildEnv(t)
	logger := Logger{}
	kapp := Kapp{t, env.Namespace, logger}
	kubectl := Kubectl{t, env.Namespace, logger}

	testYaml := `
apiVersion: v1
kind: Namespace
metadata:
  name: sg-template-test2
---
apiVersion: v1
kind: Secret
metadata:
  name: secret1
  namespace: sg-template-test2
type: Opaque
stringData:
  key1: val1
  key2: val2
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: configmap1
  namespace: sg-template-test2
data:
  key3: val3
  key4: val4
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: serviceaccount
  namespace: sg-template-test2
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: secret-template-reader
  namespace: sg-template-test2
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
  namespace: sg-template-test2
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: secret-template-reader
subjects:
- kind: ServiceAccount
  name: serviceaccount
  namespace: sg-template-test2
---
apiVersion: secretgen.carvel.dev/v1alpha1
kind: SecretTemplate
metadata:
  name: combined-secret-sa
  namespace: sg-template-test2
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
			RunOpts{StdinReader: strings.NewReader(testYaml)})
	})

	logger.Section("Check secret was created", func() {
		out := waitForSecretInNs(t, kubectl, "sg-template-test2", "combined-secret-sa")

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
			t.Fatalf("Expected secret data to match, but was: %#v vs %s", secret.Data, expectedData)
		}
	})

	logger.Section("Check status", func() {
		out := waitForSecretTemplate(t, kubectl, "sg-template-test2", "combined-secret-sa", sgv1alpha1.Condition{
			Type:   "ReconcileSucceeded",
			Status: corev1.ConditionTrue,
		})

		var secretTemplate sg2v1alpha1.SecretTemplate
		err := yaml.Unmarshal([]byte(out), &secretTemplate)
		if err != nil {
			t.Fatalf("Failed to unmarshal: %s", err)
		}

		if !reflect.DeepEqual(secretTemplate.Status.Secret.Name, "combined-secret-sa") {
			t.Fatalf("Expected .status.secret.name reference to match, but was: %#v vs %s", secretTemplate.Status.Secret.Name, "combined-secret-sa")
		}
	})
}

func TestSecretTemplate_With_Service_Account_With_Insufficient_Permissions(t *testing.T) {
	env := BuildEnv(t)
	logger := Logger{}
	kapp := Kapp{t, env.Namespace, logger}
	kubectl := Kubectl{t, env.Namespace, logger}

	testYaml := `
apiVersion: v1
kind: Namespace
metadata:
  name: sg-template-test3
---
apiVersion: v1
kind: Secret
metadata:
  name: secret1
  namespace: sg-template-test3
type: Opaque
stringData:
  key1: val1
  key2: val2
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: configmap1
  namespace: sg-template-test3
data:
  key3: val3
  key4: val4
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: insuff-serviceaccount
  namespace: sg-template-test3
---
apiVersion: secretgen.carvel.dev/v1alpha1
kind: SecretTemplate
metadata:
  name: combined-secret-insuff-sa
  namespace: sg-template-test3
spec:
  serviceAccountName: insuff-serviceaccount
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

	name := "test-secrettemplate-service-account-failure"
	cleanUp := func() {
		kapp.RunWithOpts([]string{"delete", "-a", name}, RunOpts{AllowError: true})
	}

	cleanUp()
	defer cleanUp()

	logger.Section("Deploy", func() {
		kapp.RunWithOpts([]string{"deploy", "-f", "-", "-a", name},
			RunOpts{StdinReader: strings.NewReader(testYaml)})
	})

	logger.Section("Check status is failing", func() {
		out := waitForSecretTemplate(t, kubectl, "sg-template-test3", "combined-secret-insuff-sa", sgv1alpha1.Condition{
			Type:    "ReconcileFailed",
			Status:  corev1.ConditionTrue,
			Reason:  "",
			Message: "cannot fetch input resource secret1: secrets \"secret1\" not found",
		})

		var secretTemplate sg2v1alpha1.SecretTemplate
		err := yaml.Unmarshal([]byte(out), &secretTemplate)
		if err != nil {
			t.Fatalf("Failed to unmarshal: %s", err)
		}

		if !reflect.DeepEqual(secretTemplate.Status.Secret.Name, "") {
			t.Fatalf("Expected .status.secret.name reference to match, but was: %#v vs %s", secretTemplate.Status.Secret.Name, "")
		}
	})
}

func waitForSecretTemplate(t *testing.T, kubectl Kubectl, nsName, name string, condition sgv1alpha1.Condition) string {
	waitArgs := []string{"wait", fmt.Sprintf("--for=condition=%s=%s", condition.Type, condition.Status), "secrettemplate", name}
	getArgs := []string{"get", "secrettemplate", name, "-o", "yaml"}

	noNs := false

	if len(nsName) > 0 {
		waitArgs = append(waitArgs, []string{"-n", nsName}...)
		getArgs = append(getArgs, []string{"-n", nsName}...)
		noNs = true
	}

	kubectl.RunWithOpts(waitArgs, RunOpts{AllowError: true, NoNamespace: noNs})

	out, err := kubectl.RunWithOpts(getArgs, RunOpts{AllowError: true, NoNamespace: noNs})
	if err == nil {
		return out
	}

	t.Fatalf("Expected to find secrettemplate '%s' but did not: %s", name, err)
	panic("Unreachable")
}
