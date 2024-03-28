// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
---
apiVersion: secretgen.carvel.dev/v1alpha1
kind: SecretTemplate
metadata:
  name: combined-secret
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
type: Opaque
stringData:
  key1: val1
  key2: val2
---
apiVersion: v1
kind: Secret
metadata:
  name: secret2
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
		out := waitForSecretTemplate(t, kubectl, "combined-secret", sgv1alpha1.Condition{
			Type:    "ReconcileFailed",
			Status:  corev1.ConditionTrue,
			Reason:  "",
			Message: "cannot fetch input resource secret1: secrets \"secret1\" not found",
		})

		var secretTemplate sg2v1alpha1.SecretTemplate
		err := yaml.Unmarshal([]byte(out), &secretTemplate)
		require.NoError(t, err, "Failed to unmarshal secrettemplate")

		assert.Empty(t, secretTemplate.Status.Secret.Name, "Expected .status.secret.name reference to be empty")
	})

	logger.Section("Create Input Resources", func() {
		kapp.RunWithOpts([]string{"deploy", "-f", "-", "-a", name + "-inputs"},
			RunOpts{StdinReader: strings.NewReader(testInputResourcesYaml)})
	})

	logger.Section("Check secret was created and template has ReconcileSucceeded", func() {
		out := waitForSecretTemplate(t, kubectl, "combined-secret", sgv1alpha1.Condition{
			Type:   "ReconcileSucceeded",
			Status: corev1.ConditionTrue,
		})

		var secretTemplate sg2v1alpha1.SecretTemplate
		err := yaml.Unmarshal([]byte(out), &secretTemplate)
		require.NoError(t, err, "Failed to unmarshal secrettemplate")

		assert.Equal(t, "combined-secret", secretTemplate.Status.Secret.Name, "Expected .status.secret.name reference to match template name")
	})

	logger.Section("Delete Input Resources", func() {
		kapp.RunWithOpts([]string{"delete", "-a", name + "-inputs"}, RunOpts{AllowError: true})
	})

	logger.Section("Check template has ReconcileFailed but secret remains", func() {
		out := waitForSecretTemplate(t, kubectl, "combined-secret", sgv1alpha1.Condition{
			Type:   "ReconcileFailed",
			Status: corev1.ConditionTrue,
		})

		var secretTemplate sg2v1alpha1.SecretTemplate
		err := yaml.Unmarshal([]byte(out), &secretTemplate)
		require.NoError(t, err, "Failed to unmarshal secrettemplate")

		assert.NotEmpty(t, secretTemplate.Status.Secret.Name, "Expected .status.secret.name reference to not be empty")

		_, err = kubectl.RunWithOpts([]string{"get", "secret", "combined-secret", "-o", "yaml"}, RunOpts{AllowError: true})
		require.NoError(t, err, "Expected secret to still be present")
	})
}

func TestSecretTemplate_With_Service_Account(t *testing.T) {
	env := BuildEnv(t)
	logger := Logger{}
	kapp := Kapp{t, env.Namespace, logger}
	kubectl := Kubectl{t, env.Namespace, logger}

	testYaml := `
---
apiVersion: v1
kind: Secret
metadata:
  name: secret1
type: Opaque
stringData:
  key1: val1
  key2: val2
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: configmap1
data:
  key3: val3
  key4: val4
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: serviceaccount
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: secret-template-reader
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
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: secret-template-reader
subjects:
- kind: ServiceAccount
  name: serviceaccount
---
apiVersion: secretgen.carvel.dev/v1alpha1
kind: SecretTemplate
metadata:
  name: combined-secret-sa
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
		out := waitForSecret(t, kubectl, "combined-secret-sa")

		var secret corev1.Secret
		err := yaml.Unmarshal([]byte(out), &secret)
		require.NoError(t, err, "Failed to unmarshal secret")

		expectedData := map[string][]byte{
			"key1": []byte("val1"),
			"key2": []byte("val2"),
			"key3": []byte("val3"),
			"key4": []byte("val4"),
		}

		assert.Equal(t, expectedData, secret.Data, "Expected data to match")
	})

	logger.Section("Check status", func() {
		out := waitForSecretTemplate(t, kubectl, "combined-secret-sa", sgv1alpha1.Condition{
			Type:   "ReconcileSucceeded",
			Status: corev1.ConditionTrue,
		})

		var secretTemplate sg2v1alpha1.SecretTemplate
		err := yaml.Unmarshal([]byte(out), &secretTemplate)
		require.NoError(t, err, "Failed to unmarshal secrettemplate")

		assert.Equal(t, "combined-secret-sa", secretTemplate.Status.Secret.Name, "Expected .status.secret.name reference to match template name")
	})
}

func TestSecretTemplate_With_Service_Account_With_Insufficient_Permissions(t *testing.T) {
	env := BuildEnv(t)
	logger := Logger{}
	kapp := Kapp{t, env.Namespace, logger}
	kubectl := Kubectl{t, env.Namespace, logger}

	testYaml := `
---
apiVersion: v1
kind: Secret
metadata:
  name: secret1
type: Opaque
stringData:
  key1: val1
  key2: val2
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: configmap1
data:
  key3: val3
  key4: val4
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: insuff-serviceaccount
---
apiVersion: secretgen.carvel.dev/v1alpha1
kind: SecretTemplate
metadata:
  name: combined-secret-insuff-sa
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
		out := waitForSecretTemplate(t, kubectl, "combined-secret-insuff-sa", sgv1alpha1.Condition{
			Type:    "ReconcileFailed",
			Status:  corev1.ConditionTrue,
			Reason:  "",
			Message: "cannot fetch input resource secret1: secrets \"secret1\" not found",
		})

		var secretTemplate sg2v1alpha1.SecretTemplate
		err := yaml.Unmarshal([]byte(out), &secretTemplate)
		require.NoError(t, err, "Failed to unmarshal secrettemplate")

		assert.Empty(t, secretTemplate.Status.Secret.Name, "Expected .status.secret.name reference to be empty")
	})
}

func waitForSecretTemplate(t *testing.T, kubectl Kubectl, name string, condition sgv1alpha1.Condition) string {
	waitArgs := []string{"wait", fmt.Sprintf("--for=condition=%s=%s", condition.Type, condition.Status), "secrettemplate", name}
	getArgs := []string{"get", "secrettemplate", name, "-o", "yaml"}

	kubectl.RunWithOpts(waitArgs, RunOpts{AllowError: true})

	out, err := kubectl.RunWithOpts(getArgs, RunOpts{AllowError: true})
	if err == nil {
		return out
	}

	require.NoError(t, err, "Expected to find secrettemplate '%s' but did not: %s", name)
	panic("Unreachable")
}
