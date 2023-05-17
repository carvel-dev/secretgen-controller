// Copyright 2021 VMware, Inc.
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
    field.cattle.io/projectId: "cluster1:project2"
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
apiVersion: secretgen.carvel.dev/v1alpha1
kind: SecretExport
metadata:
  name: secret
  namespace: sg-test1
spec:
  toNamespaces:
  - sg-test2
  - sg-test3
  toNamespaceAnnotation:
    field.cattle.io/projectId: "cluster1:project1"
  toNamespaceAnnotations:
    field.cattle.io/projectId: 
    - "cluster1:project2"
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
  name: secret
  namespace: sg-test5
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
`

	name := "test-export-successful"
	cleanUp := func() {
		kapp.RunWithOpts([]string{"delete", "-a", name}, RunOpts{AllowError: true})
	}

	cleanUp()
	defer cleanUp()

	logger.Section("Deploy", func() {
		kapp.RunWithOpts([]string{"deploy", "-f", "-", "-a", name},
			RunOpts{StdinReader: strings.NewReader(yaml1)})
	})

	logger.Section("Check imported secrets were created", func() {
		for _, ns := range []string{"sg-test2", "sg-test3", "sg-test4", "sg-test5"} {
			out := waitForSecretInNs(t, kubectl, ns, "secret")

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

	logger.Section("Update secret", func() {
		kapp.RunWithOpts([]string{"deploy", "-f", "-", "-a", name, "-p"},
			RunOpts{StdinReader: strings.NewReader(yaml2)})
	})

	logger.Section("Check imported secrets were updated", func() {
		// TODO proper waiting
		time.Sleep(5 * time.Second)

		for _, ns := range []string{"sg-test2", "sg-test3", "sg-test4", "sg-test5"} {
			out := waitForSecretInNs(t, kubectl, ns, "secret")

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
		kubectl.RunWithOpts([]string{"delete", "secretexport.secretgen.carvel.dev", "secret", "-n", "sg-test1"},
			RunOpts{NoNamespace: true})

		// TODO proper waiting
		time.Sleep(5 * time.Second)

		for _, ns := range []string{"sg-test2", "sg-test3", "sg-test4", "sg-test5"} {
			_, err := kubectl.RunWithOpts([]string{"get", "secret", "secret", "-n", ns},
				RunOpts{AllowError: true, NoNamespace: true})
			require.Error(t, err)

			if !strings.Contains(err.Error(), "(NotFound)") {
				t.Fatalf("Expected NotFound error but was: %s", err)
			}
		}
	})
}
