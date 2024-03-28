// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestExportSuccessful(t *testing.T) {
	env := BuildEnv(t)
	logger := Logger{}
	kapp := Kapp{t, env.Namespace, logger}
	kubectl := Kubectl{t, env.Namespace, logger}

	yaml1 := `
apiVersion: v1
kind: Namespace
metadata:
  name: sg-test1
---
apiVersion: v1
kind: Namespace
metadata:
  name: sg-test2
---
apiVersion: v1
kind: Namespace
metadata:
  name: sg-test3
---
apiVersion: v1
kind: Namespace
metadata:
  name: sg-test4
  annotations:
    field.cattle.io/projectId: "cluster1:project1"
---
apiVersion: v1
kind: Namespace
metadata:
  name: sg-test5
  annotations:
    field.cattle.io/projectId: "cluster2:project3"
---
apiVersion: v1
kind: Namespace
metadata:
  name: sg-test6
  annotations:
    field.cattle.io/projectId: "whatever:whatever"
---
apiVersion: v1
kind: Namespace
metadata:
  name: sg-test7
---
apiVersion: v1
kind: Namespace
metadata:
  name: sg-test8
  annotations:
    one: "1"
    two: "2"
---
apiVersion: v1
kind: Namespace
metadata:
  name: sg-test8-unmatched
  annotations:
    one: "1"
---
apiVersion: v1
kind: Secret
metadata:
  name: secret
  namespace: sg-test1
type: Opaque
stringData:
  key1: val1
  key2: val2
  key3: val3
---
apiVersion: v1
kind: Secret
metadata:
  name: secret-test5
  namespace: sg-test1
type: Opaque
stringData:
  key1: val1
  key2: val2
  key3: val3
---
apiVersion: v1
kind: Secret
metadata:
  name: secret-test6
  namespace: sg-test1
type: Opaque
stringData:
  key1: val1
  key2: val2
  key3: val3
---
apiVersion: v1
kind: Secret
metadata:
  name: secret-test7
  namespace: sg-test1
type: Opaque
stringData:
  key1: val1
  key2: val2
  key3: val3
---
apiVersion: v1
kind: Secret
metadata:
  name: secret-test8
  namespace: sg-test1
type: Opaque
stringData:
  key1: val1
  key2: val2
  key3: val3
---
apiVersion: secretgen.carvel.dev/v1alpha1
kind: SecretExport
metadata:
  name: secret
  namespace: sg-test1
spec:
  toNamespaces:
  - sg-test2
  - sg-test3
  dangerousToNamespacesSelector:
  - key: "metadata.annotations['field\\.cattle\\.io/projectId']"
    operator: In
    values:
    - "cluster1:project1"
---
apiVersion: secretgen.carvel.dev/v1alpha1
kind: SecretExport
metadata:
  name: secret-test5
  namespace: sg-test1
spec:
  dangerousToNamespacesSelector:
  - key: "metadata.annotations['field\\.cattle\\.io/projectId']"
    operator: NotIn
    values:
    - "cluster1:project1"
---
apiVersion: secretgen.carvel.dev/v1alpha1
kind: SecretExport
metadata:
  name: secret-test6
  namespace: sg-test1
spec:
  dangerousToNamespacesSelector:
  - key: "metadata.annotations['field\\.cattle\\.io/projectId']"
    operator: Exists
---
apiVersion: secretgen.carvel.dev/v1alpha1
kind: SecretExport
metadata:
  name: secret-test7
  namespace: sg-test1
spec:
  dangerousToNamespacesSelector:
  - key: "metadata.annotations['field\\.cattle\\.io/projectId']"
    operator: DoesNotExist
---
apiVersion: secretgen.carvel.dev/v1alpha1
kind: SecretExport
metadata:
  name: secret-test8
  namespace: sg-test1
spec:
  dangerousToNamespacesSelector:
  - key: "metadata.annotations.one"
    operator: In
    values:
    - "1"
  - key: "metadata.annotations.two"
    operator: In
    values:
    - "2"
---
apiVersion: secretgen.carvel.dev/v1alpha1
kind: SecretImport
metadata:
  name: secret
  namespace: sg-test2
spec:
  fromNamespace: sg-test1
---
apiVersion: secretgen.carvel.dev/v1alpha1
kind: SecretImport
metadata:
  name: secret
  namespace: sg-test3
spec:
  fromNamespace: sg-test1
---
apiVersion: secretgen.carvel.dev/v1alpha1
kind: SecretImport
metadata:
  name: secret
  namespace: sg-test4
spec:
  fromNamespace: sg-test1
---
apiVersion: secretgen.carvel.dev/v1alpha1
kind: SecretImport
metadata:
  name: secret-test5
  namespace: sg-test5
spec:
  fromNamespace: sg-test1
---
apiVersion: secretgen.carvel.dev/v1alpha1
kind: SecretImport
metadata:
  name: secret-test6
  namespace: sg-test6
spec:
  fromNamespace: sg-test1
---
apiVersion: secretgen.carvel.dev/v1alpha1
kind: SecretImport
metadata:
  name: secret-test7
  namespace: sg-test7
spec:
  fromNamespace: sg-test1
---
apiVersion: secretgen.carvel.dev/v1alpha1
kind: SecretImport
metadata:
  name: secret-test8
  namespace: sg-test8
spec:
  fromNamespace: sg-test1
---
apiVersion: secretgen.carvel.dev/v1alpha1
kind: SecretImport
metadata:
  name: secret-test8
  namespace: sg-test8-unmatched
spec:
  fromNamespace: sg-test1
`

	yaml2 := `
---
apiVersion: v1
kind: Secret
metadata:
  name: secret
  namespace: sg-test1
type: Opaque
stringData:
  key1: val1.1 # update
               # key2 deleted
  key3: val3   # keep
  key4: val4   # new
---
apiVersion: v1
kind: Secret
metadata:
  name: secret-test5
  namespace: sg-test1
type: Opaque
stringData:
  key1: val1.1
  key3: val3
  key4: val4
---
apiVersion: v1
kind: Secret
metadata:
  name: secret-test6
  namespace: sg-test1
type: Opaque
stringData:
  key1: val1.1
  key3: val3
  key4: val4
---
apiVersion: v1
kind: Secret
metadata:
  name: secret-test7
  namespace: sg-test1
type: Opaque
stringData:
  key1: val1.1
  key3: val3
  key4: val4
---
apiVersion: v1
kind: Secret
metadata:
  name: secret-test8
  namespace: sg-test1
type: Opaque
stringData:
  key1: val1.1
  key3: val3
  key4: val4
`

	name := "test-export-successful"
	cleanUp := func() {
		kapp.RunWithOpts([]string{"delete", "-a", name}, RunOpts{AllowError: true})
	}

	cleanUp()
	defer cleanUp()

	getSecretName := func(ns string) string {
		switch ns {
		case "sg-test5":
			return "secret-test5"
		case "sg-test6":
			return "secret-test6"
		case "sg-test7":
			return "secret-test7"
		case "sg-test8":
			return "secret-test8"
		default:
			return "secret"
		}
	}

	logger.Section("Deploy", func() {
		kapp.RunWithOpts([]string{"deploy", "-f", "-", "-a", name},
			RunOpts{StdinReader: strings.NewReader(yaml1)})
	})

	logger.Section("Check imported secrets were created", func() {
		for _, ns := range []string{"sg-test2", "sg-test3", "sg-test4", "sg-test5", "sg-test6", "sg-test7", "sg-test8"} {
			out := waitForSecretInNs(t, kubectl, ns, getSecretName(ns))

			var secret corev1.Secret

			err := yaml.Unmarshal([]byte(out), &secret)
			if err != nil {
				t.Fatalf("Failed to unmarshal: %s", err)
			}

			if secret.Type != "Opaque" {
				t.Fatalf("Wrong type")
			}
			expectedData := map[string][]byte{
				"key1": []byte("val1"),
				"key2": []byte("val2"),
				"key3": []byte("val3"),
			}
			if !reflect.DeepEqual(secret.Data, expectedData) {
				t.Fatalf("Expected secret data to match, but was: %#v vs %s", secret.Data, out)
			}
		}
	})

	logger.Section("Check secrets should not be created", func() {
		for _, ns := range []string{"sg-test8-unmatched"} {
			notInNs := waitForSecretNotInNs(kubectl, ns, getSecretName(ns))
			if !notInNs {
				t.Fatalf("Secret should not be created in ns: %s", ns)
			}
		}
	})

	logger.Section("Update secret", func() {
		kapp.RunWithOpts([]string{"deploy", "-f", "-", "-a", name, "-p"},
			RunOpts{StdinReader: strings.NewReader(yaml2)})
	})

	logger.Section("Check imported secrets were updated", func() {
		// TODO proper waiting
		time.Sleep(5 * time.Second)

		for _, ns := range []string{"sg-test2", "sg-test3", "sg-test4", "sg-test5", "sg-test6", "sg-test7", "sg-test8"} {
			out := waitForSecretInNs(t, kubectl, ns, getSecretName(ns))

			var secret corev1.Secret

			err := yaml.Unmarshal([]byte(out), &secret)
			if err != nil {
				t.Fatalf("Failed to unmarshal: %s", err)
			}

			if secret.Type != "Opaque" {
				t.Fatalf("Wrong type")
			}
			expectedData := map[string][]byte{
				"key1": []byte("val1.1"),
				"key3": []byte("val3"),
				"key4": []byte("val4"),
			}
			if !reflect.DeepEqual(secret.Data, expectedData) {
				t.Fatalf("Expected secret data to match, but was: %#v vs %s", secret.Data, out)
			}
		}
	})

	logger.Section("Delete export to see exported secrets deleted", func() {
		for _, secretName := range []string{"secret", "secret-test5", "secret-test6", "secret-test7", "secret-test8"} {
			kubectl.RunWithOpts([]string{"delete", "secretexport.secretgen.carvel.dev", secretName, "-n", "sg-test1"},
				RunOpts{NoNamespace: true})
		}

		// TODO proper waiting
		time.Sleep(5 * time.Second)

		for _, ns := range []string{"sg-test2", "sg-test3", "sg-test4", "sg-test5", "sg-test6", "sg-test7", "sg-test8"} {
			_, err := kubectl.RunWithOpts([]string{"get", "secret", getSecretName(ns), "-n", ns},
				RunOpts{AllowError: true, NoNamespace: true})
			require.Error(t, err)

			if !strings.Contains(err.Error(), "(NotFound)") {
				t.Fatalf("Expected NotFound error but was: %s", err)
			}
		}
	})
}
