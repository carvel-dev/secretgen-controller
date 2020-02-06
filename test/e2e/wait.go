package e2e

import (
	"testing"
	"time"
)

func waitForSecret(t *testing.T, kubectl Kubectl, name string) string {
	var lastErr error

	for i := 0; i < 30; i++ {
		var out string
		out, lastErr = kubectl.RunWithOpts([]string{"get", "secret", name, "-o", "yaml"},
			RunOpts{AllowError: true})
		if lastErr == nil {
			return out
		}
		time.Sleep(time.Second)
	}

	t.Fatalf("Expected to find secret '%s' but did not: %s", name, lastErr)
	panic("Unreachable")
}
