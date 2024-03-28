// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"strings"
	"testing"

	"github.com/ghodss/yaml"
	corev1 "k8s.io/api/core/v1"
)

func TestCertificate(t *testing.T) {
	env := BuildEnv(t)
	logger := Logger{}
	kapp := Kapp{t, env.Namespace, logger}
	kubectl := Kubectl{t, env.Namespace, logger}

	yaml1 := `
apiVersion: secretgen.k14s.io/v1alpha1
kind: Certificate
metadata:
  name: ca-cert
spec:
  isCA: true
---
apiVersion: secretgen.k14s.io/v1alpha1
kind: Certificate
metadata:
  name: inter-ca-cert
spec:
  isCA: true
  caRef:
    name: ca-cert
---
apiVersion: secretgen.k14s.io/v1alpha1
kind: Certificate
metadata:
  name: app1-cert
spec:
  caRef:
    name: inter-ca-cert
  alternativeNames:
  - app1.svc.cluster.local
---
apiVersion: secretgen.k14s.io/v1alpha1
kind: Certificate
metadata:
  name: app2-cert
spec:
  caRef:
    name: inter-ca-cert
  alternativeNames:
  - app2.svc.cluster.local
`

	name := "test-certificate"
	cleanUp := func() {
		kapp.RunWithOpts([]string{"delete", "-a", name}, RunOpts{AllowError: true})
	}

	cleanUp()
	defer cleanUp()

	logger.Section("Deploy", func() {
		kapp.RunWithOpts([]string{"deploy", "-f", "-", "-a", name},
			RunOpts{IntoNs: true, StdinReader: strings.NewReader(yaml1)})
	})

	logger.Section("Check secret", func() {
		out := waitForSecret(t, kubectl, "app2-cert")

		var secret corev1.Secret

		err := yaml.Unmarshal([]byte(out), &secret)
		if err != nil {
			t.Fatalf("Failed to unmarshal: %s", err)
		}

		if secret.Type != "Opaque" {
			t.Fatalf("Wrong type")
		}
		if len(secret.Data["crt.pem"]) == 0 {
			t.Fatalf("Failed to find app2-cert crt.pem")
		}
		if len(secret.Data["key.pem"]) == 0 {
			t.Fatalf("Failed to find app2-cert key.pem")
		}
		if len(secret.Annotations) == 0 {
			t.Fatalf("Failed to find annotations")
		}
		// TODO more cert checking
	})

	logger.Section("Delete", func() {
		kapp.Run([]string{"delete", "-a", name})

		_, err := kubectl.RunWithOpts([]string{"delete", "secret", "app2-cert"},
			RunOpts{AllowError: true})

		if !strings.Contains(err.Error(), "(NotFound)") {
			t.Fatalf("Expected NotFound error but was: %s", err)
		}
	})
}
