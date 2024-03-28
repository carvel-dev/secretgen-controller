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
	corev1 "k8s.io/api/core/v1"
)

func TestPlaceholderMultiSuccessful(t *testing.T) {
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
---
apiVersion: v1
kind: Namespace
metadata:
  name: sg-test5
---
apiVersion: v1
kind: Secret
metadata:
  name: secret-kappa
  namespace: sg-test1
type: kubernetes.io/dockerconfigjson
stringData:
  .dockerconfigjson: |
    {
      "auths": {
        "www.sg-test1-server-a.com": {
          "username": "sg-test1-secret-user-a",
          "password": "sg-test1-secret-password-a",
          "auth": "sgtest1-notbase64-a"
        },
        "www.sg-test1-server-b.com": {
          "username": "sg-test1-secret-user-b",
          "password": "sg-test1-secret-password-b",
          "auth": "sgtest1-notbase64-b"
        }
      }
    }
---
apiVersion: v1
kind: Secret
metadata:
  name: secret-mu
  namespace: sg-test1
type: kubernetes.io/dockerconfigjson
stringData:
  .dockerconfigjson: |
    {
      "auths": {
        "www.sg-test1-server-c.com": {
          "username": "sg-test1-secret-user-c",
          "password": "sg-test1-secret-password-c",
          "auth": "sgtest1-notbase64-c"
        },
        "www.sg-test1-server-d.com": {
          "username": "sg-test1-secret-user-d",
          "password": "sg-test1-secret-password-d",
          "auth": "sgtest1-notbase64-d"
        }
      }
    }
---
apiVersion: secretgen.carvel.dev/v1alpha1
kind: SecretExport
metadata:
  name: secret-kappa
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
---
apiVersion: v1
kind: Secret
metadata:
  annotations:
    secretgen.carvel.dev/image-pull-secret: ""
  name: secret
  namespace: sg-test4
type: kubernetes.io/dockerconfigjson
data:
  .dockerconfigjson: "e30K"
---
#! NOTE secrets are not exported to namespace 5 so its placeholder should remain at default.
apiVersion: v1
kind: Secret
metadata:
  annotations:
    secretgen.carvel.dev/image-pull-secret: ""
  name: secret
  namespace: sg-test5
type: kubernetes.io/dockerconfigjson
data:
  .dockerconfigjson: "e30K"
`
	yaml1ExpectedContents := `{"auths":{"www.sg-test1-server-a.com":{"username":"sg-test1-secret-user-a","password":"sg-test1-secret-password-a","auth":"sgtest1-notbase64-a"},"www.sg-test1-server-b.com":{"username":"sg-test1-secret-user-b","password":"sg-test1-secret-password-b","auth":"sgtest1-notbase64-b"}}}`

	yaml2 := `
---
apiVersion: v1
kind: Secret
metadata:
  name: secret-kappa
  namespace: sg-test1
type: kubernetes.io/dockerconfigjson
stringData:
  .dockerconfigjson: |
    {
      "auths": {
        "www.sg-test1-server-a.com": {
          "username": "sg-test1-secret-user-a2",
          "password": "sg-test1-secret-password-a2",
          "auth": "sgtest1-notbase64-a2"
        },
        "www.sg-test1-server-b.com": {
          "username": "sg-test1-secret-user-b",
          "password": "sg-test1-secret-password-b",
          "auth": "sgtest1-notbase64-b"
        }
      }
    }
`
	yaml2ExpectedContents := `{"auths":{"www.sg-test1-server-a.com":{"username":"sg-test1-secret-user-a2","password":"sg-test1-secret-password-a2","auth":"sgtest1-notbase64-a2"},"www.sg-test1-server-b.com":{"username":"sg-test1-secret-user-b","password":"sg-test1-secret-password-b","auth":"sgtest1-notbase64-b"}}}`

	yaml3 := `
---
apiVersion: secretgen.carvel.dev/v1alpha1
kind: SecretExport
metadata:
  name: secret-mu
  namespace: sg-test1
spec:
  toNamespaces:
  - sg-test2
  - sg-test4
`
	yaml4 := `
---
apiVersion: secretgen.carvel.dev/v1alpha1
kind: SecretExport
metadata:
  name: secret-mu
  namespace: sg-test1
spec:
  toNamespaces:
  - sg-test4
`

	yaml3MuOnlyExpectedContents := `{"auths":{"www.sg-test1-server-c.com":{"username":"sg-test1-secret-user-c","password":"sg-test1-secret-password-c","auth":"sgtest1-notbase64-c"},"www.sg-test1-server-d.com":{"username":"sg-test1-secret-user-d","password":"sg-test1-secret-password-d","auth":"sgtest1-notbase64-d"}}}`
	yaml3AllExpectedContents := `{"auths":{"www.sg-test1-server-a.com":{"username":"sg-test1-secret-user-a2","password":"sg-test1-secret-password-a2","auth":"sgtest1-notbase64-a2"},"www.sg-test1-server-b.com":{"username":"sg-test1-secret-user-b","password":"sg-test1-secret-password-b","auth":"sgtest1-notbase64-b"},"www.sg-test1-server-c.com":{"username":"sg-test1-secret-user-c","password":"sg-test1-secret-password-c","auth":"sgtest1-notbase64-c"},"www.sg-test1-server-d.com":{"username":"sg-test1-secret-user-d","password":"sg-test1-secret-password-d","auth":"sgtest1-notbase64-d"}}}`

	emptyAuthsContents := `{"auths":{}}`

	name := "test-placeholder-export-successful"
	cleanUp := func() {
		kapp.RunWithOpts([]string{"delete", "-a", name}, RunOpts{AllowError: true})
	}

	cleanUp()
	defer cleanUp()

	logger.Section("Initial Deploy", func() {
		kapp.RunWithOpts([]string{"deploy", "-f", "-", "-a", name},
			RunOpts{StdinReader: strings.NewReader(yaml1)})
	})

	nsToExpected := map[string]string{
		"sg-test2": yaml1ExpectedContents,
		"sg-test3": yaml1ExpectedContents,
		"sg-test4": emptyAuthsContents,
		"sg-test5": emptyAuthsContents,
	}

	logger.Section("Check placeholder secrets were populated appropriately", func() {
		assertSecretsConvergeToExpected(t, nsToExpected, kubectl)
	})

	logger.Section("Update initial secret should update downstream placeholders", func() {
		kapp.RunWithOpts([]string{"deploy", "-f", "-", "-a", name, "-p"},
			RunOpts{StdinReader: strings.NewReader(yaml2)})

		nsToExpected["sg-test2"] = yaml2ExpectedContents
		nsToExpected["sg-test3"] = yaml2ExpectedContents
		assertSecretsConvergeToExpected(t, nsToExpected, kubectl)
	})

	logger.Section("creating new SecretExport should populate data from corresponding Secret into placeholders", func() {
		kapp.RunWithOpts([]string{"deploy", "-f", "-", "-a", name, "-p"},
			RunOpts{StdinReader: strings.NewReader(yaml3)})

		nsToExpected["sg-test2"] = yaml3AllExpectedContents
		nsToExpected["sg-test4"] = yaml3MuOnlyExpectedContents
		assertSecretsConvergeToExpected(t, nsToExpected, kubectl)
	})

	logger.Section("Update SecretExport to remove ns from list should remove data from placeholder", func() {
		kapp.RunWithOpts([]string{"deploy", "-f", "-", "-a", name, "-p"},
			RunOpts{StdinReader: strings.NewReader(yaml4)})

		nsToExpected["sg-test2"] = yaml2ExpectedContents
		assertSecretsConvergeToExpected(t, nsToExpected, kubectl)
	})

	logger.Section("Delete Secret resource but leave SecretExport should remove data from placeholder", func() {
		kubectl.RunWithOpts([]string{"delete", "secret", "secret-mu", "-n", "sg-test1"},
			RunOpts{NoNamespace: true})

		nsToExpected["sg-test4"] = emptyAuthsContents
		assertSecretsConvergeToExpected(t, nsToExpected, kubectl)
	})

	logger.Section("Delete SecretExport of remaining Secret", func() {
		kubectl.RunWithOpts([]string{"delete", "secretexport", "secret-kappa", "-n", "sg-test1"},
			RunOpts{NoNamespace: true})

		nsToExpected["sg-test2"] = emptyAuthsContents
		nsToExpected["sg-test3"] = emptyAuthsContents
		assertSecretsConvergeToExpected(t, nsToExpected, kubectl)
	})
}

func TestPlaceholderNamespaceExclusion(t *testing.T) {
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
  annotations:
    secretgen.carvel.dev/excluded-from-wildcard-matching: ""
---
apiVersion: v1
kind: Namespace
metadata:
  name: sg-test3
  annotations:
    secretgen.carvel.dev/excluded-from-wildcard-matching: ""
---
apiVersion: v1
kind: Secret
metadata:
  name: secret-kappa
  namespace: sg-test1
type: kubernetes.io/dockerconfigjson
stringData:
  .dockerconfigjson: |
    {
      "auths": {
        "www.sg-test1-server-a.com": {
          "username": "sg-test1-secret-user-a",
          "password": "sg-test1-secret-password-a",
          "auth": "sgtest1-notbase64-a"
        },
        "www.sg-test1-server-b.com": {
          "username": "sg-test1-secret-user-b",
          "password": "sg-test1-secret-password-b",
          "auth": "sgtest1-notbase64-b"
        }
      }
    }
---
apiVersion: secretgen.carvel.dev/v1alpha1
kind: SecretExport
metadata:
  name: secret-kappa
  namespace: sg-test1
spec:
  toNamespaces:
  - "*"
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
---
`
	yaml1ExpectedContents := `{"auths":{"www.sg-test1-server-a.com":{"username":"sg-test1-secret-user-a","password":"sg-test1-secret-password-a","auth":"sgtest1-notbase64-a"},"www.sg-test1-server-b.com":{"username":"sg-test1-secret-user-b","password":"sg-test1-secret-password-b","auth":"sgtest1-notbase64-b"}}}`
	emptyAuthsContents := `{"auths":{}}`

	name := "test-placeholder-export-successful"
	cleanUp := func() {
		kapp.RunWithOpts([]string{"delete", "-a", name}, RunOpts{AllowError: true})
	}

	cleanUp()
	defer cleanUp()

	logger.Section("Initial Deploy", func() {
		kapp.RunWithOpts([]string{"deploy", "-f", "-", "-a", name},
			RunOpts{StdinReader: strings.NewReader(yaml1)})
	})

	// both ns have the exclusion annotation, but sg-test3 is also named explicitly.
	nsToExpected := map[string]string{
		"sg-test2": emptyAuthsContents,
		"sg-test3": yaml1ExpectedContents,
	}

	logger.Section("Check placeholder secrets were populated appropriately", func() {
		assertSecretsConvergeToExpected(t, nsToExpected, kubectl)
	})

	logger.Section("Removing annotation should populate secret", func() {
		args := []string{"annotate", "namespace", "sg-test2", "secretgen.carvel.dev/excluded-from-wildcard-matching-"}
		_ = kubectl.Run(args)

		nsToExpected["sg-test2"] = yaml1ExpectedContents
		assertSecretsConvergeToExpected(t, nsToExpected, kubectl)
	})

}

func assertSecretsConvergeToExpected(t *testing.T, nsToExpected map[string]string, kubectl Kubectl) {
	for ns, expected := range nsToExpected {
		out := waitUntilSecretInNsPopulated(t, kubectl, ns, "secret", func(secret *corev1.Secret) bool {
			return len(string(secret.Data[".dockerconfigjson"])) == len(expected)
		})

		var secret corev1.Secret
		err := yaml.Unmarshal([]byte(out), &secret)
		require.NoError(t, err)
		assert.Equal(t, expected, string(secret.Data[".dockerconfigjson"]), fmt.Sprintf("failed in namespace %s", ns))
	}
}
