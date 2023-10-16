// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"strings"
	"testing"

	"github.com/ghodss/yaml"
	corev1 "k8s.io/api/core/v1"
)

func TestPassword(t *testing.T) {
	env := BuildEnv(t)
	logger := Logger{}
	kapp := Kapp{t, env.Namespace, logger}
	kubectl := Kubectl{t, env.Namespace, logger}

	yaml1 := `
---
apiVersion: secretgen.k14s.io/v1alpha1
kind: Password
metadata:
  name: password
spec: {}
`

	name := "test-password"
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
		out := waitForSecret(t, kubectl, "password")

		var secret corev1.Secret

		err := yaml.Unmarshal([]byte(out), &secret)
		if err != nil {
			t.Fatalf("Failed to unmarshal: %s", err)
		}

		if secret.Type != "kubernetes.io/basic-auth" {
			t.Fatalf("Wrong type")
		}
		if len(secret.Data["password"]) != 40 {
			t.Fatalf("Failed to find password")
		}
	})

	logger.Section("Delete", func() {
		kapp.Run([]string{"delete", "-a", name})

		_, err := kubectl.RunWithOpts([]string{"delete", "secret", "password"},
			RunOpts{AllowError: true})

		if !strings.Contains(err.Error(), "(NotFound)") {
			t.Fatalf("Expected NotFound error but was: %s", err)
		}
	})
}

func TestPasswordTemplate(t *testing.T) {
	env := BuildEnv(t)
	logger := Logger{}
	kapp := Kapp{t, env.Namespace, logger}
	kubectl := Kubectl{t, env.Namespace, logger}

	yaml1 := `
---
apiVersion: secretgen.k14s.io/v1alpha1
kind: Password
metadata:
  name: password
spec:
  secretTemplate:
    type: Opaque
    stringData:
      static: value
      val: $(value)
      valx2: $(value)$(value)
`

	name := "test-password-template"
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
		out := waitForSecret(t, kubectl, "password")

		var secret corev1.Secret

		err := yaml.Unmarshal([]byte(out), &secret)
		if err != nil {
			t.Fatalf("Failed to unmarshal: %s", err)
		}

		if secret.Type != "Opaque" {
			t.Fatalf("Wrong type")
		}
		if string(secret.Data["static"]) != "value" {
			t.Fatalf("Failed to find password")
		}
		if len(secret.Data["val"]) != 40 {
			t.Fatalf("Failed to find password")
		}
		if len(secret.Data["valx2"]) != 80 {
			t.Fatalf("Failed to find password")
		}
	})

	logger.Section("Delete", func() {
		kapp.Run([]string{"delete", "-a", name})

		_, err := kubectl.RunWithOpts([]string{"delete", "secret", "password"},
			RunOpts{AllowError: true})

		if !strings.Contains(err.Error(), "(NotFound)") {
			t.Fatalf("Expected NotFound error but was: %s", err)
		}
	})
}
