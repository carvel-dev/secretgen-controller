// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"strings"
	"testing"

	"github.com/ghodss/yaml"
	corev1 "k8s.io/api/core/v1"
)

func TestSSHKey(t *testing.T) {
	env := BuildEnv(t)
	logger := Logger{}
	kapp := Kapp{t, env.Namespace, logger}
	kubectl := Kubectl{t, env.Namespace, logger}

	yaml1 := `
---
apiVersion: secretgen.k14s.io/v1alpha1
kind: SSHKey
metadata:
  name: ssh-key
spec: {}
`

	name := "test-ssh-key"
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
		out := waitForSecret(t, kubectl, "ssh-key")

		var secret corev1.Secret

		err := yaml.Unmarshal([]byte(out), &secret)
		if err != nil {
			t.Fatalf("Failed to unmarshal: %s", err)
		}

		if secret.Type != "kubernetes.io/ssh-auth" {
			t.Fatalf("Wrong type")
		}
		if len(secret.Data["ssh-privatekey"]) == 0 {
			t.Fatalf("Failed to find ssh-key pub.pem")
		}
		if len(secret.Data["ssh-authorizedkey"]) == 0 {
			t.Fatalf("Failed to find ssh-key key.pem")
		}
	})

	logger.Section("Delete", func() {
		kapp.Run([]string{"delete", "-a", name})

		_, err := kubectl.RunWithOpts([]string{"delete", "secret", "ssh-key"},
			RunOpts{AllowError: true})

		if !strings.Contains(err.Error(), "(NotFound)") {
			t.Fatalf("Expected NotFound error but was: %s", err)
		}
	})
}

func TestSSHKeyTemplate(t *testing.T) {
	env := BuildEnv(t)
	logger := Logger{}
	kapp := Kapp{t, env.Namespace, logger}
	kubectl := Kubectl{t, env.Namespace, logger}

	yaml1 := `
---
apiVersion: secretgen.k14s.io/v1alpha1
kind: SSHKey
metadata:
  name: ssh-key
spec:
  secretTemplate:
    type: Opaque
    stringData:
      pk: $(privateKey)
      ak: $(authorizedKey)
`

	name := "test-ssh-key-template"
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
		out := waitForSecret(t, kubectl, "ssh-key")

		var secret corev1.Secret

		err := yaml.Unmarshal([]byte(out), &secret)
		if err != nil {
			t.Fatalf("Failed to unmarshal: %s", err)
		}

		if secret.Type != "Opaque" {
			t.Fatalf("Wrong type")
		}
		if len(secret.Data["pk"]) == 0 {
			t.Fatalf("Failed to find ssh-key pub.pem")
		}
		if len(secret.Data["ak"]) == 0 {
			t.Fatalf("Failed to find ssh-key key.pem")
		}
	})

	logger.Section("Delete", func() {
		kapp.Run([]string{"delete", "-a", name})

		_, err := kubectl.RunWithOpts([]string{"delete", "secret", "ssh-key"},
			RunOpts{AllowError: true})

		if !strings.Contains(err.Error(), "(NotFound)") {
			t.Fatalf("Expected NotFound error but was: %s", err)
		}
	})
}
