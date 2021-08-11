// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"testing"
	"time"
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
