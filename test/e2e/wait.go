// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"testing"
	"time"

	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func waitForSecret(t *testing.T, kubectl Kubectl, name string) string {
	return waitForSecretInNs(t, kubectl, "", name)
}

func waitForSecretInNs(t *testing.T, kubectl Kubectl, nsName, name string) string {
	var lastErr error

	args := []string{"get", "secret", name, "-o", "yaml"}
	noNs := false

	if len(nsName) > 0 {
		args = append(args, []string{"-n", nsName}...)
		noNs = true
	}

	for i := 0; i < 30; i++ {
		var out string
		out, lastErr = kubectl.RunWithOpts(args, RunOpts{AllowError: true, NoNamespace: noNs})
		if lastErr == nil {
			return out
		}
		time.Sleep(time.Second)
	}

	t.Fatalf("Expected to find secret '%s' but did not: %s", name, lastErr)
	panic("Unreachable")
}

func waitForSecretNotInNs(kubectl Kubectl, nsName, name string) bool {
	var lastErr error

	args := []string{"get", "secret", name, "-o", "yaml"}
	noNs := false

	if len(nsName) > 0 {
		args = append(args, []string{"-n", nsName}...)
		noNs = true
	}

	for i := 0; i < 30; i++ {
		_, lastErr = kubectl.RunWithOpts(args, RunOpts{AllowError: true, NoNamespace: noNs})
		if lastErr == nil {
			return false
		}
		time.Sleep(time.Second)
	}

	return true
}

func waitUntilSecretInNsPopulated(t *testing.T, kubectl Kubectl, nsName, name string, checkFun func(*corev1.Secret) bool) string {
	var secret corev1.Secret

	for i := 0; i < 10; i++ {
		out := waitForSecretInNs(t, kubectl, nsName, name)
		err := yaml.Unmarshal([]byte(out), &secret)
		require.NoError(t, err)

		if checkFun(&secret) {
			return out
		}
		time.Sleep(time.Second)
	}

	t.Fatalf("Timed out before Secret '%s' satisfied condition in Namespace '%s'; current data: %s", name, nsName, string(secret.Data[".dockerconfigjson"]))
	panic("Unreachable")
}
