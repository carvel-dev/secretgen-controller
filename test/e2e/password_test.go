// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"strings"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func assertOpaque(secret corev1.Secret, t *testing.T) {
	if secret.Type != "Opaque" {
		t.Fatalf("Expected Opaque, Wrong type %s", secret.Type)
	}
}

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

		assertOpaque(secret, t)
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

func TestLenghtPasswordTemplate(t *testing.T) {
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
  length: 24
  secretTemplate:
    type: Opaque
    stringData:     
      val: $(value)
      valx2: $(value)$(value)
`

	name := "test-length-password-template"
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

		assertOpaque(secret, t)
		assert.Len(t, secret.Data["val"], 24, "Expect password length is 24, found %d", len(secret.Data["val"]))
		assert.Len(t, secret.Data["valx2"], 2*24, "Expect password length is 48, found %d", len(secret.Data["valx2"]))
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

func TestComplexPasswordTemplate(t *testing.T) {
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
  length: 27
  digits: 2
  uppercaseLetters: 4
  lowercaseLetters: 10
  symbols: 3
  secretTemplate:
    type: Opaque
    stringData:     
      val: $(value)
      
`

	name := "test-complex-password-template"
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

		assertOpaque(secret, t)
		assert.Len(t, secret.Data["val"], 27, "Expect password length is 27, found %d (value %s) ", len(secret.Data["val"]), secret.Data["val"])
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

func TestSymbolPasswordTemplate(t *testing.T) {
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
  length: 3
  symbols: 3
  symbolCharSet: "!$#%"
  secretTemplate:
    type: Opaque
    stringData:     
      val: $(value)
      
`

	name := "test-symbol-password-template"
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

		assertOpaque(secret, t)
		assert.Len(t, secret.Data["val"], 3, "Expect password length is 3, found %d", len(secret.Data["val"]))

		symbolSet := "!$#%"
		split := strings.Split(string(secret.Data["val"]), "")

		for _, ch := range split {
			res := strings.Contains(symbolSet, ch)
			if !res {
				t.Fatalf("Found wrong char %s in the generated password, the symbolSet is  %#v", ch, symbolSet)
			}
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
