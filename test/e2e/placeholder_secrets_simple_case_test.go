// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"strings"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

// TestPlaceholderSuccessful : TODO - we have a more complicated test that should make this one redundant; move things from here into unit tests and/or remove this file
func TestPlaceholderSuccessful(t *testing.T) {
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
kind: Secret
metadata:
  name: secret
  namespace: sg-test1
type: kubernetes.io/dockerconfigjson
stringData:
  .dockerconfigjson: |
    {
      "auths": {
        "www.sg-test1-server.com": {
          "username": "sg-test1-secret-user",
          "password": "sg-test1-secret-password",
          "auth": "sgtest1-notbase64"
        }
      }
    }
---
apiVersion: secretgen.k14s.io/v1alpha1
kind: SecretExport
metadata:
  name: secret
  namespace: sg-test1
spec:
  toNamespaces:
  - sg-test2
  - sg-test3
---
apiVersion: v1
kind: Secret
metadata:
  annotations:
    secretgen.carvel.dev/image-pull-secret: ""
  name: secret
  namespace: sg-test2
type: kubernetes.io/dockerconfigjson
data:
  .dockerconfigjson: "e30K"
---
apiVersion: v1
kind: Secret
metadata:
  annotations:
    secretgen.carvel.dev/image-pull-secret: ""
  name: secret
  namespace: sg-test3
type: kubernetes.io/dockerconfigjson
data:
  .dockerconfigjson: "e30K"
`

	yaml2 := `
---
apiVersion: v1
kind: Secret
metadata:
  name: secret
  namespace: sg-test1
type: kubernetes.io/dockerconfigjson
stringData:
  .dockerconfigjson: |
    {
      "auths": {
        "www.sg-test1-server.com": {
          "username": "sg-test1-secret-user2",
          "password": "sg-test1-secret-password2",
          "auth": "sgtest1-notbase64-2"
        }
      }
    }
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
		for _, ns := range []string{"sg-test2", "sg-test3"} {
			out := waitUntilSecretInNsPopulated(t, kubectl, ns, "secret", func(secret *corev1.Secret) bool {
				return len(secret.Data[".dockerconfigjson"]) > 20
			})

			var secret corev1.Secret

			err := yaml.Unmarshal([]byte(out), &secret)
			if err != nil {
				t.Fatalf("Failed to unmarshal: %s", err)
			}

			expected := `{"auths":{"www.sg-test1-server.com":{"username":"sg-test1-secret-user","password":"sg-test1-secret-password","auth":"sgtest1-notbase64"}}}`
			require.Equal(t, "kubernetes.io/dockerconfigjson", string(secret.Type))
			assert.Equal(t, expected, string(secret.Data[".dockerconfigjson"]))
		}
	})

	logger.Section("Update secret", func() {
		kapp.RunWithOpts([]string{"deploy", "-f", "-", "-a", name, "-p"},
			RunOpts{StdinReader: strings.NewReader(yaml2)})
	})

	logger.Section("Check imported secrets were updated", func() {
		for _, ns := range []string{"sg-test2", "sg-test3"} {
			out := waitUntilSecretInNsPopulated(t, kubectl, ns, "secret", func(secret *corev1.Secret) bool {
				return strings.Contains(string(secret.Data[".dockerconfigjson"]), "user2")
			})

			var secret corev1.Secret

			err := yaml.Unmarshal([]byte(out), &secret)
			require.NoError(t, err)

			expected := `{"auths":{"www.sg-test1-server.com":{"username":"sg-test1-secret-user2","password":"sg-test1-secret-password2","auth":"sgtest1-notbase64-2"}}}`
			require.Equal(t, "kubernetes.io/dockerconfigjson", string(secret.Type))
			assert.Equal(t, expected, string(secret.Data[".dockerconfigjson"]))

		}
	})

	logger.Section("Delete export to see exported secrets deleted", func() {
		kubectl.RunWithOpts([]string{"delete", "secretexport", "secret", "-n", "sg-test1"},
			RunOpts{NoNamespace: true})
		for _, ns := range []string{"sg-test2", "sg-test3"} {
			out := waitUntilSecretInNsPopulated(t, kubectl, ns, "secret", func(secret *corev1.Secret) bool {
				return len(secret.Data[".dockerconfigjson"]) < 20
			})

			var secret corev1.Secret

			err := yaml.Unmarshal([]byte(out), &secret)
			if err != nil {
				t.Fatalf("Failed to unmarshal: %s", err)
			}

			expected := `{"auths":{}}`
			require.Equal(t, "kubernetes.io/dockerconfigjson", string(secret.Type))
			assert.Equal(t, expected, string(secret.Data[".dockerconfigjson"]))
		}
	})
}
