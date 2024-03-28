// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"strings"
	"testing"

	"github.com/ghodss/yaml"
	corev1 "k8s.io/api/core/v1"
)

func TestRSAKey(t *testing.T) {
	env := BuildEnv(t)
	logger := Logger{}
	kapp := Kapp{t, env.Namespace, logger}
	kubectl := Kubectl{t, env.Namespace, logger}

	yaml1 := `
---
apiVersion: secretgen.k14s.io/v1alpha1
kind: RSAKey
metadata:
  name: rsa-key
spec: {}
`

	name := "test-rsa-key"
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
		out := waitForSecret(t, kubectl, "rsa-key")

		var secret corev1.Secret

		err := yaml.Unmarshal([]byte(out), &secret)
		if err != nil {
			t.Fatalf("Failed to unmarshal: %s", err)
		}

		if secret.Type != "Opaque" {
			t.Fatalf("Wrong type")
		}
		if len(secret.Data["pub.pem"]) == 0 {
			t.Fatalf("Failed to find rsa-key pub.pem")
		}
		if len(secret.Data["key.pem"]) == 0 {
			t.Fatalf("Failed to find rsa-key key.pem")
		}
	})

	logger.Section("Delete", func() {
		kapp.Run([]string{"delete", "-a", name})

		_, err := kubectl.RunWithOpts([]string{"delete", "secret", "rsa-key"},
			RunOpts{AllowError: true})

		if !strings.Contains(err.Error(), "(NotFound)") {
			t.Fatalf("Expected NotFound error but was: %s", err)
		}
	})
}
